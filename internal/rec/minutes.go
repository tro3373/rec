package rec

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// defaultMinutesCmd generates the minutes when no command is configured.
const defaultMinutesCmd = "claude -p"

// minutesPrompt fills the minutes prompt with a real sample of the transcript
// format, so that the line format and the speaker labels have a single owner.
func minutesPrompt() (string, error) {
	sample := strings.TrimSuffix(
		renderTranscript([]line{{Speaker: speakerSelf, Text: "発言"}}), "\n")
	return renderPrompt(promptMinutes, minutesPromptData{Format: sample, Self: speakerSelf, Other: speakerOther})
}

// minutesArgv splits the configured command line. It is split on spaces only,
// so anything that needs quoting belongs in a wrapper script.
func minutesArgv(cmdline string) []string {
	return strings.Fields(cmp.Or(strings.TrimSpace(cmdline), defaultMinutesCmd))
}

// generateMinutes pipes the prompt and the transcript through the minutes
// command. Both go through stdin, so the command needs no prompt argument.
func generateMinutes(cmdline, transcript string) ([]byte, error) {
	prompt, err := minutesPrompt()
	if err != nil {
		return nil, err
	}
	argv := minutesArgv(cmdline)
	// #nosec G204 -- the command line is the user's own configuration.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = strings.NewReader(prompt + "\n---\n" + transcript)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot generate the minutes with %s: %w", argv[0], err)
	}
	return out, nil
}
