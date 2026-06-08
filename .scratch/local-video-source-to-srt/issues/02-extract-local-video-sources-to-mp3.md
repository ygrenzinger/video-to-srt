# Extract Local Video Sources to MP3

Status: ready-for-agent

## Parent

.scratch/local-video-source-to-srt/PRD.md

## What to build

Add the local extraction path that turns a Local Video Source into an MP3 Audio Artifact. When the user provides a valid local video file, the CLI should use `ffmpeg` to write an MP3 into the selected output directory using the local source basename, replacing the source extension with `.mp3`.

The extraction behavior should be testable without real media files or a real `ffmpeg` process, following the existing fake-runner pattern used for external command adapters.

## Acceptance criteria

- [ ] Local extraction requires `ffmpeg` on `PATH` and reports a clear dependency error when it is missing.
- [ ] Local extraction writes the MP3 Audio Artifact to `--output-dir`.
- [ ] A source named `talk.final.mp4` produces an MP3 named `talk.final.mp3`.
- [ ] Local extraction overwrites generated MP3 output on rerun.
- [ ] Local extraction uses VBR quality 2 MP3 settings through `libmp3lame`.
- [ ] Failed `ffmpeg` execution surfaces useful stderr or stdout diagnostics.
- [ ] Source-adapter tests verify the external command name and arguments exactly.
- [ ] App-level tests verify that a valid Local Video Source reaches extraction and returns the generated MP3 path to orchestration.

## Blocked by

- .scratch/local-video-source-to-srt/issues/01-add-media-source-classification-and-glossary.md
