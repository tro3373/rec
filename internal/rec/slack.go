package rec

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// slkBin owns the Slack token, the channel and the upload dance already.
	slkBin = "slk"
	// slkNoAutoUpload keeps slk from turning a long message into a file on its
	// own, which would leave the post itself empty.
	slkNoAutoUpload = "--no-auto-upload"
	// slackFallbackText stands in when the minutes carry no summary section.
	slackFallbackText = "議事録ができました"
)

// summarySection returns the summary part of the minutes, its heading included.
// The heading level is whatever claude produced, so only the text is matched.
func summarySection(minutes string) string {
	lines := strings.Split(minutes, "\n")
	start := -1
	for i, l := range lines {
		if !strings.HasPrefix(l, "#") {
			continue
		}
		if start >= 0 {
			return strings.TrimSpace(strings.Join(lines[start:i], "\n"))
		}
		if strings.Contains(l, "概要") {
			start = i
		}
	}
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:], "\n"))
}

// runSlk posts one message through slk, which reads it from stdin.
// Passing it as an argument does not work: slk ignores argv once stdin is not a tty.
func runSlk(opts Options, stdin string, args ...string) error {
	full := append([]string{slkNoAutoUpload}, args...)
	if opts.SlackChannel != "" {
		full = append(full, "-c", opts.SlackChannel)
	}
	// #nosec G204 -- the arguments are our own flags and the configured channel.
	cmd := exec.Command(slkBin, full...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot post to Slack with %s: %w", slkBin, err)
	}
	return nil
}

// postToSlack posts the summary and attaches the whole minutes as a file.
func postToSlack(opts Options, minutesPath string, minutes []byte) error {
	if err := runSlk(opts, cmp.Or(summarySection(string(minutes)), slackFallbackText)); err != nil {
		return err
	}
	return runSlk(opts, minutesPath, "-f")
}
