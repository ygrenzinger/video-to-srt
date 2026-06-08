# Add Voxtral Transcription Provider happy path

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Add the first complete transcription path: a Transcription Provider contract, a Voxtral provider implementation using direct multipart HTTP to Mistral's audio transcription API, and app wiring from YouTube Source to Audio Artifact to SRT.

This slice should produce a `.voxtral.srt` file from a fake Voxtral response with usable timestamped segments. Error hardening and retry behavior can be completed in the following slice.

## Acceptance criteria

- [ ] The CLI accepts `--provider voxtral` and rejects unsupported providers.
- [ ] The CLI accepts `--model`, defaulting to `voxtral-mini-latest`.
- [ ] Voxtral requires `MISTRAL_API_KEY` and sends it using the Mistral API authentication header.
- [ ] The Voxtral HTTP request uploads the Audio Artifact as multipart form data with the selected model and segment timestamp granularity.
- [ ] Timestamped Voxtral segments are converted into Subtitle Cues and written to `<audio-stem>.voxtral.srt`.
- [ ] App-level tests verify a full fake YouTube Source to Audio Artifact to SRT success path without real network calls.

## Blocked by

- .scratch/youtube-voxtral-audio-to-srt/issues/02-add-subtitle-cue-formatting-and-atomic-srt-writing.md
