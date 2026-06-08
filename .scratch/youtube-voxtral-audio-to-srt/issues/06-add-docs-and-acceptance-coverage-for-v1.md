# Add docs and acceptance coverage for v1

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Document the v1 workflow and add final acceptance coverage around the completed behavior. The docs should explain dependencies, environment configuration, command examples, outputs, and scoped limitations using the project glossary language.

This slice should verify that the implemented tool matches the PRD as a whole.

## Acceptance criteria

- [ ] README documents installing or providing `yt-dlp`, setting `MISTRAL_API_KEY`, and running `video-to-srt`.
- [ ] README documents default outputs, `--output-dir`, `--model`, YouTube cookie flags, and `--quiet`.
- [ ] README clearly states v1 scope: YouTube Sources only, Voxtral only, no language flag, and no approximate timestamp fallback.
- [ ] A top-level acceptance-style test verifies the final successful workflow using fake external seams.
- [ ] The full Go test suite passes without real YouTube or Mistral access.

## Blocked by

- .scratch/youtube-voxtral-audio-to-srt/issues/04-harden-voxtral-error-handling-and-retry-policy.md
- .scratch/youtube-voxtral-audio-to-srt/issues/05-complete-cli-output-and-progress-contract.md
