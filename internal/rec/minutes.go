package rec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// minutesPromptTmpl is sent to claude, so it is written in the output language.
// The transcript line format and the speaker labels are filled in from
// renderTranscript so that they have a single owner.
const minutesPromptTmpl = `標準入力は会議の文字起こしです。
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

// generateMinutes pipes the transcript through claude to produce the minutes.
func generateMinutes(transcript string) ([]byte, error) {
	// #nosec G204 -- the argument is a fixed prompt built from package constants.
	cmd := exec.Command("claude", "-p", minutesPrompt())
	cmd.Stdin = strings.NewReader(transcript)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("cannot generate the minutes with claude: %w", err)
	}
	return out, nil
}
