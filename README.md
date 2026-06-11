# video-to-srt

`video-to-srt` is a small CLI that turns a Media Source into a SRT file.

It accepts a YouTube Source, Local Video Source, or Local Audio Source, prepares audio when needed, sends audio to a Transcription Provider, and writes timestamped Subtitle Cues as `.srt`.

Supported Transcription Providers:

- `voxtral` using Mistral's Voxtral transcription model family
- `grok` using xAI's speech-to-text transcription service

Supported Translation Providers:

- `mistral` using Mistral Large for Subtitle Cue translation
- `grok` using Grok for Subtitle Cue translation

## Quick Start

Install the CLI, set the API key for the provider you want to use, then pass one Media Source:

```sh
export MISTRAL_API_KEY='your-mistral-api-key'
video-to-srt 'https://www.youtube.com/watch?v=abc123'
```

For Grok transcription or translation, use `XAI_API_KEY`:

```sh
export XAI_API_KEY='your-xai-api-key'
video-to-srt --provider grok ./talk.final.mp3
```

Transcribe and translate a Media Source:

```sh
video-to-srt --target-language fr ./talk.final.mp4
```

Translate an existing Subtitle Source without re-transcribing:

```sh
video-to-srt translate --target-language fr ./talk.final.voxtral.srt
```

## Install With Homebrew

On macOS or Linux, install `video-to-srt` from the Homebrew tap:

```sh
brew tap ygrenzinger/tap
brew install video-to-srt
```

Or install it with the fully qualified formula name:

```sh
brew install ygrenzinger/tap/video-to-srt
```

The Homebrew formula also installs the external tools used to prepare Media Sources:

- `yt-dlp` for YouTube Sources
- `ffmpeg` for Local Video Sources

Check that the binary runs:

```sh
video-to-srt --version
```

## Install From GitHub Releases

Download the archive for your operating system and CPU from the project's GitHub Releases page, then extract the `video-to-srt` binary.

Check that the binary runs:

```sh
video-to-srt --version
```

When installing manually, also install the external tools needed for the Media Sources you want to use:

- YouTube Sources require `yt-dlp` on `PATH`.
- Local Video Sources require `ffmpeg` on `PATH`.
- Local Audio Sources do not require `yt-dlp` or `ffmpeg`.

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

Transcribe a Local Audio Source:

```sh
video-to-srt ./talk.final.mp3
```

Transcribe and translate a Media Source:

```sh
video-to-srt --target-language fr ./talk.final.mp4
```

Translate an existing Subtitle Source without re-transcribing:

```sh
video-to-srt translate --target-language fr ./talk.final.voxtral.srt
```

## Accepted Sources

The default command is `transcribe`, so these two commands are equivalent:

```sh
video-to-srt ./talk.final.mp4
video-to-srt transcribe ./talk.final.mp4
```

`transcribe` accepts exactly one Media Source. Translation-only retry uses the explicit `translate` command and accepts exactly one Subtitle Source.

Accepted Media Sources:

- YouTube Source: `http` or `https` URL with host `youtube.com`, `www.youtube.com`, `m.youtube.com`, or `youtu.be`.
- Local Video Source: readable local file with extension `.mp4`, `.mov`, `.mkv`, `.webm`, `.avi`, or `.m4v`.
- Local Audio Source: readable local file with extension `.mp3`, `.wav`, `.flac`, or `.ogg`.

Accepted Subtitle Source:

- Subtitle Source: readable local `.srt` file passed to `video-to-srt translate`. This requires `--target-language` and translates the existing Subtitle Cues without re-transcribing media.

Local file extension matching is case-insensitive. Current Media Source support does not include bare `.srt` translation shortcuts, non-YouTube HTTP media, directories, arbitrary file extensions, batch processing, recursive directory processing, or audio-only extraction mode.

Generated files are written to the current directory by default.

For YouTube Sources, the SRT file is named:

```text
<youtube title> [id].<provider>.srt
```

For Local Video Sources, the SRT file is named:

```text
<local basename>.<provider>.srt
```

For Local Audio Sources, the SRT file is named:

```text
<local basename>.<provider>.srt
```

When `--target-language` is used with a Media Source, both files are written:

```text
<basename>.<provider>.srt
<basename>.<provider>.<target-language>.srt
```

When `--target-language` is used with a Subtitle Source, the translated SRT is named:

```text
<subtitle source basename>.<target-language>.srt
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

With `--quiet` and Media Source translation, stdout contains two paths: the source-language SRT first and the translated SRT second. With `--quiet` and Subtitle Source translation, stdout contains only the translated SRT path.

Translate an existing Subtitle Source:

```sh
video-to-srt translate --target-language fr ./talk.final.voxtral.srt
```

For YouTube Sources that need browser cookies, pass them through to `yt-dlp`:

```sh
video-to-srt \
  --youtube-cookies-from-browser chrome \
  'https://www.youtube.com/watch?v=abc123'
```

Supported Target Language codes are `ar`, `bn`, `br`, `ca`, `cs`, `da`, `de`, `el`, `en`, `es`, `fa`, `fi`, `fr`, `gu`, `he`, `hi`, `hr`, `id`, `it`, `ja`, `kn`, `ko`, `lo`, `mr`, `ms`, `ne`, `nl`, `no`, `pl`, `pt`, `pa`, `ro`, `ru`, `sr`, `sv`, `ta`, `te`, `th`, `tl`, `tr`, `uk`, `ur`, `vi`, and `zh`.

Target Language values are product-defined language codes, not provider capability guarantees. Translation quality depends on the selected Translation Provider and model.

## Options

Usage:

```text
video-to-srt [transcribe] [options] <media-source>
video-to-srt translate [options] <subtitle-source.srt>
```

Commands:

```text
transcribe  Turn a Media Source into SRT subtitles. Optional by default.
translate   Translate an existing Subtitle Source. Required for .srt inputs.
```

Transcription options:

```text
--output-dir <dir>                  Directory for generated files. Defaults to the current directory.
--provider <voxtral|grok>           Transcription Provider. Defaults to voxtral.
--model <model-id>                  Provider-specific model id. Defaults to the provider default.
--target-language <code>            Translate Subtitle Cues to a supported Target Language code.
--translation-provider <mistral|grok>
                                    Translation Provider. Defaults to mistral for voxtral and grok for grok.
--translation-model <model-id>      Translation Provider model id. Defaults to the provider default.
--youtube-cookies <path>            Cookies file to pass to yt-dlp. Valid only for YouTube Sources.
--youtube-cookies-from-browser <id> Browser cookie store to pass to yt-dlp, such as chrome or firefox.
--quiet                             Print only generated SRT paths to stdout.
--version                           Print version and exit.
--help                              Print help and exit.
```

Translation-only options:

```text
--target-language <code>            Required. Translate Subtitle Cues to a supported Target Language code.
--translation-provider <mistral|grok>
                                    Translation Provider. Defaults to mistral.
--translation-model <model-id>      Translation Provider model id. Defaults to the provider default.
--quiet                             Print only the translated SRT path to stdout.
--help                              Print help and exit.
```

## Build And Run From Source

Requirements:

- Go 1.23 or newer
- `yt-dlp` available on `PATH` for YouTube Sources
- `ffmpeg` available on `PATH` for Local Video Sources
- Local Audio Sources can be used without `yt-dlp` or `ffmpeg`
- `MISTRAL_API_KEY` for Voxtral, or `XAI_API_KEY` for Grok
- `MISTRAL_API_KEY` for Mistral translation, or `XAI_API_KEY` for Grok translation

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
