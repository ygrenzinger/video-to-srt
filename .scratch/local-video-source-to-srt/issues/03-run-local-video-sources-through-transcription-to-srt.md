# Run Local Video Sources through transcription to SRT

Status: ready-for-agent

## Parent

.scratch/local-video-source-to-srt/PRD.md

## What to build

Wire Local Video Source extraction into the existing transcription pipeline. After the local video has been converted into an MP3 Audio Artifact, the CLI should pass that artifact to the selected Transcription Provider and create the final SRT using the same output path convention as the YouTube path.

This slice should preserve provider/model behavior, quiet output, final path reporting, and failure handling across both Voxtral and Grok.

## Acceptance criteria

- [ ] A Local Video Source can complete the full pipeline from local video path to MP3 Audio Artifact to SRT.
- [ ] The SRT path is derived from the generated MP3 path and includes the selected Transcription Provider name.
- [ ] `--provider voxtral` works with Local Video Sources.
- [ ] `--provider grok` works with Local Video Sources.
- [ ] `--model` is forwarded to the selected Transcription Provider for Local Video Sources.
- [ ] `--quiet` prints only the final SRT path for Local Video Sources.
- [ ] Local extraction failures return a non-zero exit code and do not call transcription.
- [ ] Transcription failures after local extraction return a non-zero exit code and do not print a misleading final path.
- [ ] App-level tests cover the full local orchestration path using fake extraction and transcription seams.

## Blocked by

- .scratch/local-video-source-to-srt/issues/02-extract-local-video-sources-to-mp3.md
