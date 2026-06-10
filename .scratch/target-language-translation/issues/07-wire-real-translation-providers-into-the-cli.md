# Wire real Translation Providers into the CLI

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Connect the translation CLI flags and orchestration to the real Mistral and Grok Translation Providers while preserving the test seam for fake Translation Providers.

## Acceptance criteria

- [ ] `--translation-provider mistral` uses the Mistral Translation Provider.
- [ ] `--translation-provider grok` uses the Grok Translation Provider.
- [ ] Unsupported Translation Provider values are rejected before Media Source preparation, Subtitle Source parsing, transcription, or translation starts.
- [ ] Media Source translation uses the derived default Translation Provider when no explicit translation provider is passed.
- [ ] Subtitle Source translation defaults to Mistral when no explicit translation provider is passed.
- [ ] Translation model overrides flow only to the selected Translation Provider.
- [ ] App-level tests cover real-provider routing through fake provider functions or test seams without live API calls.

## Blocked by

- .scratch/target-language-translation/issues/05-add-mistral-translation-provider.md
- .scratch/target-language-translation/issues/06-add-grok-translation-provider.md
