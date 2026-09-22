package rec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Options configures a rec run. Reading the environment is left to main.
type Options struct {
	OutRoot      string // Parent directory for the artifacts.
	Engine       string // Transcription engine name. Empty means the default.
	WhisperModel string // Path to the whisper model. Empty means the default.
	VADModel     string // Path to the silero VAD model. Empty means the default.
	GeminiModel  string // Gemini model name.
	GeminiAPIKey string // Gemini API key.
	MinutesCmd   string // Command that turns the prompt and transcript on stdin into minutes. Empty means claude -p.
	PostCmd      string // Command run with the output directory once the minutes exist. Empty means none.
}

// Run records a meeting and produces the minutes.
// Recording continues until Ctrl-C.
func Run(ctx context.Context, opts Options) error {
	e, err := parseEngine(opts.Engine)
	if err != nil {
		return err
	}
	// Check the prerequisites first so a whole meeting is not wasted.
	if err := preflight(e, opts); err != nil {
		return err
	}
	return runOnce(ctx, e, opts, "recording (Ctrl-C to stop)", recordUntilInterrupt)
}

// runOnce records one meeting and produces the minutes.
// The stop condition belongs to rec, which returns once the recording is over.
func runOnce(ctx context.Context, e engine, opts Options, banner string, rec func([]track) error) error {
	dev, err := detectDevices()
	if err != nil {
		return err
	}
	dir := filepath.Join(opts.OutRoot, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("cannot create the output directory: %w", err)
	}

	ts := tracks(dev, dir)
	fmt.Fprintf(os.Stderr, "%s\n  self:   %s\n  other:  %s\n  output: %s\n",
		banner, dev.Self, dev.Other, dir)
	if err := rec(ts); err != nil {
		return err
	}

	// The mixdown is only for listening back, so a failure must not lose the minutes.
	if err := mix(ts, filepath.Join(dir, mixName)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "transcribing with %s\n", e)
	segments, err := transcribeAll(ctx, e, ts, opts)
	if err != nil {
		return err
	}

	transcript := renderTranscript(mergeSegments(segments[0], segments[1]))
	transcriptPath := filepath.Join(dir, "transcript.md")
	if err := os.WriteFile(transcriptPath, []byte(transcript), 0o600); err != nil {
		return fmt.Errorf("cannot save the transcript: %w", err)
	}

	fmt.Fprintln(os.Stderr, "generating the minutes")
	minutes, err := generateMinutes(opts.MinutesCmd, transcript)
	if err != nil {
		return fmt.Errorf("%w\n  the transcript is kept at %s", err, transcriptPath)
	}
	minutesPath := filepath.Join(dir, "minutes.md")
	if err := os.WriteFile(minutesPath, minutes, 0o600); err != nil {
		return fmt.Errorf("cannot save the minutes: %w", err)
	}
	fmt.Fprintf(os.Stderr, "done: %s\n", minutesPath)

	if opts.PostCmd == "" {
		return nil
	}
	// The minutes are already on disk, so a failed post command must not fail the run.
	if err := runPostCmd(opts.PostCmd, dir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	return nil
}
