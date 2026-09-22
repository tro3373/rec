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

// minutesPromptTmpl is sent to the minutes command, so it is written in the
// output language. The transcript line format and the speaker labels are
// filled in from renderTranscript so that they have a single owner.
const minutesPromptTmpl = `「---」の行より後ろは会議の文字起こしです。
各行は次の形式です。
%s
話者は「%s」か「%s」です。
これを読みやすい議事録の markdown に整形してください。

- 見出し構成: 概要 / 決定事項 / TODO / 議論の流れ
- TODO は担当者 (%s / %s) が分かるように書く
- フィラー (えー、あの 等) と言い直しは落とす
- 文字起こしに無い内容を足さない
- markdown 本文だけを出力し、前置きや説明を付けない`

// minutesPrompt fills the template with a real sample of the output format.
func minutesPrompt() string {
	sample := strings.TrimSuffix(
		renderTranscript([]line{{Speaker: speakerSelf, Text: "発言"}}), "\n")
	return fmt.Sprintf(minutesPromptTmpl, sample, speakerSelf, speakerOther, speakerSelf, speakerOther)
}

// minutesArgv splits the configured command line. It is split on spaces only,
// so anything that needs quoting belongs in a wrapper script.
func minutesArgv(cmdline string) []string {
	return strings.Fields(cmp.Or(strings.TrimSpace(cmdline), defaultMinutesCmd))
}

// generateMinutes pipes the prompt and the transcript through the minutes
// command. Both go through stdin, so the command needs no prompt argument.
func generateMinutes(cmdline, transcript string) ([]byte, error) {
	argv := minutesArgv(cmdline)
	// #nosec G204 -- the command line is the user's own configuration.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = strings.NewReader(minutesPrompt() + "\n---\n" + transcript)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot generate the minutes with %s: %w", argv[0], err)
	}
	return out, nil
}
