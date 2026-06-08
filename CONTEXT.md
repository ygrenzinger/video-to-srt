# Video to SRT

This context covers the language for turning a YouTube source into an audio artifact and timestamped subtitles.

## Language

**YouTube Source**:
A YouTube URL accepted by the CLI as the source media for transcription.
_Avoid_: input video, source file

**Audio Artifact**:
The MP3 file created from a YouTube Source and used as the transcription input.
_Avoid_: downloaded video, audio source

**Transcription Provider**:
A service that turns an Audio Artifact into timestamped Subtitle Cues.
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
