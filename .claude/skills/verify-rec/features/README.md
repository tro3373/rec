# rec verification map

The maintained source of truth for how to prove rec's user-facing behavior.
Read this index before driving rec, then use the matching feature file as the
recipe. The harness is `scripts/rv` (see `../SKILL.md`).

## Baseline preconditions

- `rv=.claude/skills/verify-rec/scripts/rv; id=$($rv up)` from the repo root.
- `$rv doctor "$id"` shows only `ok` lines (a `WARN` about the tree means
  rebuild with a fresh `up`).
- rec reads no user config: private `XDG_CONFIG_HOME`, `REC_*` cleared,
  minutes and post commands are the `rv-minutes` / `rv-post` stubs.
- Never drive the user's own `rec` (the `rec.service` unit, or a `rec` they
  started). Only the pane `rv start` created.

## Driving conventions

- One `rv up` per scenario. Recipes start from a fresh run unless they say
  otherwise, and end with `$rv down "$id"`.
- Match on pane strings rec prints (`recording (Ctrl-C to stop)`,
  `transcribing with`, `generating the minutes`, `done: `, `error: `,
  `Pane is dead (status N`), never on screen positions.
- `say` text is kana. Assert on keywords, since espeak + whisper garble
  the rest.

## Evidence and skip reporting

- Capture the pane before and after the action (`$rv capture "$id" <name>`),
  not only the end.
- For every run, name the feature id and the entry point used.
- Side effects count as evidence: files in `$rv path "$id" out`,
  `minutes-stdin.txt`, `post-argv.txt`.
- An entry point you could not drive is reported as skipped, with the
  command tried and the missing precondition. Never report it as covered by a
  different entry point.

## Feature entry contract

Each feature file opens with an H1 and one paragraph of user-visible
behavior, then exactly four H2 sections in order: `Sub-features`,
`How to get to it (user POV)`, `Driving it with rv` (starting with
`Preconditions:`), `Gotchas`. User paths, stable handles, commands and
observable evidence only; no implementation detail.

## Features

- [Record a meeting](./record.md): `rec` until Ctrl-C, two tracks, whisper transcript, minutes, post command. Proven end to end on 2026-09-22.
- [Watch for calls](./watch.md): `rec watch` starting and stopping recordings on its own. Not yet driven (see its Gotchas).
- [Minutes and post commands](./minutes-and-post.md): `-minutes-cmd`, `-post-cmd`, their env vars and failure handling.
- [Settings and prompts](./settings-and-prompts.md): `~/.config/rec/env`, precedence, custom prompt files, `-version`.
- [Gemini engine](./gemini-engine.md): `-engine gemini`; the key check is free, a real run bills the API.
