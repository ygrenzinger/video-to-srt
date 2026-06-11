# PRD: Local Audio Source to SRT

Status: ready-for-agent

## Problem Statement

The user can create timestamped SRT files from a YouTube Source or a Local Video Source, but cannot pass an audio file already on disk as the Media Source. This forces users with podcasts, voice memos, exported meeting audio, or already-extracted tracks to wrap audio in a video container or run an unnecessary conversion step before using the CLI.

The user needs local audio files to behave as first-class Media Sources: provide one readable file, send it directly to the selected Transcription Provider, and receive an SRT without deleting or modifying the original audio file.

## Solution

Extend the CLI so its single positional argument accepts a Local Audio Source alongside the existing YouTube Source and Local Video Source. A Local Audio Source is a readable local audio file with one of the initial cross-provider safe extensions: `.mp3`, `.wav`, `.flac`, or `.ogg`.

When the Media Source is a Local Audio Source, the CLI passes the original file path directly to the selected Transcription Provider, writes the generated SRT under `--output-dir`, and does not remove the original audio file after transcription. YouTube Sources and Local Video Sources keep their existing temporary MP3 preparation and cleanup behavior.

## User Stories

1. As a CLI user, I want to pass an `.mp3` file directly, so that I can create subtitles from common audio exports.
2. As a CLI user, I want to pass a `.wav` file directly, so that I can transcribe high-quality raw audio captures.
3. As a CLI user, I want to pass a `.flac` file directly, so that I can transcribe lossless audio without converting it first.
4. As a CLI user, I want to pass an `.ogg` file directly, so that open audio formats are supported.
5. As a CLI user, I want local audio files to use the same single positional argument as other Media Sources, so that the command shape stays simple.
6. As a CLI user, I want the CLI to recognize Local Audio Sources separately from Local Video Sources, so that the tool does not require video extraction for audio-only files.
7. As a CLI user, I want direct local audio transcription, so that I avoid unnecessary ffmpeg conversion work.
8. As a CLI user, I want the original audio file passed to the Transcription Provider, so that the provider receives the file I chose.
9. As a CLI user, I want the original audio file preserved after successful transcription, so that the CLI never deletes user-owned media.
10. As a CLI user, I want the original audio file preserved after failed transcription, so that provider errors do not cause data loss.
11. As a CLI user, I want SRT output names based on the audio basename, so that generated files are easy to identify.
12. As a CLI user, I want Local Audio Source outputs to honor `--output-dir`, so that generated SRT files land where I asked.
13. As a CLI user, I want the SRT filename to include the selected Transcription Provider, so that Voxtral and Grok outputs can coexist.
14. As a CLI user, I want `--provider voxtral` to work with Local Audio Sources, so that the default provider supports audio files.
15. As a CLI user, I want `--provider grok` to work with Local Audio Sources, so that provider selection stays independent of source type.
16. As a CLI user, I want `--model` to pass through for Local Audio Sources, so that provider-specific model overrides continue to work.
17. As a CLI user, I want `--target-language` to work with Local Audio Sources, so that I can transcribe and translate audio in one command.
18. As a CLI user, I want `--translation-provider` and `--translation-model` to work after Local Audio Source transcription, so that translation options remain consistent.
19. As a scripting user, I want `--quiet` to print only generated SRT paths for Local Audio Sources, so that automation can consume the result reliably.
20. As a scripting user, I want failed Local Audio Source validation to return a non-zero exit code, so that scripts can stop early.
21. As a scripting user, I want transcription failures for Local Audio Sources to behave like other Media Source transcription failures, so that error handling is consistent.
22. As a CLI user, I want missing local audio files rejected before transcription, so that I get a clear local file error.
23. As a CLI user, I want directories rejected as Local Audio Sources, so that invalid paths are not sent to providers.
24. As a CLI user, I want unsupported audio extensions rejected clearly, so that I understand which local audio files are accepted.
25. As a CLI user, I want YouTube cookie flags rejected for Local Audio Sources, so that source-specific options do not hide mistakes.
26. As a CLI user, I want YouTube Source behavior unchanged, so that existing URL workflows do not regress.
27. As a CLI user, I want Local Video Source behavior unchanged, so that video extraction and cleanup continue to work.
28. As a CLI user, I want Subtitle Source translation behavior unchanged, so that existing SRT retry workflows do not regress.
29. As a maintainer, I want `Media Source` language to include Local Audio Source, so that the domain model matches supported inputs.
30. As a maintainer, I want direct audio support tested through the CLI orchestration seam, so that behavior is verified at the user-facing boundary.
31. As a maintainer, I want cleanup ownership covered by tests, so that future changes do not delete user-owned Local Audio Sources.
32. As a maintainer, I want provider code reused unchanged for direct audio files, so that source support does not duplicate Transcription Provider logic.
33. As a maintainer, I want the accepted Local Audio Source extension set kept intentionally small, so that the first release stays within cross-provider support.
34. As a maintainer, I want user-facing docs updated, so that users know audio files are accepted and ffmpeg is not required for them.

## Implementation Decisions

- Add `Local Audio Source` as a domain term and concrete Media Source type, sibling to `Local Video Source`.
- A Local Audio Source must be an existing non-directory local file with one of these extensions: `.mp3`, `.wav`, `.flac`, `.ogg`.
- The initial Local Audio Source extension set is the cross-provider safe set supported by both current Transcription Providers.
- Local Audio Sources are passed directly to the Transcription Provider without copying or transcoding.
- Local Audio Sources are user-owned files and must never be removed by post-transcription cleanup.
- Temporary audio cleanup remains only for artifacts produced from YouTube Sources and Local Video Sources.
- Local Audio Source SRT output is written under `--output-dir` using the audio basename and selected Transcription Provider name.
- The existing Transcription Provider flow is reused. This feature does not add provider-specific source branches.
- YouTube cookie options remain valid only for YouTube Sources and must be rejected for Local Audio Sources.
- Translation options continue to operate on the generated source-language Subtitle Cues after transcription.
- User-facing documentation should describe Local Audio Source support, accepted extensions, output naming, and dependency requirements.
- No ADR is required because the feature is an additive Media Source capability and does not introduce a hard-to-reverse architectural tradeoff.

## Testing Decisions

- Use the CLI orchestration seam only for this feature: app-level tests should exercise the public command behavior through fake runner functions.
- Good tests should assert observable behavior: accepted inputs, rejected inputs, provider request paths, output paths, quiet output, translation continuation, and cleanup ownership.
- Tests should not call real Transcription Provider APIs, real `yt-dlp`, or real `ffmpeg`.
- Prior art is the existing app-level runner tests that fake source preparation, transcription, and translation.
- Add tests proving `.mp3`, `.wav`, `.flac`, and `.ogg` are accepted as Local Audio Sources.
- Add tests proving direct transcription receives the original audio path.
- Add tests proving Local Audio Source output respects `--output-dir`.
- Add tests proving Local Audio Sources are not deleted after successful transcription.
- Add tests proving Local Audio Sources are not deleted after failed transcription.
- Add tests proving YouTube cookie flags are rejected for Local Audio Sources.
- Add tests proving unsupported local extensions still fail with a clear Media Source validation error.
- Existing regression tests for YouTube Sources, Local Video Sources, Subtitle Sources, providers, and translation should continue to pass.
- The final regression command should be `go test ./...`.

## Out of Scope

- Provider-specific extra formats such as `.aac`, `.m4a`, `.opus`, `.webm`, `.mp4`, and `.mkv` as Local Audio Sources.
- Raw audio formats that require explicit `audio_format` or sample-rate metadata.
- Audio URL inputs.
- Non-YouTube HTTP media inputs.
- Batch processing multiple Media Sources.
- Recursive directory processing.
- Copying Local Audio Sources into `--output-dir`.
- Transcoding Local Audio Sources to MP3.
- Custom accepted extension configuration.
- Provider-side size or duration preflight validation.
- New Transcription Providers.

## Further Notes

Current provider documentation was checked before setting the initial extension set. Voxtral supports WAV, MP3, FLAC, OGG, and WEBM with duration and file-size limits. Grok supports the selected cross-provider set plus additional container and raw formats. This PRD intentionally starts with the shared safe set to avoid provider-specific validation in the first Local Audio Source release.
