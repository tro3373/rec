package rec

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// whisperLang fixes the transcription language. The minutes and the
	// speaker labels assume Japanese as well.
	whisperLang = "ja"
	// blankAudio is the literal whisper emits for silent stretches.
	blankAudio = "[BLANK_AUDIO]"
	// vadModelName is the silero model whisper.cpp loads for --vad.
	vadModelName = "ggml-silero-v5.1.2.bin"
	// whisperModelName is the transcription model used when nothing is configured.
	whisperModelName = "ggml-large-v3-turbo.bin"
)

// whisperModel resolves the transcription model path and verifies that it exists.
func whisperModel(configured string) (string, error) {
	return resolveModel(configured, os.Getenv("REC_WHISPER_MODEL"), whisperModelName,
		"-model or REC_WHISPER_MODEL", "ggerganov/whisper.cpp")
}

// vadModel resolves the silero VAD model path and verifies that it exists.
// VAD keeps silence away from whisper, which otherwise hallucinates a learned
// phrase and repeats it for the whole silent stretch.
func vadModel(configured string) (string, error) {
	return resolveModel(configured, os.Getenv("REC_VAD_MODEL"), vadModelName,
		"-vad-model or REC_VAD_MODEL", "ggml-org/whisper-vad")
}

// resolveModel picks the first configured path, falling back to the cache dir,
// and turns a missing file into a message that says how to fetch it.
func resolveModel(configured, env, name, setHint, repo string) (string, error) {
	path := cmp.Or(configured, env, cacheModel(name))
	if path == "" {
		return "", fmt.Errorf("cannot locate %s: set %s", name, setHint)
	}
	// #nosec G703 -- path points at the model the user chose; nothing but a stat happens.
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf(
			"model not found: %s\n"+
				"  get it with: mkdir -p %s && curl -L -o %s https://huggingface.co/%s/resolve/main/%s",
			path, filepath.Dir(path), path, repo, filepath.Base(path))
	}
	return path, nil
}

// cacheModel is where a model lives when nothing is configured.
func cacheModel(name string) string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cache, "whisper.cpp", name)
}

// transcribeWhisper transcribes every track with the local whisper-cli.
// All tracks go through a single process so the 1.6GB model is loaded once.
// whisper-cli pairs repeated -of flags with the input files positionally.
func transcribeWhisper(ts []track, opts Options) ([][]segment, error) {
	model, err := whisperModel(opts.WhisperModel)
	if err != nil {
		return nil, err
	}
	vad, err := vadModel(opts.VADModel)
	if err != nil {
		return nil, err
	}
	bases := make([]string, len(ts))
	args := []string{"-m", model, "-l", whisperLang, "-oj", "--vad", "-vm", vad}
	for i, t := range ts {
		bases[i] = stripExt(t.Path)
		args = append(args, "-of", bases[i])
	}
	for _, t := range ts {
		args = append(args, t.Path)
	}

	// #nosec G204 -- args are the model path and destinations we built ourselves.
	cmd := exec.Command("whisper-cli", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cannot run whisper-cli: %w", err)
	}

	all := make([][]segment, len(ts))
	for i, base := range bases {
		// #nosec G304 -- base is a path we built ourselves, not external input.
		b, err := os.ReadFile(base + ".json")
		if err != nil {
			return nil, fmt.Errorf("cannot read the transcript of %s: %w", ts[i].Speaker, err)
		}
		all[i], err = parseWhisperJSON(b)
		if err != nil {
			return nil, err
		}
	}
	return all, nil
}

// parseWhisperJSON extracts segments from the -oj output of whisper-cli.
func parseWhisperJSON(b []byte) ([]segment, error) {
	var out struct {
		Transcription []struct {
			Offsets struct {
				From int `json:"from"`
			} `json:"offsets"`
			Text string `json:"text"`
		} `json:"transcription"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("cannot parse whisper JSON output: %w", err)
	}
	segments := make([]segment, 0, len(out.Transcription))
	for _, t := range out.Transcription {
		text := strings.TrimSpace(t.Text)
		if text == "" || text == blankAudio {
			continue
		}
		segments = append(segments, segment{StartMs: t.Offsets.From, Text: text})
	}
	return segments, nil
}
