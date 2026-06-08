# PRD: Local Video Source to SRT

Status: ready-for-agent

## Problem Statement

The user can currently create an MP3 Audio Artifact and timestamped SRT from a YouTube Source, but cannot run the same workflow against a video file already on disk. This forces users with local recordings, downloaded videos, screen captures, or conference talks to manually extract audio before using the CLI.

The user needs local video files to behave like the existing YouTube path: provide one Media Source, let the CLI create the MP3 Audio Artifact, then transcribe that artifact with the selected Transcription Provider into an SRT.

## Solution

Extend the CLI so its single positional argument accepts a Media Source: either an existing YouTube Source or a Local Video Source. A Local Video Source is an existing local video file with an accepted video extension.

When the Media Source is local, the CLI extracts audio to MP3 with `ffmpeg`, writes the MP3 Audio Artifact into the selected output directory, then sends that artifact through the existing Voxtral or Grok transcription flow. The SRT path remains derived from the MP3 path and includes the selected Transcription Provider name.

## User Stories

1. As a CLI user, I want to pass a local video file path, so that I can create subtitles from media I already have on disk.
2. As a CLI user, I want local video files to use the same command shape as YouTube URLs, so that I do not need to learn a separate mode.
3. As a CLI user, I want the CLI to create an MP3 Audio Artifact from a local video file, so that transcription providers receive the same kind of input as the YouTube path.
4. As a CLI user, I want the CLI to create an SRT after extracting local audio, so that local video processing is a complete subtitle workflow.
5. As a CLI user, I want local video outputs to honor `--output-dir`, so that generated files land in a predictable location.
6. As a CLI user, I want local output filenames based on the source basename, so that `talk.final.mp4` produces recognizable generated files.
7. As a CLI user, I want the local MP3 output to be named predictably, so that rerunning and scripting are simple.
8. As a CLI user, I want the local SRT output to include the Transcription Provider name, so that I can distinguish Voxtral and Grok results.
9. As a CLI user, I want local generated files to be overwritten on rerun, so that I can regenerate outputs without manual cleanup.
10. As a CLI user, I want `.mp4` files accepted, so that common video recordings work.
11. As a CLI user, I want `.mov` files accepted, so that macOS and camera recordings work.
12. As a CLI user, I want `.mkv` files accepted, so that common archived video files work.
13. As a CLI user, I want `.webm` files accepted, so that browser-recorded media works.
14. As a CLI user, I want `.avi` files accepted, so that older video files can still be processed.
15. As a CLI user, I want `.m4v` files accepted, so that Apple-style video files can be processed.
16. As a CLI user, I want unsupported local extensions rejected clearly, so that I know the CLI did not silently ignore my source.
17. As a CLI user, I want missing local files rejected before extraction, so that I get a direct file error instead of a confusing transcription failure.
18. As a CLI user, I want directories rejected as Local Video Sources, so that I do not accidentally ask the CLI to process an invalid source.
19. As a CLI user, I want a clear error when `ffmpeg` is missing, so that I know which dependency to install for local video processing.
20. As a CLI user, I want `ffmpeg` extraction errors surfaced clearly, so that I can diagnose unsupported codecs or corrupt files.
21. As a CLI user, I want YouTube Sources to keep working unchanged, so that existing commands do not regress.
22. As a CLI user, I want YouTube cookie flags to remain available for YouTube Sources, so that blocked YouTube downloads still work.
23. As a CLI user, I want YouTube cookie flags rejected for Local Video Sources, so that source-specific options do not hide mistakes.
24. As a CLI user, I want `--provider grok` to work with Local Video Sources, so that provider selection is independent of source type.
25. As a CLI user, I want `--provider voxtral` to work with Local Video Sources, so that the default provider supports both source types.
26. As a CLI user, I want `--model` to pass through for local video transcription, so that provider-specific model overrides continue to work.
27. As a scripting user, I want `--quiet` to print only the final SRT path for Local Video Sources, so that scripts can consume the result reliably.
28. As a scripting user, I want local extraction failures to return a non-zero exit code, so that automation can stop on bad media.
29. As a scripting user, I want transcription failures after local extraction to behave like YouTube transcription failures, so that error handling is consistent.
30. As a maintainer, I want the domain language to describe Media Sources, so that future features do not keep treating YouTube as the only possible source.
31. As a maintainer, I want Local Video Source extraction behind a testable seam, so that tests do not require real media processing.
32. As a maintainer, I want source-type detection covered at the app boundary, so that invalid inputs are rejected before external tools run.
33. As a maintainer, I want the `ffmpeg` adapter tested separately from orchestration, so that command construction and failure handling can be verified precisely.
34. As a maintainer, I want the existing transcription adapter contracts reused, so that local file support does not duplicate provider logic.

## Implementation Decisions

- Introduce `Media Source` as the general term for the single CLI input, with `YouTube Source` and `Local Video Source` as concrete source types.
- Keep one positional argument for the CLI. The argument is classified as a YouTube Source when it is an accepted YouTube URL; otherwise it may be a Local Video Source.
- A Local Video Source must be an existing non-directory local file with one of these extensions: `.mp4`, `.mov`, `.mkv`, `.webm`, `.avi`, `.m4v`.
- Non-YouTube HTTP URLs remain unsupported.
- YouTube Sources continue to use `yt-dlp` to produce an MP3 Audio Artifact.
- Local Video Sources use `ffmpeg` to produce an MP3 Audio Artifact.
- Local extraction uses overwrite behavior so reruns replace generated MP3 output.
- Local extraction uses VBR quality 2 MP3 settings through `libmp3lame`, optimized for a practical speech quality and size tradeoff.
- Local MP3 output is written to `--output-dir` and named from the local video basename with the extension replaced by `.mp3`.
- The SRT output path is still derived from the MP3 Audio Artifact path and includes the selected Transcription Provider.
- The existing Transcription Provider flow is reused after MP3 creation. Local source support does not add provider-specific branches.
- YouTube cookie options are valid only for YouTube Sources and must be rejected for Local Video Sources.
- No audio-only extraction mode is added. MP3 extraction is part of the subtitle creation pipeline.
- The README should update requirements, usage, options, and scope to reflect Local Video Source support.
- The domain glossary should be updated to avoid YouTube-only terminology for the whole workflow.
- No ADR is required because the decision is a reversible extension of the existing source pipeline and does not introduce a hard-to-reverse architectural tradeoff.

## Testing Decisions

- Prefer the highest practical seam: app-level orchestration tests should validate source classification, option validation, output path behavior, provider forwarding, quiet output, and failure short-circuiting.
- Existing orchestration tests with fake source and fake Transcription Provider functions are the prior art for local source behavior.
- Source-adapter tests should validate exact external command arguments for local MP3 extraction, missing dependency errors, and failed process diagnostics.
- Tests should verify external behavior and contracts, not internal helper function details.
- YouTube regression tests should verify existing URL acceptance, cookie propagation, output path derivation, and provider forwarding remain unchanged.
- Local validation tests should cover accepted extensions, unsupported extensions, missing files, and directories.
- Local option tests should verify YouTube cookie flags are rejected for Local Video Sources.
- Local output tests should verify `talk.final.mp4` becomes `talk.final.mp3` and `talk.final.<provider>.srt` in the chosen output directory.
- Quiet mode tests should verify only the final SRT path is printed for Local Video Sources.
- Adapter tests should use fake process runners and must not require real `ffmpeg`.
- Integration-style tests should not require real YouTube, real local media decoding, or real Transcription Provider API calls.
- The final regression command should be `go test ./...`.

## Out of Scope

- Local audio files as direct inputs.
- Non-YouTube HTTP media inputs.
- Batch processing multiple Media Sources.
- Recursive directory processing.
- Audio-only extraction mode.
- Custom MP3 bitrate or quality flags.
- Custom accepted extension configuration.
- Codec probing beyond extension validation and `ffmpeg` failure reporting.
- Subtitle improvement or readability rewriting.
- Approximate Subtitle Cue timing fallback when providers return unusable timestamps.
- New Transcription Providers.
- GUI or web app support.

## Further Notes

This PRD updates the previous YouTube-only scope by making the source concept explicit. The implementation should preserve the current YouTube behavior while adding the Local Video Source branch only up to the point where an MP3 Audio Artifact exists. From that point onward, transcription and SRT generation should behave exactly like the existing path.
