# Update user-facing docs and full regression coverage

Status: ready-for-agent

## Parent

.scratch/local-video-source-to-srt/PRD.md

## What to build

Update user-facing documentation so the CLI is documented as accepting a Media Source, including both YouTube Sources and Local Video Sources. The docs should explain local video requirements, accepted extensions, output naming, YouTube-only cookie flags, and the current scope boundaries.

This slice should also complete regression coverage for accepted local video extensions and run the full Go test suite.

## Acceptance criteria

- [ ] README usage shows both YouTube URL and local video examples.
- [ ] README requirements mention `ffmpeg` for Local Video Sources and `yt-dlp` for YouTube Sources.
- [ ] README documents accepted local video extensions: `.mp4`, `.mov`, `.mkv`, `.webm`, `.avi`, `.m4v`.
- [ ] README documents local output naming from source basename.
- [ ] README documents that YouTube cookie flags apply only to YouTube Sources.
- [ ] README scope excludes local audio files, non-YouTube HTTP media, batch processing, and audio-only extraction mode.
- [ ] Tests cover each accepted local video extension at the app validation boundary.
- [ ] `go test ./...` passes.

## Blocked by

- .scratch/local-video-source-to-srt/issues/02-extract-local-video-sources-to-mp3.md
- .scratch/local-video-source-to-srt/issues/03-run-local-video-sources-through-transcription-to-srt.md
- .scratch/local-video-source-to-srt/issues/04-preserve-and-scope-youtube-specific-behavior.md
