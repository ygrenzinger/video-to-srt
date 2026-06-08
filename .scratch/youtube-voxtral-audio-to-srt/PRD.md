# PRD: YouTube to SRT with Voxtral

Status: ready-for-agent

## Problem Statement

The user needs a focused Go CLI that turns a YouTube Source into a reusable MP3 Audio Artifact and a timestamped SRT file. The repo is empty, so the feature also needs to establish a clean Go architecture that can support future Transcription Providers without overfitting v1.

The first useful workflow is: provide a YouTube URL, let `yt-dlp` create the MP3, send that MP3 to Voxtral, and receive a valid SRT whose Subtitle Cues use provider timestamps rather than guessed timings.

## Solution

Build a Go command named `video-to-srt`. It accepts one YouTube URL, downloads/extracts an MP3 directly through `yt-dlp`, transcribes the MP3 with Voxtral through Mistral's audio transcription API, and writes both outputs in the current directory by default.

The CLI uses `voxtral` as the canonical provider name, defaults to model `voxtral-mini-latest`, and requests segment-level timestamps for SRT generation. It exposes a small set of practical flags for output placement, model override, YouTube cookies, and quiet scripting output.

## User Stories

1. As a CLI user, I want to pass a YouTube URL, so that I can create subtitles without manually downloading media first.
2. As a CLI user, I want the tool to create an MP3 Audio Artifact, so that I can reuse or inspect the audio separately from the SRT.
3. As a CLI user, I want the tool to write a timestamped SRT, so that I can load subtitles into video players and editing tools.
4. As a CLI user, I want Subtitle Cues to use Voxtral timestamps, so that cue timings are based on provider output rather than rough estimates.
5. As a CLI user, I want the default output location to be the current directory, so that a simple command creates visible artifacts where I ran it.
6. As a CLI user, I want an optional output directory, so that scripts can place generated artifacts in a predictable location.
7. As a CLI user, I want output names based on the YouTube title and id, so that files are recognizable and stable enough for local use.
8. As a CLI user, I want the SRT filename to include `voxtral`, so that I can tell which Transcription Provider generated it.
9. As a CLI user, I want to use browser cookies with YouTube, so that videos blocked for anonymous downloads can still be processed.
10. As a CLI user, I want to use an exported cookies file with YouTube, so that I can run the tool in environments without browser access.
11. As a CLI user, I want a clear error when `yt-dlp` is missing, so that I know which dependency to install.
12. As a CLI user, I want a clear error when `MISTRAL_API_KEY` is missing, so that I know how to configure transcription.
13. As a CLI user, I want a model override flag, so that I can choose a different Voxtral model when needed.
14. As a CLI user, I want concise progress on stderr, so that I can see which stage is running without parsing noisy logs.
15. As a scripting user, I want `--quiet` to print only the final SRT path, so that shell scripts can consume the result reliably.
16. As a scripting user, I want provider failures to return non-zero exit codes, so that automation can stop on failed transcriptions.
17. As a scripting user, I want YouTube download failures to surface `yt-dlp` diagnostics, so that I can debug blocked or unavailable videos.
18. As a developer, I want source download, transcription, subtitle writing, and orchestration separated by capability, so that each part can be tested independently.
19. As a developer, I want the Voxtral adapter hidden behind a Transcription Provider contract, so that future providers can be added without rewriting the CLI.
20. As a developer, I want external commands and HTTP clients behind seams, so that tests do not require real YouTube or Mistral calls.
21. As a developer, I want transient Voxtral failures retried, so that short-lived network or rate-limit errors do not fail the whole run immediately.
22. As a developer, I want `yt-dlp` failures not retried by the app, so that long downloads are not repeated and the original diagnostic remains visible.
23. As a developer, I want invalid or missing timestamp segments to fail the run, so that the tool does not produce misleading SRT timing.
24. As a maintainer, I want a domain glossary, so that future work uses consistent terms for YouTube Sources, Audio Artifacts, Transcription Providers, and Subtitle Cues.

## Implementation Decisions

- Create a Go module and command both named `video-to-srt`.
- Accept exactly one positional argument, and treat it as a YouTube Source. Local files and non-YouTube HTTP URLs are out of scope for v1.
- Use `yt-dlp` direct audio extraction to create MP3 output, not a video download followed by local conversion.
- Support `--output-dir`, defaulting to the current directory.
- Support `--provider`, defaulting to `voxtral`, with no other providers accepted in v1.
- Support `--model`, defaulting to `voxtral-mini-latest`.
- Support `--youtube-cookies` and `--youtube-cookies-from-browser`, passing them through to `yt-dlp`.
- Support `--quiet`; otherwise print concise human-readable stage progress to stderr.
- Require `MISTRAL_API_KEY` for Voxtral transcription.
- Call Mistral with direct multipart HTTP rather than an SDK.
- Request segment timestamp granularity from Voxtral and convert returned segments into Subtitle Cues.
- Do not expose a language flag in v1 because the timestamp requirement is more important than language hints.
- Do not fabricate timestamps. If Voxtral returns text without usable timestamped segments, fail clearly.
- Retry only transient Voxtral failures: network errors, rate limits, and server errors.
- Keep the architecture as capability modules, not a formal layered framework. The core capabilities are orchestration, YouTube source handling, transcription, Voxtral, and subtitles.
- Use atomic SRT writes so failed formatting or write errors do not leave partial final output.

## Testing Decisions

- Test external behavior at the highest practical seam: CLI/app orchestration for end-to-end flow, and adapter-level tests for `yt-dlp` and Voxtral boundaries.
- YouTube source tests should cover URL acceptance, unsupported input rejection, missing `yt-dlp`, cookie propagation, output directory behavior, and parsing the reported MP3 path.
- Voxtral tests should use an HTTP test server to verify auth, multipart payload shape, model selection, timestamp granularity, retry behavior, malformed responses, and missing segment handling.
- Subtitle tests should cover timestamp formatting, cue indexing, final newline, invalid timestamps, and atomic write behavior.
- App orchestration tests should use fake source and fake Transcription Provider implementations to verify stage order, output paths, quiet output, and failure short-circuiting.
- Tests should avoid real YouTube, real Mistral, and real network dependencies.

## Out of Scope

- Local video or audio file inputs.
- Non-YouTube HTTP media inputs.
- Subtitle improvement or readability rewriting.
- Multiple Transcription Providers beyond Voxtral.
- JSONL logging or verbose legacy progress output.
- Language hints for transcription.
- Download retry policy outside the behavior already provided by `yt-dlp`.
- Approximate SRT timing fallback when provider timestamps are unavailable.
- GUI, web app, queueing, or batch processing.

## Further Notes

The project starts from an empty repo. A sibling implementation exists, but this PRD intentionally narrows v1 to the requested YouTube-to-MP3-to-Voxtral workflow instead of copying the full broader tool.

The first implementation should prioritize a small working vertical slice with strong seams around process execution and HTTP. That gives the next feature, such as local file support or another Transcription Provider, a clean place to attach without changing the CLI contract unnecessarily.
