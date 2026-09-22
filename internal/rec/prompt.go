package rec

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// defaultPrompts are the built-in prompts, written in the output language.
// A file of the same name under promptDir replaces one of them.
//
//go:embed prompts/*.md
var defaultPrompts embed.FS

const (
	promptMinutes    = "minutes.md"    // Filled with minutesPromptData.
	promptTranscribe = "transcribe.md" // Filled with transcribePromptData.
)

// minutesPromptData is what the minutes prompt can refer to.
type minutesPromptData struct {
	Format string // One sample transcript line.
	Self   string // Speaker label of your own voice.
	Other  string // Speaker label of the other side.
}

// transcribePromptData is what the Gemini transcription prompt can refer to.
type transcribePromptData struct {
	Speaker string // Speaker label of the track being transcribed.
}

// promptDir is where the user's own prompts live.
func promptDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve the config directory: %w", err)
	}
	return filepath.Join(dir, "rec", "prompts"), nil
}

// readPrompt returns the user's copy of a prompt, or the built-in one.
func readPrompt(name string) ([]byte, error) {
	dir, err := promptDir()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- a fixed name under the user's config dir.
	if err == nil {
		return b, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot read the prompt: %w", err)
	}
	return defaultPrompts.ReadFile("prompts/" + name)
}

// renderPrompt fills a prompt template with data.
func renderPrompt(name string, data any) (string, error) {
	src, err := readPrompt(name)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Parse(string(src))
	if err != nil {
		return "", fmt.Errorf("cannot parse the prompt %s: %w", name, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("cannot fill the prompt %s: %w", name, err)
	}
	return strings.TrimSpace(sb.String()), nil
}
