# Gemini engine

`rec -engine gemini` (or `REC_ENGINE=gemini`) sends each track to the Gemini
API instead of running whisper locally, one request per track, with the
`transcribe.md` prompt. It needs `GEMINI_API_KEY`. The minutes still come from
the minutes command.

## Sub-features

- `gemini-key-check` no key => `GEMINI_API_KEY is not set` before recording.
- `gemini-transcribe` both tracks transcribed via the API, labels as with whisper.
- `gemini-model` `-gemini-model <name>` picks the model (default `gemini-2.5-flash`).
- `engine-unknown` any other engine name fails with `unknown transcription engine`.

## How to get to it (user POV)

- `rec -engine gemini`, `REC_ENGINE=gemini` in the shell or env file.

## Driving it with rv

Preconditions:

- A fresh run. `rv start` forwards `GEMINI_API_KEY` from the caller's shell
  only when `gemini` appears in the rec args.

- **Key check (free).** `GEMINI_API_KEY= $rv start "$id" -engine gemini`,
  `$rv wait "$id" 'Pane is dead'`. The pane shows
  `error: GEMINI_API_KEY is not set`, status 1, no recording banner.
- **Unknown engine (free).** `$rv start "$id" -engine nope`. The pane shows
  `unknown transcription engine: nope (whisper|gemini)`.
- **Real transcription (bills the API, ask first).**
  `$rv start "$id" -engine gemini`, then the record steps. The pane shows
  `transcribing with gemini`; `transcript.md` has the `自分` / `相手`
  keywords, `self.json` / `other.json` do not exist (whisper-only files).

## Gotchas

- The audio goes inline; long recordings exceed the 18MB request limit.
  Keep verification clips short.
- No VAD on this path: silence can come back as hallucinated text.
- The caller's `GEMINI_MODEL` env var is not what rec reads; only
  `-gemini-model` changes the model.
