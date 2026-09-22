# rec

A CLI that records an online meeting (Zoom, Google Meet, Teams, ...) and turns
it into markdown minutes.

- Records two tracks: your voice (default microphone) and everyone else
  (monitor of the default speaker output)
- Ctrl-C stops the recording, then transcription and minutes generation run
  automatically
- Attributes every utterance to either you or the other side
- `rec watch` detects a call starting on its own and records it unattended

The transcript and the minutes are written in Japanese. See [Limitations](#limitations).

## Requirements

| Purpose | Needed |
| --- | --- |
| Recording | PipeWire or PulseAudio, `ffmpeg`, `pactl` |
| Transcription (default) | `whisper-cli` plus a transcription model and a VAD model |
| Transcription (alternative) | `GEMINI_API_KEY` |
| Minutes | the `claude` CLI, or any command set with `-minutes-cmd` |
| Posting to Slack (optional) | the `slk` CLI |

## Setup

```sh
make bootstrap              # dependencies, the binary, the unit and an env template
$EDITOR ~/.config/rec/env
make service                # enable the unit and start it
```

`make bootstrap` is `make setup` followed by `make install`. `install` is not
wired to `setup` on its own: setup wants `sudo`, re-resolves the Go tools over
the network, and ends in a check that fails on anything it cannot install
itself. That cost belongs to the first run, not to every rebuild.

What `make setup` runs:

| Target | Does |
| --- | --- |
| `deps-system` | `pacman -S --needed ffmpeg libpulse whisper-cpp` |
| `deps-go` | golangci-lint, gotestsum, go-test-coverage, goreleaser |
| `whisper-model` | downloads ggml-large-v3-turbo.bin (about 1.6GB), skipped if present |
| `vad-model` | downloads ggml-silero-v5.1.2.bin (about 900KB), skipped if present |
| `deps-check` | lists every missing command and model path, and fails if any is missing |

Running it again is safe: pacman gets `--needed`, both models are skipped once
the file is there, and `go install` overwrites.

The `claude` CLI has no distro package, so install it yourself. Until you do,
`deps-check` stops with the missing list and takes `setup` and `bootstrap` down
with it. Without pacman, `deps-system` prints the package names and stops.
`make deps-check` on its own reports whatever is still missing.

## Usage

```sh
rec              # start recording, hit Ctrl-C when the meeting ends
rec watch        # keep running, record every call it detects
rec -engine gemini
rec -version
```

## Output

Everything lands in `out/<YYYYmmdd-HHMMSS>/`.

- `self.wav`: your voice
- `other.wav`: the other side
- `mixed.wav`: both sides in one file, for listening back
- `transcript.md`: timestamped transcript
- `minutes.md`: the minutes

## Watching for calls

`rec watch` stays running and records on its own whenever Slack, Zoom or Google
Meet opens the microphone. Every call produces its own output directory, exactly
like a manual run.

Detection reads the PulseAudio/PipeWire recording streams through `pactl`:

| App | `application.process.binary` |
| --- | --- |
| Slack huddle | `slack` |
| Zoom | `zoom` |
| Google Meet | `chrome` (it runs inside the browser) |

A call starts as soon as one of them has a *running* stream, and ends only once
every stream has been gone for 10 seconds. That delay is the point: the apps
tear a stream down and open the next one within the same second while the call
goes on. Muting yourself does not stop the recording either, because a paused
stream still counts as a call in progress.

## Running as a service

It has to be a **user** unit. Recording talks to the session's PipeWire socket,
which a system unit running as root cannot reach.

`make install` writes the binary to `~/.local/bin`, the unit to
`~/.config/systemd/user`, and a starter `~/.config/rec/env` only when that file
does not exist yet, so your own edits survive a reinstall.

The unit hardcodes the first and the last of those, so the `bin_dir` and
`env_file` variables move the files without the service following them. Edit
`systemd/rec.service` too if you want them somewhere else.

After rebuilding, `make install && make service` picks up the new binary:
`service` restarts the unit rather than leaving the old process running.

```sh
journalctl --user -u rec -f
```

`loginctl enable-linger` is not needed. Without a login there is no PipeWire
session, and so no call to record.

The unit sets `PATH` explicitly. The systemd user manager does not inherit the
login shell's `PATH`, and `rec` shells out to `claude` and `slk`.

## Generating the minutes

`-engine` only picks the transcription. The minutes always come from the
`-minutes-cmd` command, `claude -p` unless set, whichever engine ran.

The command gets the prompt, a `---` line and the transcript on stdin, and
prints the markdown minutes on stdout. The command line is split on spaces
only, so put anything that needs quoting in a script.

```sh
rec -minutes-cmd "gemini"             # Gemini CLI
rec -minutes-cmd "ollama run gemma3"  # a local model
```

`rec` checks that the command exists before it starts recording.

## Posting to Slack

`-slack` posts the minutes when the run is over, through the `slk` CLI, which
already owns the token and the channel.

- the summary section becomes the message
- the whole `minutes.md` is attached as a file, since Slack truncates a post at
  40,000 characters and recommends a snippet for anything long

`rec` calls `slk` with `--no-auto-upload`. Without that flag `slk` turns any
message over 10 lines into a file on its own, which would leave the post empty.

## Options

| Flag | Env | Default | Description |
| --- | --- | --- | --- |
| `-o` | `REC_OUT_DIR` | `out` | output directory for the artifacts |
| `-engine` | `REC_ENGINE` | `whisper` | `whisper` or `gemini` |
| `-model` | `REC_WHISPER_MODEL` | `$XDG_CACHE_HOME/whisper.cpp/ggml-large-v3-turbo.bin` | whisper model file |
| `-vad-model` | `REC_VAD_MODEL` | `$XDG_CACHE_HOME/whisper.cpp/ggml-silero-v5.1.2.bin` | silero VAD model file |
| `-gemini-model` | - | `gemini-2.5-flash` | Gemini model name |
| `-minutes-cmd` | `REC_MINUTES_CMD` | `claude -p` | command that writes the minutes |
| `-slack` | `REC_SLACK` | off | post the minutes to Slack with `slk` |
| `-slack-channel` | `REC_SLACK_CHANNEL` | the `slk` config | Slack channel to post to |
| `-version` | - | - | print the version and exit |

## Development

```sh
make build         # writes ./rec
go build -o rec ./cmd/rec
make test
make lint
make clean-cache   # drop the Go build and test caches
```

`clean` only removes `./rec`. The Go caches are shared by every module on the
machine, so `clean-cache` is a separate target rather than part of every build.

`bootstrap` and `all` live in the root `Makefile` because they span the files
below; everything else is split by concern under `.mk/`.

| File | Contains |
| --- | --- |
| `.mk/tools.mk` | dependency install (`setup`, `deps-*`, `whisper-model`) |
| `.mk/go.mk` | `tidy`, `fmt`, `deps`, `update` |
| `.mk/lint.mk` | `lint` (diff only), `lint-all` (whole tree) |
| `.mk/build.mk` | `build`, `run`, `clean`, `clean-cache`, `install`, `service` |
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
- `rec watch` cannot tell Google Meet apart from anything else Chrome uses the
  microphone for. Any page that records audio starts a recording.
- whisper hallucinates on silence, repeating a phrase learned from its training
  data ("ご視聴ありがとうございました") for the whole silent stretch. VAD drops
  the silence before whisper sees it, and identical segments are collapsed to at
  most two in a row. Gemini gets the collapsing but not the VAD.
