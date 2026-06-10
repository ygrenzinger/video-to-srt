# Separate translation from transcription

Translation is a transformation of Subtitle Cues, not part of Media Source transcription, so `video-to-srt` uses a separate Translation Provider and keeps Media Source scoped to source media. Existing SRT files are accepted as Subtitle Sources for translation-only retry instead of expanding Media Source, which preserves the domain boundary between media-for-transcription and subtitles-for-translation.
