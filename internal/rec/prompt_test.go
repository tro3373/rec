package rec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPrompt(t *testing.T) {
	data := transcribePromptData{Speaker: "自分"}
	tests := []struct {
		name    string
		custom  string // Written to the user's prompt dir unless empty.
		want    string
		wantErr bool
	}{
		{name: "上書きが無い場合_組み込みを使うこと", want: "この音声は 自分 の発言です。"},
		{name: "上書きがある場合_そちらを使うこと", custom: "Transcribe {{.Speaker}}.\n", want: "Transcribe 自分."},
		{name: "上書きが壊れている場合_エラーになること", custom: "{{.Speaker", wantErr: true},
		{name: "未知の変数を使う場合_エラーになること", custom: "{{.Nope}}", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", cfg)
			if tt.custom != "" {
				dir := filepath.Join(cfg, "rec", "prompts")
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, promptTranscribe), []byte(tt.custom), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := renderPrompt(promptTranscribe, data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("renderPrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("renderPrompt() = %q, want prefix %q", got, tt.want)
			}
		})
	}
}
