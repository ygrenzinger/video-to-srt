# Harden Voxtral error handling and retry policy

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Make the Voxtral Transcription Provider production-ready for v1 by handling expected failure modes clearly and retrying only transient provider failures.

The provider should retry network errors, rate limits, and server errors with short backoff. It should not fabricate timestamps when Voxtral returns text without usable timestamped segments.

## Acceptance criteria

- [ ] Missing `MISTRAL_API_KEY` fails before attempting a provider request.
- [ ] Network failures, HTTP 429, and HTTP 5xx responses are retried according to the v1 retry policy.
- [ ] Non-retryable HTTP responses fail without retrying.
- [ ] Malformed JSON responses fail clearly.
- [ ] Responses with no usable timestamped segments fail clearly and do not write an approximate SRT.
- [ ] Tests cover retryable failures, non-retryable failures, malformed responses, missing timestamps, and retry exhaustion.

## Blocked by

- .scratch/youtube-voxtral-audio-to-srt/issues/03-add-voxtral-transcription-provider-happy-path.md
