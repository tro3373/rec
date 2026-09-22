# Watch for calls

`rec watch` keeps running and records on its own whenever Slack, Zoom or
Chrome (Google Meet) opens the microphone. A call starts when one of those
apps has a running recording stream and ends once every such stream has been
gone for 10 seconds. Each call gets its own output dir and the same
transcript / minutes / post pipeline as a manual run. This is what the
user's `rec.service` unit runs.

## Sub-features

- `watch-start` a running stream from `slack`, `zoom` or `chrome` starts a recording.
- `watch-ignore` streams from other binaries (ffmpeg, parecord, ...) start nothing.
- `watch-muted` a paused (corked) stream alone does not start a call, but keeps one going.
- `watch-end` the recording stops 10s after the last watched stream goes away, then the pipeline runs.
- `watch-repeat` a second call in the same watch gets a second output dir.

## How to get to it (user POV)

- `rec watch` in a terminal.
- `systemctl --user start rec` (the installed unit runs `rec watch`).

## Driving it with rv

Preconditions:

- The user's `rec.service` is **stopped**. `$rv doctor` prints its state;
  `rv call` refuses while it is `active`, because that instance would record
  the fake call too and run the user's real minutes and post commands. Ask
  the user to stop it, and to restart it afterwards.
- A fresh run: `id=$($rv up)`, doctor all `ok`.

- **Start watching.** `$rv start "$id" watch`, then
  `$rv wait "$id" 'watching for calls from slack, zoom, chrome' 30`.
- **Ignored stream.** `$rv call "$id" rvprobe`, wait 3s,
  `$rv capture "$id" ignored`: no `call detected` line. `$rv hangup "$id"`.
- **Call starts.** `$rv call "$id" zoom`, then
  `$rv wait "$id" 'call detected, recording' 10`. The banner names the
  `rv_<id>_*` devices.
- **Talk.** `$rv say "$id" other 'あしたの かいぎは さんじから です'`.
- **Call ends.** `$rv hangup "$id"`, then `$rv wait "$id" 'done: ' 300`
  (10s quiet period plus whisper). `$rv capture "$id" after-call`. The
  output dir has `transcript.md` with a `相手:` line containing `会議`.
- **Stop watching.** `$rv stop "$id"`, `$rv wait "$id" 'Pane is dead'`.
  rec watch has no Ctrl-C handler, so the status is a signal, not 0.

## Gotchas

- Not yet driven end to end: when this skill was written the user's
  `rec.service` was active. `PULSE_PROP=application.process.binary=<app>`
  was confirmed to set the binary pactl reports, which is what `rv call`
  relies on. Update this line after the first successful run.
- The watcher sees the whole session's streams, not only this run's. A real
  Chrome tab or Slack huddle during the run starts a recording of the
  (silent) virtual devices; check `said.txt` against the output dirs.
- `rv call` opens an uncorked stream. `watch-muted` needs a corked stream and
  has no rv helper yet; report it as skipped.
- The 10s quiet period is fixed; do not wait less than ~12s after `hangup`
  before expecting `transcribing with`.
