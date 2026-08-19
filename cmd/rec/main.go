package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/tro3373/rec/internal/rec"
)

// version is overwritten at build time via -ldflags.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	opts := rec.Options{GeminiAPIKey: os.Getenv("GEMINI_API_KEY")}
	flag.StringVar(&opts.OutRoot, "o", cmp.Or(os.Getenv("REC_OUT_DIR"), "out"), "output directory for the artifacts")
	flag.StringVar(&opts.Engine, "engine", os.Getenv("REC_ENGINE"), "transcription engine (whisper|gemini, default whisper)")
	flag.StringVar(&opts.WhisperModel, "model", "", "whisper model file (default REC_WHISPER_MODEL or the user cache dir)")
	flag.StringVar(&opts.VADModel, "vad-model", "", "silero VAD model file (default REC_VAD_MODEL or the user cache dir)")
	flag.StringVar(&opts.GeminiModel, "gemini-model", "gemini-2.5-flash", "Gemini model name")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if err := rec.Run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
