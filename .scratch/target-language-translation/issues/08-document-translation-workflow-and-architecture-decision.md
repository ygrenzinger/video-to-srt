# Document translation workflow and architecture decision

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Update user-facing documentation and record the architectural decision behind the new translation boundary. The docs should explain how to request translation, how output files are named, how quiet mode behaves, which target language codes are accepted, and what translation quality guarantees the CLI does and does not make.

## Acceptance criteria

- [ ] README usage includes Media Source translation examples.
- [ ] README usage includes Subtitle Source retry translation examples.
- [ ] README options document `--target-language`, `--translation-provider`, and `--translation-model`.
- [ ] README lists or links the accepted Target Language codes.
- [ ] README explains quiet output for translated Media Source and Subtitle Source runs.
- [ ] README documents `MISTRAL_API_KEY` and `XAI_API_KEY` requirements for translation.
- [ ] README states that the language-code allowlist is product-defined and not a provider capability guarantee.
- [ ] A short ADR records the separate Translation Provider boundary and the Subtitle Source decision.
- [ ] `go test ./...` passes after documentation and ADR changes.

## Blocked by

- .scratch/target-language-translation/issues/07-wire-real-translation-providers-into-the-cli.md
