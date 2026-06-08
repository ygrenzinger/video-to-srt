# Preserve and scope YouTube-specific behavior

Status: ready-for-agent

## Parent

.scratch/local-video-source-to-srt/PRD.md

## What to build

Keep the existing YouTube Source path working unchanged while making YouTube-only options invalid for Local Video Sources. YouTube cookie options should continue to pass through to `yt-dlp` for YouTube Sources, but should fail fast when combined with a Local Video Source.

This slice protects existing behavior and prevents source-specific flags from silently doing nothing.

## Acceptance criteria

- [ ] Existing YouTube URL commands still download an MP3 Audio Artifact through the YouTube source path.
- [ ] `--youtube-cookies` is still passed through for YouTube Sources.
- [ ] `--youtube-cookies-from-browser` is still passed through for YouTube Sources.
- [ ] `--youtube-cookies` is rejected for Local Video Sources with a clear error.
- [ ] `--youtube-cookies-from-browser` is rejected for Local Video Sources with a clear error.
- [ ] Rejected local cookie-flag combinations do not call extraction or transcription.
- [ ] Regression tests cover YouTube cookie pass-through and local cookie-flag rejection.

## Blocked by

- .scratch/local-video-source-to-srt/issues/01-add-media-source-classification-and-glossary.md
