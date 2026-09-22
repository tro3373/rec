package rec

import (
	"slices"
	"strings"
	"testing"
)

func TestMinutesArgv(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "未指定の場合_claudeになること", in: "", want: []string{"claude", "-p"}},
		{name: "空白だけの場合_claudeになること", in: "  ", want: []string{"claude", "-p"}},
		{name: "指定した場合_空白で分割されること", in: "ollama run  gemma3", want: []string{"ollama", "run", "gemma3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := minutesArgv(tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("minutesArgv() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateMinutes(t *testing.T) {
	const transcript = "- [00:00:01] 自分: こんにちは\n"
	out, err := generateMinutes("cat", transcript)
	if err != nil {
		t.Fatalf("generateMinutes() error = %v", err)
	}
	if want := minutesPrompt() + "\n---\n" + transcript; string(out) != want {
		t.Errorf("stdin = %q, want %q", out, want)
	}
	if _, err := generateMinutes("false", transcript); err == nil {
		t.Error("generateMinutes() with a failing command: error = nil")
	}
	if strings.Contains(minutesPrompt(), "%") {
		t.Errorf("minutesPrompt() has an unfilled verb: %q", minutesPrompt())
	}
}
