---
name: verify-rec
description: Prove a change to rec (the meeting-recorder CLI in this repo) works by running the real binary - record two audio tracks, Ctrl-C, whisper transcription, minutes, post command - against private virtual audio devices, and keep the transcript, minutes and pane captures as evidence. Use after changing anything under cmd/ or internal/rec/, or when asked to run, demo or check rec (single recording or `rec watch`).
---

# verify-rec

rec is a CLI (surface: terminal, no server). A run records the default
microphone (`自分`) and the monitor of the default speaker (`相手`) with two
ffmpeg processes until Ctrl-C, then runs `whisper-cli`, writes
`transcript.md`, pipes the prompt plus transcript into the minutes command,
writes `minutes.md`, and runs the post command with the output dir.

Everything below goes through `scripts/rv` (run `scripts/rv help`). Paths in
this file are relative to the repo root; the helper is
`.claude/skills/verify-rec/scripts/rv`.

## Why a helper, and what it isolates

The audio defaults are global to the user's desktop session, and the user runs
`rec watch` as a systemd user service. Never `pactl set-default-*`, never touch
`~/.config/rec`, never `systemctl --user stop rec` without asking. `rv` gives
each run its own:

| Thing | Where | Removed by `down` |
| --- | --- | --- |
| rec binary built from the working tree, version = `git describe --always --dirty` | `$XDG_RUNTIME_DIR/rec-verify/<id>/bin/rec` | yes |
| two null sinks `rv_<id>_self`, `rv_<id>_other` (silent, never the default) | PipeWire, module ids in `run.env` | yes |
| `pactl` shim: only `pactl -f json info` is rewritten to name those sinks as the defaults; `list sinks`, `list source-outputs`, `subscribe` are the real pactl | `<state>/shim/pactl` | yes |
| config dir (`XDG_CONFIG_HOME`), so `~/.config/rec/env` and custom prompts are not read | `<state>/config` | yes |
| `REC_*` and `GEMINI_API_KEY` cleared; `REC_MINUTES_CMD=rv-minutes`, `REC_POST_CMD=rv-post` stubs | env of the pane | - |
| tmux server `-L rv-<id>` inside systemd scope `rec-verify-<id>.scope` | - | yes |
| evidence | `out/verify/<id>/` (gitignored) | **no, never** |

Runs are independent, so two can go side by side. The scope matters: the
agent harness reaps processes a tool call leaves behind, and a plain
`tmux new-session -d` got killed mid-recording between tool calls.

The stubs are the only mocks. The minutes command and post command are
already a user-configured process boundary in rec (`-minutes-cmd`,
`-post-cmd`); the stubs record what crossed it. Recording, ffmpeg, whisper,
merging, file writes and the post call are all real.

## Launch

Prerequisites (already on this machine): `go ffmpeg whisper-cli espeak-ng
paplay parecord jq tmux systemd-run` and both models in
`~/.cache/whisper.cpp/` (`make whisper-model vad-model`). `rv up` fails with
the missing one.

```sh
rv=.claude/skills/verify-rec/scripts/rv
id=$($rv up)                  # builds, creates sinks + dirs, prints the id
$rv start "$id"               # rec with no args; append any rec flags
$rv wait "$id" 'output:' 30   # ready: the banner ends with the output dir
```

Ready means the pane shows `recording (Ctrl-C to stop)` and the `self:`
line names `rv_<id>_self.monitor`. If it names a real device, the shim was
bypassed; stop and run doctor.

A shell variable does not survive between tool calls. Keep the id and
re-set `rv=` and `id=` in every call.

## Doctor

```sh
$rv doctor "$id"
```

Read-only. `ok` lines: binary version matches, the user's default sink and
source are what they were at `up` (proves nothing global moved), both sinks
exist, the shim resolves them, evidence dir exists. `WARN` if the tree
changed since the build (then `down` and `up` again). `info` lines: the
user's `rec.service` state and the pane's last line. Run it first whenever
anything looks off.

## Drive

Single recording, the main user path:

```sh
$rv say "$id" self  'りんご と みかん を かいました'     # into 自分
$rv say "$id" other 'あしたの かいぎは さんじから です'  # into 相手
$rv stop "$id"                                           # Ctrl-C in the pane
$rv wait "$id" 'Pane is dead' 300
$rv capture "$id" final
```

- `say` uses espeak-ng's Japanese voice. Write kana: kanji gets misread.
  whisper returns e.g. `リンゴとミカの買いまひいた` and
  `明日の会議派騒ぎからです`, so assert on keywords (`リンゴ|りんご`,
  `会議`), not exact text.
- `say` returns after playback plus 1s.
- The pane ends with `Pane is dead (status N, ...)`. `status 0` is success.
- Whisper on CPU takes a few seconds for a short clip; the model load is
  about 0.4s.

Other entry points (flags, watch, prompts, env file) are in
[`features/`](features/README.md). Pass rec flags after the id:
`$rv start "$id" -minutes-cmd "claude -p"`. Flags win over the stub env vars.
`--real-devices` as the first arg skips the shim and records the user's real
mic and speaker monitor: that proves device detection only, the content is
whatever the room produces, and it records the user. Ask first.

## Evidence

All under `out/verify/<id>/`; `$rv path "$id" out` prints rec's own
timestamped dir inside it.

| File | Proves |
| --- | --- |
| `said.txt` | what was injected, when, into which track |
| `pane-*.txt` | the terminal as the user saw it: banner, whisper log, `done: .../minutes.md`, exit status |
| `<out>/self.wav`, `other.wav`, `mixed.wav` | the recording happened; `other.wav` non-silent means the monitor track works |
| `<out>/transcript.md` | merge + labels: a `自分:` line with the self keyword, a `相手:` line with the other keyword, ordered by time |
| `minutes-stdin.txt` | what the minutes command got: the rendered prompt, a `---` line, the transcript |
| `<out>/minutes.md` | the minutes command's stdout was saved (stub text `rv-minutes-stub`) |
| `post-argv.txt`, `post-dir-listing.txt` | the post command ran once with the output dir as last arg, after `minutes.md` existed |

Minimum proof for a recording change:

```sh
out=$($rv path "$id" out); ev=$($rv path "$id" evidence)
grep -E '自分: .*(リンゴ|りんご)' "$out/transcript.md"
grep -E '相手: .*会議' "$out/transcript.md"
grep -c '^---$' "$ev/minutes-stdin.txt"        # 1
diff <(tail -n +2 <(sed -n '/^---$/,$p' "$ev/minutes-stdin.txt")) "$out/transcript.md"
cat "$ev/post-argv.txt"                         # == $out
grep -q 'Pane is dead (status 0' "$ev/pane-final.txt"
! grep -q 'Immediate exit requested' "$ev/pane-final.txt"
ffprobe -v error -show_entries format=duration -of csv=p=0 "$out/self.wav"  # ~ seconds from start to stop
ffmpeg -hide_banner -nostdin -i "$out/other.wav" -af volumedetect -f null - 2>&1 | grep max_volume  # well above -91 dB
```

To prove the real minutes path, start with `-minutes-cmd "claude -p"`. That
calls the Claude API with the transcript and costs money; the default stub
does not.

## Teardown

```sh
$rv down "$id"
```

Stops `rec-verify-<id>.scope` (tmux, rec, ffmpeg) and the fake-call unit,
unloads exactly the two modules recorded at `up`, trashes the state dir.
Nothing is killed by name. The evidence dir stays. Run `down` after a failed
attempt too, before the next `up`. Leftovers from a lost id:
`pactl list short modules | grep rv_` and
`systemctl --user list-units 'rec-verify-*'`.

## Helpers

`scripts/rv` is the only helper; `rv help` prints every sub-command. It is
the only place that knows the state layout, so read its comments before
bypassing it.
