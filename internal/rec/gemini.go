package rec

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	geminiEndpoint  = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent"
	geminiAudioMime = "audio/ogg"
	// geminiInlineLimit stays under the 20MB request limit with headroom.
	geminiInlineLimit = 18 << 20
)

// geminiPromptTmpl is sent to the model, so it is written in the output language.
const geminiPromptTmpl = `この音声は %s の発言です。日本語で文字起こししてください。
発言のまとまりごとに、音声先頭からの開始時刻 (HH:MM:SS) と本文を返してください。
相槌や無音は含めないでください。`

// geminiSegment is one chunk returned by the model.
// The field names must match the responseSchema in geminiGenConfig.
type geminiSegment struct {
	Start string `json:"start"`
	Text  string `json:"text"`
}

// geminiGenConfig forces the model to answer with an array of geminiSegment.
const geminiGenConfig = `{
  "responseMimeType": "application/json",
  "responseSchema": {
    "type": "ARRAY",
    "items": {
      "type": "OBJECT",
      "properties": {
        "start": {"type": "STRING"},
        "text": {"type": "STRING"}
      },
      "required": ["start", "text"]
    }
  }
}`

// transcribeGemini transcribes every track through the Gemini API.
func transcribeGemini(ctx context.Context, ts []track, model, apiKey string) ([][]segment, error) {
	all := make([][]segment, len(ts))
	for i, t := range ts {
		segments, err := transcribeGeminiTrack(ctx, t, model, apiKey)
		if err != nil {
			return nil, err
		}
		all[i] = segments
	}
	return all, nil
}

// transcribeGeminiTrack sends one track to Gemini.
// A raw wav is too large to inline, so it is compressed to opus first.
func transcribeGeminiTrack(ctx context.Context, t track, model, apiKey string) ([]segment, error) {
	audio, err := compressForUpload(t.Path)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(geminiRequest(t.Speaker, audio))
	if err != nil {
		return nil, fmt.Errorf("cannot build the Gemini request: %w", err)
	}
	raw, err := postGemini(ctx, fmt.Sprintf(geminiEndpoint, model), apiKey, body)
	if err != nil {
		return nil, fmt.Errorf("cannot transcribe %s: %w", t.Speaker, err)
	}
	return parseGeminiJSON(raw)
}

// compressForUpload converts the wav to opus and reads it back.
// The intermediate file is removed.
func compressForUpload(wav string) ([]byte, error) {
	ogg := stripExt(wav) + ".ogg"
	cmd := ffmpegCmd("-i", wav, "-c:a", "libopus", "-b:a", "16k", "-y", ogg)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cannot convert %s to opus: %w", wav, err)
	}
	defer os.Remove(ogg)

	info, err := os.Stat(ogg)
	if err != nil {
		return nil, fmt.Errorf("cannot stat the opus output: %w", err)
	}
	if info.Size() > geminiInlineLimit {
		return nil, fmt.Errorf(
			"audio is too large for Gemini (%d bytes); split the meeting or use whisper", info.Size())
	}
	// #nosec G304 -- ogg is a path we built ourselves, not external input.
	return os.ReadFile(ogg)
}

// geminiRequest builds a request that returns one track as segment JSON.
func geminiRequest(speaker string, audio []byte) map[string]any {
	return map[string]any{
		"contents": []any{map[string]any{
			"parts": []any{
				map[string]any{"text": fmt.Sprintf(geminiPromptTmpl, speaker)},
				map[string]any{"inline_data": map[string]any{
					"mime_type": geminiAudioMime,
					"data":      base64.StdEncoding.EncodeToString(audio),
				}},
			},
		}},
		"generationConfig": json.RawMessage(geminiGenConfig),
	}
}

// postGemini calls generateContent and returns the JSON string it produced.
func postGemini(ctx context.Context, url, apiKey string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot create the Gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot call Gemini: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini returned %s", res.Status)
	}

	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("cannot parse the Gemini response: %w", err)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini returned an empty response")
	}
	return []byte(out.Candidates[0].Content.Parts[0].Text), nil
}

// parseGeminiJSON converts the segment JSON produced by Gemini into segments.
func parseGeminiJSON(b []byte) ([]segment, error) {
	var raw []geminiSegment
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("cannot parse Gemini JSON output: %w", err)
	}
	segments := make([]segment, 0, len(raw))
	for _, r := range raw {
		text := strings.TrimSpace(r.Text)
		if text == "" {
			continue
		}
		startMs, err := parseTimestamp(r.Start)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment{StartMs: startMs, Text: text})
	}
	return segments, nil
}

// parseTimestamp converts HH:MM:SS or MM:SS into milliseconds.
func parseTimestamp(s string) (int, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, fmt.Errorf("malformed start time: %q", s)
	}
	sec := 0
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, fmt.Errorf("malformed start time: %q: %w", s, err)
		}
		sec = sec*60 + n
	}
	return sec * 1000, nil
}
