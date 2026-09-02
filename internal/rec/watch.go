package rec

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"
)

// watchedApps are the process names that mean a call is in progress.
// Measured with pactl: a Slack huddle reports slack, Zoom reports zoom, and
// Google Meet reports chrome because it runs inside the browser. Chrome
// therefore also fires for any other page that opens the microphone.
var watchedApps = []string{"slack", "zoom", "chrome"}

const (
	// callQuietFor is how long every stream must stay gone before a call is over.
	// The apps tear a stream down and open the next one within the same second,
	// so a bare disappearance is not the end of anything.
	callQuietFor = 10 * time.Second
	// callPollInterval bounds how late the quiet period is noticed. Starting a
	// call is driven by pactl events instead, so nothing is missed at the front.
	callPollInterval = time.Second
)

// callStreams counts the recording streams owned by the watched apps.
type callStreams struct {
	Total  int // Streams that exist, running or paused.
	Active int // Streams that are actually running.
}

// parseCallStreams counts the watched apps' streams in pactl source-output JSON.
func parseCallStreams(data []byte, apps []string) (callStreams, error) {
	var outs []struct {
		Corked     bool `json:"corked"`
		Properties struct {
			Binary string `json:"application.process.binary"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &outs); err != nil {
		return callStreams{}, fmt.Errorf("cannot parse pactl list source-outputs JSON: %w", err)
	}
	var s callStreams
	for _, o := range outs {
		if !slices.Contains(apps, o.Properties.Binary) {
			continue
		}
		s.Total++
		if !o.Corked {
			s.Active++
		}
	}
	return s, nil
}

// detectCallStreams asks pactl which watched apps hold the microphone.
func detectCallStreams(apps []string) (callStreams, error) {
	out, err := exec.Command("pactl", "-f", "json", "list", "source-outputs").Output()
	if err != nil {
		return callStreams{}, fmt.Errorf("cannot run pactl list source-outputs: %w", err)
	}
	return parseCallStreams(out, apps)
}

// callEvent is what one observation changed.
type callEvent int

const (
	callNone callEvent = iota
	callStarted
	callEnded
)

// callWatcher turns a series of stream counts into call boundaries.
type callWatcher struct {
	quietFor time.Duration
	inCall   bool
	goneAt   time.Time // When the last stream disappeared. Zero while streams exist.
}

// step folds one observation into the watcher and reports the transition.
func (w *callWatcher) step(s callStreams, now time.Time) callEvent {
	if !w.inCall {
		// A paused stream is an app preparing itself, not a call.
		if s.Active == 0 {
			return callNone
		}
		w.inCall = true
		w.goneAt = time.Time{}
		return callStarted
	}
	// Paused streams still count here: muting yourself must not end the call.
	if s.Total > 0 {
		w.goneAt = time.Time{}
		return callNone
	}
	if w.goneAt.IsZero() {
		w.goneAt = now
		return callNone
	}
	if now.Sub(w.goneAt) < w.quietFor {
		return callNone
	}
	w.inCall = false
	w.goneAt = time.Time{}
	return callEnded
}

// subscribeStreams wakes the caller whenever a recording stream appears or goes.
// The channel is closed once ctx is done or pactl exits.
func subscribeStreams(ctx context.Context) (<-chan struct{}, error) {
	cmd := exec.CommandContext(ctx, "pactl", "subscribe")
	// The event lines are human readable and therefore localized.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cannot pipe pactl subscribe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cannot run pactl subscribe: %w", err)
	}
	ch := make(chan struct{}, 1)
	go func() {
		defer close(ch)
		defer func() { _ = cmd.Wait() }()
		sc := bufio.NewScanner(out)
		for sc.Scan() {
			if !strings.Contains(sc.Text(), "on source-output") {
				continue
			}
			select {
			case ch <- struct{}{}:
			default: // A pending wake-up already covers this event.
			}
		}
	}()
	return ch, nil
}

// startRecording records one call in the background and returns its stop switch.
// Transcription and the minutes run after the recording, so the call is over
// long before the goroutine is.
func startRecording(ctx context.Context, e engine, opts Options) context.CancelFunc {
	recCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		err := runOnce(ctx, e, opts, "call detected, recording", func(ts []track) error {
			return record(recCtx, ts)
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
	}()
	return cancel
}

// Watch records every call it detects, until ctx is done.
func Watch(ctx context.Context, opts Options) error {
	e, err := parseEngine(opts.Engine)
	if err != nil {
		return err
	}
	// Fail at startup rather than when the first call comes in.
	if err := preflight(e, opts); err != nil {
		return err
	}
	events, err := subscribeStreams(ctx)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(callPollInterval)
	defer ticker.Stop()

	fmt.Fprintf(os.Stderr, "watching for calls from %s\n", strings.Join(watchedApps, ", "))
	w := &callWatcher{quietFor: callQuietFor}
	var stopRecording context.CancelFunc
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-events:
			if !ok {
				// pactl is gone. The ticker still drives the polling below.
				fmt.Fprintln(os.Stderr, "warning: pactl subscribe stopped, falling back to polling")
				events = nil
			}
		case <-ticker.C:
		}
		s, err := detectCallStreams(watchedApps)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warning:", err)
			continue
		}
		switch w.step(s, time.Now()) {
		case callStarted:
			stopRecording = startRecording(ctx, e, opts)
		case callEnded:
			stopRecording()
		case callNone:
		}
	}
}
