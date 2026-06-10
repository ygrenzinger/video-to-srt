# Video to SRT

This context covers the language for turning a Media Source into timestamped subtitles.

## Language

**Media Source**:
The single positional CLI argument that identifies the source media for transcription. A Media Source can be a YouTube Source or a Local Video Source.
_Avoid_: input, source input

**YouTube Source**:
A YouTube URL accepted by the CLI as a Media Source.
_Avoid_: input video, source file

**Local Video Source**:
A readable local video file accepted by the CLI as a Media Source. Supported extensions are `.mp4`, `.mov`, `.mkv`, `.webm`, `.avi`, and `.m4v`.
_Avoid_: local audio, arbitrary file, source file

**Transcription Provider**:
A service that turns a Media Source into timestamped Subtitle Cues.
_Avoid_: model, backend

**Voxtral**:
The first supported Transcription Provider, using Mistral's Voxtral transcription model family.
_Avoid_: Mistral provider

**Grok**:
A Transcription Provider using xAI's speech-to-text transcription service.
_Avoid_: xAI backend, Grok model

**Subtitle Cue**:
One timestamped text item in an SRT output.
_Avoid_: segment, line

**Target Language**:
The requested language for translated Subtitle Cues. Target Language values are simple language codes, not regional locale tags.
_Avoid_: locale, output language

**Translation Provider**:
A service that translates Subtitle Cue text while preserving cue timing.
_Avoid_: transcription provider, model, backend

**Subtitle Source**:
An existing SRT file accepted by the CLI for translation-only retry.
_Avoid_: Media Source, source file
