# Add Mistral Translation Provider

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Implement the Mistral Translation Provider for translating batches of Subtitle Cues through chat completions with structured JSON output. The provider should preserve cue identity, reject malformed mappings, and use the Mistral API key and default model.

## Acceptance criteria

- [ ] Mistral translation uses `MISTRAL_API_KEY`.
- [ ] The default Mistral translation model is `mistral-large-latest`.
- [ ] A custom `--translation-model` value overrides the default.
- [ ] Requests ask for structured JSON output and low-randomness translation.
- [ ] Responses are accepted only when every requested cue ID is returned exactly once with non-empty translated text.
- [ ] HTTP 429 and 5xx failures are retryable.
- [ ] HTTP 4xx failures other than 429 are not retryable.
- [ ] Network failures are retryable.
- [ ] Tests use local HTTP servers and fake environment lookups, not live Mistral calls.

## Blocked by

- .scratch/target-language-translation/issues/03-translate-media-source-output-with-fake-translation-provider-seam.md
