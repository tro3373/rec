# rec

A CLI that records an online meeting (Zoom, Google Meet, Teams, ...) and turns
it into markdown minutes.

- Records two tracks: your voice (default microphone) and everyone else
  (monitor of the default speaker output)
- Ctrl-C stops the recording, then transcription and minutes generation run
  automatically
- Attributes every utterance to either you or the other side

The transcript and the minutes are written in Japanese. See [Limitations](#limitations).

## Requirements

| Purpose | Needed |
| --- | --- |
| Recording | PipeWire or PulseAudio, `ffmpeg`, `pactl` |
| Transcription (default) | `whisper-cli` plus a transcription model and a VAD model |
| Transcription (alternative) | `GEMINI_API_KEY` |
| Minutes | the `claude` CLI |

## Setup

```sh
make setup        # system packages, Go tools and the whisper model
make deps-check   # list whatever is still missing
```

What `make setup` runs:

| Target | Does |
| --- | --- |
| `deps-system` | `pacman -S --needed ffmpeg libpulse whisper-cpp` |
| `deps-go` | golangci-lint, gotestsum, go-test-coverage, goreleaser |
| `whisper-model` | downloads ggml-large-v3-turbo.bin (about 1.6GB), skipped if present |
| `vad-model` | downloads ggml-silero-v5.1.2.bin (about 900KB), skipped if present |
| `deps-check` | reports every missing command and model path |

The `claude` CLI has no distro package, so install it yourself.
Without pacman, `deps-system` prints the package names and stops.

## Usage

```sh
make build         # writes /tmp/rec
go build -o rec ./cmd/rec

./rec              # start recording, hit Ctrl-C when the meeting ends
./rec -engine gemini
./rec -version
```

## Output

Everything lands in `out/<YYYYmmdd-HHMMSS>/`.

- `self.wav`: your voice
- `other.wav`: the other side
- `mixed.wav`: both sides in one file, for listening back
- `transcript.md`: timestamped transcript
- `minutes.md`: the minutes

## Options

| Flag | Env | Default | Description |
| --- | --- | --- | --- |
| `-o` | `REC_OUT_DIR` | `out` | output directory for the artifacts |
| `-engine` | `REC_ENGINE` | `whisper` | `whisper` or `gemini` |
| `-model` | `REC_WHISPER_MODEL` | `$XDG_CACHE_HOME/whisper.cpp/ggml-large-v3-turbo.bin` | whisper model file |
| `-vad-model` | `REC_VAD_MODEL` | `$XDG_CACHE_HOME/whisper.cpp/ggml-silero-v5.1.2.bin` | silero VAD model file |
| `-gemini-model` | - | `gemini-2.5-flash` | Gemini model name |
| `-version` | - | - | print the version and exit |

## Development

The Makefile is split by concern under `.mk/`.

| File | Contains |
| --- | --- |
| `.mk/tools.mk` | dependency install (`setup`, `deps-*`, `whisper-model`) |
| `.mk/go.mk` | `tidy`, `fmt`, `deps`, `update` |
| `.mk/lint.mk` | `lint` (diff only), `lint-all` (whole tree) |
| `.mk/build.mk` | `build`, `run`, `clean` |
| `.mk/test.mk` | `test` (gotestsum plus the coverage check) |
| `.mk/release.mk` | goreleaser wrappers |

## Limitations

- Output is Japanese only. The prompts and the speaker labels are hardcoded.
- Gemini receives the audio inline, so a long meeting exceeds the 18MB limit.
  Use whisper in that case.
- The recording devices are fixed at startup. Switching devices mid-meeting
  has no effect.
- The two tracks are transcribed sequentially. whisper handles both in one
  process so the model is loaded only once.
- Linux only. It depends on `pactl` and `ffmpeg -f pulse`.
- whisper hallucinates on silence, repeating a phrase learned from its training
  data ("ご視聴ありがとうございました") for the whole silent stretch. VAD drops
  the silence before whisper sees it, and identical segments are collapsed to at
  most two in a row. Gemini gets the collapsing but not the VAD.
