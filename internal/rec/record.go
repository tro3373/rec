package rec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
)

const ffmpegBin = "ffmpeg"

// ffmpegCommonArgs quiets ffmpeg the same way for every invocation.
var ffmpegCommonArgs = []string{"-hide_banner", "-loglevel", "error", "-nostdin"}

// track is one recorded channel.
type track struct {
	Speaker string // Speaker label used in the transcript.
	Source  string // Pulse source name.
	Path    string // Destination wav path.
}

// tracks builds the recording targets, self first and other second.
func tracks(dev devices, dir string) []track {
	return []track{
		{Speaker: speakerSelf, Source: dev.Self, Path: filepath.Join(dir, "self.wav")},
		{Speaker: speakerOther, Source: dev.Other, Path: filepath.Join(dir, "other.wav")},
	}
}

// ffmpegArgs records one track as 16kHz mono wav, the format whisper expects.
func ffmpegArgs(t track) []string {
	return []string{
		"-f", "pulse", "-i", t.Source,
		"-ac", "1", "-ar", "16000",
		"-y", t.Path,
	}
}

// ffmpegCmd builds an ffmpeg command with the common flags and stderr wired up.
func ffmpegCmd(args ...string) *exec.Cmd {
	// #nosec G204 -- args are device names from pactl and paths we built ourselves.
	cmd := exec.Command(ffmpegBin, slices.Concat(ffmpegCommonArgs, args)...)
	cmd.Stderr = os.Stderr
	return cmd
}

// mixName is the merged wav that sits next to the per-speaker tracks.
const mixName = "mixed.wav"

// mixArgs merges the recorded tracks into one wav for listening back.
// normalize=0 keeps the original levels (amix would otherwise halve both) and
// alimiter absorbs the peaks when everyone talks at once.
func mixArgs(ts []track, path string) []string {
	args := make([]string, 0, len(ts)*2+4)
	for _, t := range ts {
		args = append(args, "-i", t.Path)
	}
	return append(args,
		"-filter_complex", fmt.Sprintf("amix=inputs=%d:duration=longest:normalize=0,alimiter=limit=0.95", len(ts)),
		"-y", path)
}

// mix writes the merged wav. The transcript does not depend on it.
func mix(ts []track, path string) error {
	if err := ffmpegCmd(mixArgs(ts, path)...).Run(); err != nil {
		return fmt.Errorf("cannot mix the tracks into %s: %w", path, err)
	}
	return nil
}

// stripExt drops the extension so sibling file names can be derived.
func stripExt(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

// recordUntilInterrupt records until Ctrl-C.
// The binding is released on return, so Ctrl-C stops the recording only and
// the transcription that follows stays interruptible on its own.
func recordUntilInterrupt(ts []track) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return record(ctx, ts)
}

// record starts every track and keeps recording until ctx is done.
func record(ctx context.Context, ts []track) error {
	cmds := make([]*exec.Cmd, 0, len(ts))
	for _, t := range ts {
		cmd := ffmpegCmd(ffmpegArgs(t)...)
		if err := cmd.Start(); err != nil {
			stopAll(cmds)
			return fmt.Errorf("cannot start recording %s (source=%s): %w", t.Speaker, t.Source, err)
		}
		cmds = append(cmds, cmd)
	}
	<-ctx.Done()
	stopAll(cmds)
	return nil
}

// stopAll sends SIGINT to ffmpeg and waits for the wav files to be finalized.
// ffmpeg exits non-zero when interrupted, so the exit status is ignored.
func stopAll(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	for _, cmd := range cmds {
		_ = cmd.Wait()
	}
}
