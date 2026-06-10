# Add Grok Translation Provider

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Implement the Grok Translation Provider for translating batches of Subtitle Cues through chat completions with structured JSON output. It should follow the same cue identity, validation, retry, and model override contract as Mistral.

## Acceptance criteria

- [ ] Grok translation uses `XAI_API_KEY`.
- [ ] The default Grok translation model is `grok-4.3`.
- [ ] A custom `--translation-model` value overrides the default.
- [ ] Requests ask for structured JSON output and low-randomness translation.
- [ ] Responses are accepted only when every requested cue ID is returned exactly once with non-empty translated text.
- [ ] HTTP 429 and 5xx failures are retryable.
- [ ] HTTP 4xx failures other than 429 are not retryable.
- [ ] Network failures are retryable.
- [ ] Tests use local HTTP servers and fake environment lookups, not live Grok calls.

## Blocked by

- .scratch/target-language-translation/issues/05-add-mistral-translation-provider.md
