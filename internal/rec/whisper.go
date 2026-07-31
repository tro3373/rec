package rec

import (
	"cmp"
	"encoding/json"
	"errors"
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
)

// whisperModel resolves the model path and verifies that it exists.
func whisperModel(configured string) (string, error) {
	path := cmp.Or(configured, os.Getenv("REC_WHISPER_MODEL"), defaultWhisperModel())
	if path == "" {
		return "", errors.New("cannot locate a whisper model: set -model or REC_WHISPER_MODEL")
	}
	// #nosec G703 -- path points at the model the user chose; nothing but a stat happens.
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf(
			"whisper model not found: %s\n"+
				"  get it with: mkdir -p %s && curl -L -o %s https://huggingface.co/ggerganov/whisper.cpp/resolve/main/%s",
			path, filepath.Dir(path), path, filepath.Base(path))
	}
	return path, nil
}

// defaultWhisperModel is where the model lives when nothing is configured.
func defaultWhisperModel() string {
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cache, "whisper.cpp", "ggml-large-v3-turbo.bin")
}

// transcribeWhisper transcribes every track with the local whisper-cli.
// All tracks go through a single process so the 1.6GB model is loaded once.
// whisper-cli pairs repeated -of flags with the input files positionally.
func transcribeWhisper(ts []track, configuredModel string) ([][]segment, error) {
	model, err := whisperModel(configuredModel)
	if err != nil {
		return nil, err
	}
	bases := make([]string, len(ts))
	args := []string{"-m", model, "-l", whisperLang, "-oj"}
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
