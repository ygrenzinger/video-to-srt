# video-to-srt

`video-to-srt` is a small CLI that turns a Media Source into a SRT file.

It accepts either a YouTube Source or a Local Video Source, prepares audio from it, sends that audio to a Transcription Provider, and writes timestamped Subtitle Cues as `.srt`.

Supported Transcription Providers:

- `voxtral` using Mistral's Voxtral transcription model family
- `grok` using xAI's speech-to-text transcription service

## Use A Released Version

Download the archive for your operating system and CPU from the project's GitHub Releases page, then extract the `video-to-srt` binary.

Check that the binary runs:

```sh
video-to-srt --version
```

Install the external tools needed for the Media Sources you want to use:

- YouTube Sources require `yt-dlp` on `PATH`.
- Local Video Sources require `ffmpeg` on `PATH`.

Set an API key for the Transcription Provider you want to use:

```sh
export MISTRAL_API_KEY='your-mistral-api-key'
```

Or, for Grok:

```sh
export XAI_API_KEY='your-xai-api-key'
```

Transcribe a YouTube Source:

```sh
video-to-srt 'https://www.youtube.com/watch?v=abc123'
```

Transcribe a Local Video Source:

```sh
video-to-srt ./talk.final.mp4
```

Generated files are written to the current directory by default.

For YouTube Sources, the SRT file is named:

```text
<youtube title> [id].<provider>.srt
```

For Local Video Sources, the SRT file is named:

```text
<local basename>.<provider>.srt
```

Use `--output-dir` to write generated files somewhere else:

```sh
video-to-srt --output-dir ./out ./talk.final.mp4
```

Use `--provider` to choose a Transcription Provider:

```sh
video-to-srt --provider grok ./talk.final.mp4
```

Use `--quiet` when you only want the final SRT path on stdout:

```sh
video-to-srt --quiet ./talk.final.mp4
```

For YouTube Sources that need browser cookies, pass them through to `yt-dlp`:

```sh
video-to-srt \
  --youtube-cookies-from-browser chrome \
  'https://www.youtube.com/watch?v=abc123'
```

Supported Local Video Source extensions are `.mp4`, `.mov`, `.mkv`, `.webm`, `.avi`, and `.m4v`.

## Options

```text
--output-dir <dir>                  Directory for generated files. Defaults to the current directory.
--provider <voxtral|grok>           Transcription Provider. Defaults to voxtral.
--model <model-id>                  Provider-specific model id. Defaults to the provider default.
--youtube-cookies <path>            Cookies file to pass to yt-dlp. Valid only for YouTube Sources.
--youtube-cookies-from-browser <id> Browser cookie store to pass to yt-dlp, such as chrome or firefox.
--quiet                             Print only the final SRT path to stdout.
--version                           Print version and exit.
```

## Build And Run From Source

Requirements:

- Go 1.23 or newer
- `yt-dlp` available on `PATH` for YouTube Sources
- `ffmpeg` available on `PATH` for Local Video Sources
- `MISTRAL_API_KEY` for Voxtral, or `XAI_API_KEY` for Grok

Clone the repository, then run the CLI directly from source:

```sh
go run ./cmd/video-to-srt 'https://www.youtube.com/watch?v=abc123'
```

Build a local binary:

```sh
go build -o video-to-srt ./cmd/video-to-srt
```

Run the local binary:

```sh
./video-to-srt ./talk.final.mp4
```

Run the test suite:

```sh
go test ./...
```
