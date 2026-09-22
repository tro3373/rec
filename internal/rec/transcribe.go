package rec

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

// preflight verifies the prerequisites before recording starts,
// so a whole meeting is not recorded only to fail afterwards.
func preflight(e engine, opts Options) error {
	if err := preflightEngine(e, opts); err != nil {
		return err
	}
	if err := preflightPrompts(e); err != nil {
		return err
	}
	if err := requireCmd(minutesArgv(opts.MinutesCmd)[0], "the minutes command"); err != nil {
		return err
	}
	if opts.PostCmd == "" {
		return nil
	}
	return requireCmd(strings.Fields(opts.PostCmd)[0], "the post command")
}

// requireCmd fails unless bin can be run.
func requireCmd(bin, role string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("cannot find %s, %s: %w", bin, role, err)
	}
	return nil
}

// preflightPrompts renders the prompts this run will use, so that a broken
// custom prompt shows up now rather than after the meeting.
func preflightPrompts(e engine) error {
	if _, err := minutesPrompt(); err != nil {
		return err
	}
	if e != engineGemini {
		return nil
	}
	_, err := renderPrompt(promptTranscribe, transcribePromptData{Speaker: speakerSelf})
	return err
}

// preflightEngine verifies what the transcription engine needs.
func preflightEngine(e engine, opts Options) error {
	if e == engineGemini {
		if opts.GeminiAPIKey == "" {
			return errors.New("GEMINI_API_KEY is not set")
		}
		return nil
	}
	if _, err := whisperModel(opts.WhisperModel); err != nil {
		return err
	}
	_, err := vadModel(opts.VADModel)
	return err
}

// repeatLimit is how many identical segments in a row survive. Whisper collapses
// into a phrase learned from its training data when it hears no speech and then
// repeats it for the whole stretch; real speech rarely repeats more than twice.
const repeatLimit = 2

// transcribeAll transcribes every track and returns results in track order.
func transcribeAll(ctx context.Context, e engine, ts []track, opts Options) ([][]segment, error) {
	all, err := transcribe(ctx, e, ts, opts)
	if err != nil {
		return nil, err
	}
	for i, segs := range all {
		all[i] = dropRepeats(segs)
	}
	return all, nil
}

// transcribe dispatches to the engine.
func transcribe(ctx context.Context, e engine, ts []track, opts Options) ([][]segment, error) {
	if e == engineGemini {
		return transcribeGemini(ctx, ts, opts.GeminiModel, opts.GeminiAPIKey)
	}
	return transcribeWhisper(ts, opts)
}

// dropRepeats keeps at most repeatLimit identical segments in a row.
func dropRepeats(segs []segment) []segment {
	out := make([]segment, 0, len(segs))
	run := 0
	for i, s := range segs {
		if i > 0 && s.Text == segs[i-1].Text {
			run++
		} else {
			run = 0
		}
		if run < repeatLimit {
			out = append(out, s)
		}
	}
	return out
}
