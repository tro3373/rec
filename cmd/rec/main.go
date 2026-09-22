package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tro3373/rec/internal/rec"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

// command picks the entry point, an empty name meaning a single recording.
func command(name string) (func(context.Context, rec.Options) error, error) {
	switch name {
	case "":
		return rec.Run, nil
	case "watch":
		return rec.Watch, nil
	}
	return nil, fmt.Errorf("unknown command %q, expected watch or nothing", name)
}

// splitCommand peels a leading sub-command off the arguments, so that the flags
// can still follow it. The flag package would otherwise stop at the first one.
func splitCommand(args []string) (string, []string) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", args
	}
	return args[0], args[1:]
}

// loadSettings exports the variables of the env file.
func loadSettings() error {
	path, err := envFile()
	if err != nil {
		return err
	}
	return loadEnvFile(path)
}

func main() {
	name, args := splitCommand(os.Args[1:])
	// The flag defaults below read the environment, so the file goes in first.
	if err := loadSettings(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	showVersion := flag.Bool("version", false, "print the version and exit")
	opts := rec.Options{GeminiAPIKey: os.Getenv("GEMINI_API_KEY")}
	flag.StringVar(&opts.OutRoot, "o", cmp.Or(os.Getenv("REC_OUT_DIR"), "out"), "output directory for the artifacts")
	flag.StringVar(&opts.Engine, "engine", os.Getenv("REC_ENGINE"), "transcription engine (whisper|gemini, default whisper)")
	flag.StringVar(&opts.WhisperModel, "model", "", "whisper model file (default REC_WHISPER_MODEL or the user cache dir)")
	flag.StringVar(&opts.VADModel, "vad-model", "", "silero VAD model file (default REC_VAD_MODEL or the user cache dir)")
	flag.StringVar(&opts.GeminiModel, "gemini-model", "gemini-2.5-flash", "Gemini model name")
	flag.StringVar(&opts.MinutesCmd, "minutes-cmd", os.Getenv("REC_MINUTES_CMD"), "command that reads the prompt and transcript on stdin and prints the minutes (default \"claude -p\")")
	flag.StringVar(&opts.PostCmd, "post-cmd", os.Getenv("REC_POST_CMD"), "command run with the output directory once the minutes are written")
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}

	// A blank post command means none, so it never reaches rec as an empty argv.
	opts.PostCmd = strings.TrimSpace(opts.PostCmd)
	if *showVersion {
		fmt.Println(version)
		return
	}
	action, err := command(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := action(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
