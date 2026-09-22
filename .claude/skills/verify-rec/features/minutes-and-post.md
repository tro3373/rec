# Minutes and post commands

After the transcript is written, rec pipes the minutes prompt, a `---` line
and the transcript into the minutes command (`claude -p` unless set) and saves
its stdout as `minutes.md`. Then, if set, it runs the post command with the
output dir as the last argument. A failing minutes command fails the run but
keeps the transcript; a failing post command only prints a warning.

## Sub-features

- `minutes-default` `claude -p` when nothing is set.
- `minutes-custom` `-minutes-cmd` / `REC_MINUTES_CMD`, split on spaces only.
- `minutes-fail` non-zero exit => `error: cannot generate the minutes`, the transcript path is printed, exit 1.
- `post-run` `-post-cmd` / `REC_POST_CMD` gets the output dir as last arg.
- `post-fail` non-zero exit => `warning: post command ... failed`, exit 0.
- `cmd-preflight` a missing minutes or post command fails before recording.

## How to get to it (user POV)

- `rec -minutes-cmd "<cmd args>"`, `rec -post-cmd <cmd>`.
- `REC_MINUTES_CMD`, `REC_POST_CMD` in the shell or in `~/.config/rec/env`.

## Driving it with rv

Preconditions:

- A fresh run per bullet. Every recording bullet uses the `say` / `stop` /
  `wait 'Pane is dead'` steps of [record](./record.md).

- **Stub path (default).** `$rv start "$id"`. Evidence: `minutes-stdin.txt`,
  `minutes.md` containing `rv-minutes-stub`, `post-argv.txt`.
- **Real minutes.** Ask first: it sends the transcript to the Claude API.
  `$rv start "$id" -minutes-cmd "claude -p"`. `minutes.md` has the headings
  from the prompt (概要 / 決定事項 / TODO / 議論の流れ) and mentions only
  what the transcript says.
- **Minutes failure.** `$rv start "$id" -minutes-cmd false`. The pane shows
  `error: cannot generate the minutes with false` and
  `the transcript is kept at <out>/transcript.md`, `Pane is dead (status 1`;
  `transcript.md` exists, `minutes.md` does not, `post-argv.txt` does not.
- **Post failure.** `$rv start "$id" -post-cmd false`. The pane shows
  `done: <out>/minutes.md`, `warning: post command false failed`,
  `Pane is dead (status 0`.
- **Missing command.** `$rv start "$id" -post-cmd rv-nope`. The pane shows
  `cannot find rv-nope, the post command` right away, with no `recording`
  banner, `Pane is dead (status 1`.

## Gotchas

- The command line is split on spaces, no quoting: `-minutes-cmd "sh -c 'x y'"`
  does not do what it looks like. Use a script.
- A flag wins over the `REC_*_CMD` values rv sets, so passing
  `-minutes-cmd` replaces the stub; `minutes-stdin.txt` is then not written.
- An empty `-post-cmd ""` means no post command, not an error.
