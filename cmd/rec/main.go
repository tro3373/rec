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

func main() {
	name, args := splitCommand(os.Args[1:])
	showVersion := flag.Bool("version", false, "print the version and exit")
	opts := rec.Options{GeminiAPIKey: os.Getenv("GEMINI_API_KEY")}
	flag.StringVar(&opts.OutRoot, "o", cmp.Or(os.Getenv("REC_OUT_DIR"), "out"), "output directory for the artifacts")
	flag.StringVar(&opts.Engine, "engine", os.Getenv("REC_ENGINE"), "transcription engine (whisper|gemini, default whisper)")
	flag.StringVar(&opts.WhisperModel, "model", "", "whisper model file (default REC_WHISPER_MODEL or the user cache dir)")
	flag.StringVar(&opts.VADModel, "vad-model", "", "silero VAD model file (default REC_VAD_MODEL or the user cache dir)")
	flag.StringVar(&opts.GeminiModel, "gemini-model", "gemini-2.5-flash", "Gemini model name")
	flag.StringVar(&opts.MinutesCmd, "minutes-cmd", os.Getenv("REC_MINUTES_CMD"), "command that reads the prompt and transcript on stdin and prints the minutes (default \"claude -p\")")
	flag.BoolVar(&opts.SlackPost, "slack", os.Getenv("REC_SLACK") != "", "post the minutes to Slack with slk")
	flag.StringVar(&opts.SlackChannel, "slack-channel", os.Getenv("REC_SLACK_CHANNEL"), "Slack channel to post to (default: the slk config)")
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}

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
