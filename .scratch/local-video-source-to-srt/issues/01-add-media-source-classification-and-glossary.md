# Add Media Source classification and glossary

Status: ready-for-agent

## Parent

.scratch/local-video-source-to-srt/PRD.md

## What to build

Update the product language and app boundary so the single CLI argument is treated as a Media Source rather than always as a YouTube Source. The CLI should continue accepting YouTube URLs while also recognizing valid Local Video Source paths and rejecting invalid local inputs before any external extraction or transcription work starts.

This slice should make the source classification behavior visible through app-level tests and update the domain glossary so future work uses `Media Source`, `YouTube Source`, `Local Video Source`, `Audio Artifact`, `Transcription Provider`, and `Subtitle Cue` consistently.

## Acceptance criteria

- [ ] The glossary defines `Media Source` as the general CLI input concept and `Local Video Source` as a local video file accepted for transcription.
- [ ] The glossary no longer describes the whole workflow as YouTube-only.
- [ ] A YouTube URL is still accepted as a YouTube Source.
- [ ] A local path with an accepted extension is recognized as a Local Video Source when the file exists and is not a directory.
- [ ] Local paths with unsupported extensions are rejected with a clear error before extraction.
- [ ] Missing local files are rejected with a clear error before extraction.
- [ ] Directory paths are rejected with a clear error before extraction.
- [ ] Non-YouTube HTTP URLs remain unsupported.
- [ ] App-level tests cover source classification and invalid-input rejection without calling real external tools.

## Blocked by

None - can start immediately
