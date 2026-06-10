# Translate Media Source output with fake Translation Provider seam

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Add the end-to-end Media Source translation orchestration using an injectable Translation Provider seam. When a Target Language is requested for a Media Source, the CLI should write the source-language SRT, translate the Subtitle Cues, write the translated SRT, and report paths according to quiet mode.

## Acceptance criteria

- [ ] Media Source translation writes both `<base>.<transcription-provider>.srt` and `<base>.<transcription-provider>.<target-language>.srt`.
- [ ] Translated Subtitle Cues preserve source cue timestamps.
- [ ] `--translation-provider mistral|grok` is accepted and defaults from the Transcription Provider for Media Source runs.
- [ ] `--translation-model` is accepted and passed only to translation.
- [ ] `--model` remains scoped to transcription.
- [ ] Quiet Media Source translation prints exactly two stdout lines: source SRT first, translated SRT second.
- [ ] Successful quiet Media Source translation writes nothing to stderr.
- [ ] Non-quiet Media Source translation reports both created SRT paths.
- [ ] If translation fails after transcription, the source-language SRT remains on disk, the command returns non-zero, and quiet mode prints no success paths.

## Blocked by

- .scratch/target-language-translation/issues/02-return-subtitle-cues-before-writing-srt.md
