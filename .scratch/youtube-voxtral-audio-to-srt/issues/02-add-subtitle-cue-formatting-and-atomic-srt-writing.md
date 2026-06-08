# Add Subtitle Cue formatting and atomic SRT writing

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Add the subtitle capability needed by the transcription flow: a Subtitle Cue model, SRT timestamp formatting, SRT document formatting, and atomic file writing.

This slice should be independently verifiable by formatting known timestamped cues into valid SRT output and writing them safely. It does not need to call Voxtral yet.

## Acceptance criteria

- [ ] Subtitle Cues can be formatted into numbered SRT blocks with `HH:MM:SS,mmm --> HH:MM:SS,mmm` timestamps.
- [ ] SRT output ends with a final newline and preserves cue order.
- [ ] Invalid cue timings fail clearly instead of writing misleading output.
- [ ] SRT writes are atomic so a formatting or write failure does not leave a partial final SRT.
- [ ] Tests cover timestamp conversion, cue indexing, final newline behavior, invalid timings, and atomic write behavior.

## Blocked by

- .scratch/youtube-voxtral-audio-to-srt/issues/01-create-go-cli-and-youtube-source-to-mp3.md
