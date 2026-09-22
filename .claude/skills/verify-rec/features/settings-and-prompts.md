# Settings and prompts

Every run reads `~/.config/rec/env` (honoring `XDG_CONFIG_HOME`) and exports
all its `KEY=VALUE` lines; a shell variable wins over the file and a flag wins
over both. Files in `~/.config/rec/prompts/` named `minutes.md` or
`transcribe.md` replace the built-in prompts, and are rendered before
recording so a broken template fails at once. `rec -version` prints the build
version.

## Sub-features

- `env-file` the file's variables reach rec and the commands it runs.
- `env-precedence` flag > shell > file.
- `env-parse-error` a malformed line fails with `cannot parse <path>: line N`.
- `prompt-override` a `minutes.md` in the prompts dir replaces the built-in.
- `prompt-broken` an unknown template variable fails before recording.
- `version` `rec -version` prints the version and exits 0.

## How to get to it (user POV)

- Edit `~/.config/rec/env`, then run `rec` (or restart the service).
- Copy `internal/rec/prompts/minutes.md` to `~/.config/rec/prompts/` and edit.
- `rec -version`.

## Driving it with rv

Preconditions:

- A fresh run. `cfg=$($rv path "$id" config)/rec; mkdir -p "$cfg/prompts"`.
  Write files there, never in `~/.config/rec`.

- **Env file is read.** `echo 'REC_VAD_MODEL=/nonexistent-vad' >"$cfg/env"`,
  `$rv start "$id"`, `$rv wait "$id" 'Pane is dead'`. The pane shows
  `model not found: /nonexistent-vad`.
- **Flag wins.** Same file, fresh pane (`down`, `up`, rewrite the file):
  `$rv start "$id" -vad-model ~/.cache/whisper.cpp/ggml-silero-v5.1.2.bin`.
  The recording banner appears.
- **Parse error.** `echo 'not a pair' >"$cfg/env"`, `$rv start "$id"`. The
  pane shows `cannot parse .../rec/env: line 1: expected KEY=VALUE`.
- **Prompt override.** `printf 'RV-PROMPT-MARKER {{.Self}}\n' >"$cfg/prompts/minutes.md"`,
  record per [record](./record.md). `minutes-stdin.txt` starts with
  `RV-PROMPT-MARKER 自分`.
- **Broken prompt.** `printf '{{.Nope}}\n' >"$cfg/prompts/minutes.md"`,
  `$rv start "$id"`. The pane shows `cannot fill the prompt minutes.md` with
  no recording banner, `Pane is dead (status 1`.
- **Version.** `"$($rv path "$id" state)/bin/rec" -version` prints the same
  value doctor shows.

## Gotchas

- rv puts `REC_MINUTES_CMD` / `REC_POST_CMD` in the environment, so the env
  file cannot override those two (shell wins). Use another variable to prove
  the file is read.
- rv passes `-o`, so `REC_OUT_DIR` in the file is ignored too, by design.
- The env file is read at startup only; editing it mid-run changes nothing.
