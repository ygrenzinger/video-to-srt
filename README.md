# video-to-srt

`video-to-srt` is a Go CLI that turns a YouTube Source into an MP3 Audio Artifact and a timestamped SRT file using Voxtral or Grok.

## Requirements

- Go 1.23 or newer
- `yt-dlp` available on `PATH`
- `MISTRAL_API_KEY` set in the environment for Voxtral, or `XAI_API_KEY` for Grok

## Usage

```sh
go run ./cmd/video-to-srt 'https://www.youtube.com/watch?v=abc123'
```

By default, generated files are written to the current directory:

- Audio Artifact: `<youtube title> [id].mp3`
- SRT: `<youtube title> [id].<provider>.srt`

The command prints concise progress to stderr and reports the final SRT path when it succeeds.

## Options

```sh
go run ./cmd/video-to-srt \
  --output-dir ./out \
  --provider grok \
  --youtube-cookies-from-browser chrome \
  'https://www.youtube.com/watch?v=abc123'
```

- `--output-dir`: directory for generated files; defaults to the current directory.
- `--provider`: Transcription Provider; `voxtral` by default, or `grok`.
- `--model`: provider-specific model id; defaults to the selected provider's default model.
- `--youtube-cookies`: exported cookies file to pass to `yt-dlp`.
- `--youtube-cookies-from-browser`: browser cookie store to pass to `yt-dlp`, such as `chrome` or `firefox`.
- `--quiet`: print only the final SRT path to stdout.

## Scope

V1 accepts YouTube Sources only. Local files, non-YouTube HTTP media, JSONL logging, subtitle improvement, and language hints are out of scope.

The SRT uses timestamped cues returned by the selected Transcription Provider. If the provider returns text without usable timestamps, the command fails instead of inventing approximate Subtitle Cue timing.
