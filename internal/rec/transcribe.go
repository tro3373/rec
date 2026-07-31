package rec

import (
	"context"
	"errors"
	"fmt"
)

// engine selects where transcription runs.
type engine string

const (
	engineWhisper engine = "whisper" // Local whisper-cli.
	engineGemini  engine = "gemini"  // Gemini API.
)

// parseEngine resolves an engine name. An empty string means the default.
func parseEngine(s string) (engine, error) {
	switch engine(s) {
	case "", engineWhisper:
		return engineWhisper, nil
	case engineGemini:
		return engineGemini, nil
	}
	return "", fmt.Errorf("unknown transcription engine: %s (whisper|gemini)", s)
}

// preflight verifies the engine prerequisites before recording starts,
// so a whole meeting is not recorded only to fail afterwards.
func preflight(e engine, opts Options) error {
	if e == engineGemini {
		if opts.GeminiAPIKey == "" {
			return errors.New("GEMINI_API_KEY is not set")
		}
		return nil
	}
	_, err := whisperModel(opts.WhisperModel)
	return err
}

// transcribeAll transcribes every track and returns results in track order.
func transcribeAll(ctx context.Context, e engine, ts []track, opts Options) ([][]segment, error) {
	if e == engineGemini {
		return transcribeGemini(ctx, ts, opts.GeminiModel, opts.GeminiAPIKey)
	}
	return transcribeWhisper(ts, opts.WhisperModel)
}
