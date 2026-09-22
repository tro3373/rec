package main

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnv(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "コメントと空行がある場合_読み飛ばすこと",
			in:   "# comment\n\nREC_ENGINE=whisper\n  # indented\n",
			want: map[string]string{"REC_ENGINE": "whisper"},
		},
		{
			name: "引用符で囲んだ場合_外すこと",
			in:   "A=\"claude -p\"\nB='#rec'\nC=\"unbalanced'\n",
			want: map[string]string{"A": "claude -p", "B": "#rec", "C": "\"unbalanced'"},
		},
		{
			name: "値に等号を含む場合_最初の等号で分けること",
			in:   "A = x=y\nB=\n",
			want: map[string]string{"A": "x=y", "B": ""},
		},
		{name: "等号が無い場合_エラーになること", in: "REC_ENGINE\n", wantErr: true},
		{name: "キーが空の場合_エラーになること", in: "=x\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEnv(strings.NewReader(tt.in))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("parseEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	tests := []struct {
		name  string
		file  string // Written unless empty.
		shell string // REC_ENGINE in the environment. Empty means unset.
		want  string // REC_ENGINE after loading.
	}{
		{name: "ファイルだけにある場合_ファイルの値になること", file: "REC_ENGINE=gemini\n", want: "gemini"},
		{name: "シェルにもある場合_シェルの値が勝つこと", file: "REC_ENGINE=gemini\n", shell: "whisper", want: "whisper"},
		{name: "ファイルが無い場合_何もしないこと", shell: "whisper", want: "whisper"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "env")
			if tt.file != "" {
				if err := os.WriteFile(path, []byte(tt.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("REC_ENGINE", tt.shell)
			if tt.shell == "" {
				os.Unsetenv("REC_ENGINE")
			}
			if err := loadEnvFile(path); err != nil {
				t.Fatalf("loadEnvFile() error = %v", err)
			}
			if got := os.Getenv("REC_ENGINE"); got != tt.want {
				t.Errorf("REC_ENGINE = %q, want %q", got, tt.want)
			}
		})
	}
}
