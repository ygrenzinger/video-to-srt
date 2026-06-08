# Create the Go CLI skeleton and YouTube Source to MP3 path

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Build the first runnable vertical slice of `video-to-srt`: a Go command that accepts exactly one YouTube Source, validates that it is supported, invokes `yt-dlp` to create an MP3 Audio Artifact directly, and reports the resulting MP3 path.

This slice should establish the repo's Go module, command entrypoint, basic app orchestration seam, and YouTube source handling seam. It should support the v1 output directory and YouTube cookie flags, but it does not need to transcribe or write SRT yet.

## Acceptance criteria

- [ ] `video-to-srt <youtube-url>` runs through the YouTube Source to MP3 path and exits successfully when `yt-dlp` reports an MP3 file.
- [ ] The command rejects missing arguments, extra arguments, local paths, and non-YouTube HTTP URLs with clear non-zero errors.
- [ ] `--output-dir` defaults to the current directory and is passed to `yt-dlp` when provided.
- [ ] `--youtube-cookies` and `--youtube-cookies-from-browser` are passed through to `yt-dlp` only when provided.
- [ ] Missing `yt-dlp` and failing `yt-dlp` commands surface clear errors, including useful downloader diagnostics.
- [ ] Tests use fake process execution or fake binaries and do not call real YouTube.

## Blocked by

None - can start immediately
