# Record a meeting

The user runs `rec`, talks, and presses Ctrl-C when the meeting ends. rec
records their microphone as `自分` and the speaker output as `相手`, then
transcribes both, merges them by time into `transcript.md`, writes
`minutes.md`, and runs the post command if one is set. Every run gets its own
`<out>/<YYYYmmdd-HHMMSS>/` directory.

## Sub-features

- `record-banner` prints the devices and the output dir before recording.
- `record-stop` Ctrl-C stops only the recording; the rest of the pipeline runs.
- `record-tracks` writes `self.wav`, `other.wav` and `mixed.wav`.
- `record-transcript` labels each utterance `自分` / `相手` and orders by time.
- `record-repeats` collapses identical consecutive segments to at most two.
- `record-minutes` hands prompt + `---` + transcript to the minutes command and saves its stdout.
- `record-preflight` fails before recording when a model, the minutes command or the post command is missing.

## How to get to it (user POV)

- `rec` in a terminal, then Ctrl-C.
- `rec -o <dir>` or `REC_OUT_DIR` to choose where the output goes.
- `rec -model <path>` / `-vad-model <path>` to choose the whisper models.

## Driving it with rv

Preconditions:

- A fresh run: `id=$($rv up)`, doctor all `ok`.

- **Start.** `$rv start "$id"`, then `$rv wait "$id" 'output:' 30` and
  `$rv capture "$id" recording`. The pane shows `recording (Ctrl-C to stop)`,
  `self:   rv_<id>_self.monitor`, `other:  rv_<id>_other.monitor`, and
  `output: <repo>/out/verify/<id>/<timestamp>`.
- **Talk on both sides.** `$rv say "$id" self 'りんご と みかん を かいました'`
  then `$rv say "$id" other 'あしたの かいぎは さんじから です'`. `said.txt`
  gains two lines.
- **Stop.** `$rv stop "$id"`, then `$rv wait "$id" 'Pane is dead' 300` and
  `$rv capture "$id" final`. The pane shows `transcribing with whisper`,
  `generating the minutes`, `done: <out>/minutes.md`,
  `Pane is dead (status 0`.
- **Transcript.** `cat "$($rv path "$id" out)/transcript.md"` has a
  `- [HH:MM:SS] 自分:` line containing `リンゴ` or `りんご` before a
  `相手:` line containing `会議`.
- **Minutes input.** `minutes-stdin.txt` in the evidence dir is the prompt
  from `internal/rec/prompts/minutes.md` with the sample line
  `- [00:00:00] 自分: 発言`, one `---` line, then exactly `transcript.md`.
- **Post.** `post-argv.txt` is the single line `<out>`, and
  `post-dir-listing.txt` includes `minutes.md`.
- **Preflight.** Fresh run, `$rv start "$id" -model /nonexistent`,
  `$rv wait "$id" 'Pane is dead'`. The pane shows `model not found:
  /nonexistent` with a `curl` hint, `Pane is dead (status 1`, and no
  timestamped dir exists under the evidence dir.

## Gotchas

- Regression to watch: with ffmpeg 9 the wavs are written in 256KiB blocks
  (about 8s). If ffmpeg gets SIGINT twice (the terminal's Ctrl-C plus rec's
  own), it prints `Immediate exit requested` and drops the unwritten block,
  so the wavs stop at a multiple of 8.19s and late speech is missing. Fixed
  on 2026-09-22 by giving the recording ffmpeg its own process group. Always
  compare the wav duration with the wall time between start and stop.
- Silence on a track gives no lines for it, not an error. An empty transcript
  still reaches the minutes command.
- whisper hallucinates on silence in long recordings without VAD; the run
  passes `--vad`, so `record-repeats` needs a crafted repeat, not a quiet
  track, to show up.
- `--real-devices` records the user's real microphone. Ask before using it.
- `out/verify/<id>/` also holds evidence files next to rec's timestamped dir;
  use `$rv path "$id" out` rather than globbing.
