# Return Subtitle Cues before writing SRT

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Refactor the transcription boundary so Transcription Providers return Subtitle Cues to app orchestration before SRT writing. The visible CLI behavior without translation should remain unchanged while creating the transformation point needed by translation.

## Acceptance criteria

- [ ] Transcription produces Subtitle Cues before SRT formatting.
- [ ] App orchestration writes the source-language SRT after transcription succeeds.
- [ ] Existing YouTube Source and Local Video Source commands produce the same SRT paths as before.
- [ ] Existing quiet and non-quiet output behavior is unchanged when no Target Language is requested.
- [ ] Temporary audio cleanup behavior remains unchanged on transcription success and failure.
- [ ] Provider and transport tests continue to verify request shape, response decoding, retry behavior, and SRT output behavior through the public provider/app contract.

## Blocked by

- .scratch/target-language-translation/issues/01-add-target-language-cli-contract-and-glossary.md
