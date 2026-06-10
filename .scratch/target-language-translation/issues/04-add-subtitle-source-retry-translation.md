# Add Subtitle Source retry translation

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Accept existing SRT files as Subtitle Sources for translation-only retry runs. A Subtitle Source is valid only when a Target Language is present; the CLI should parse the SRT into Subtitle Cues, translate them, and write a translated SRT without touching the original file.

## Acceptance criteria

- [ ] An existing `.srt` path is accepted as a Subtitle Source only when `--target-language` is present.
- [ ] `.srt` input without `--target-language` is rejected clearly.
- [ ] Missing, unreadable, or malformed Subtitle Sources are rejected before Translation Provider calls.
- [ ] Subtitle Source translation defaults to the Mistral Translation Provider.
- [ ] Subtitle Source translation writes `<source-basename>.<target-language>.srt`.
- [ ] The original Subtitle Source is never overwritten.
- [ ] Quiet Subtitle Source translation prints exactly one stdout line: the translated SRT path.
- [ ] SRT parser tests cover valid SRT, multi-line cue text, invalid timing, malformed separators, and round trip with SRT formatting.

## Blocked by

- .scratch/target-language-translation/issues/03-translate-media-source-output-with-fake-translation-provider-seam.md
