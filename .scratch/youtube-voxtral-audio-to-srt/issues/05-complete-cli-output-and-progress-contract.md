# Complete CLI output and progress contract

Status: ready-for-agent

## Parent

.scratch/youtube-voxtral-audio-to-srt/PRD.md

## What to build

Finalize the user-facing CLI behavior for successful and failed runs. The default mode should print concise human-readable stage progress to stderr, and `--quiet` should print only the final SRT path to stdout.

This slice should make the command script-friendly without adding JSONL or verbose logging, which are out of scope for v1.

## Acceptance criteria

- [ ] Default successful runs emit concise stage progress to stderr.
- [ ] Default successful runs report the final SRT path.
- [ ] `--quiet` successful runs print only the final SRT path to stdout.
- [ ] Failed runs exit non-zero and do not print a misleading final path.
- [ ] Unsupported provider, bad arguments, YouTube Source failures, and Voxtral failures have clear CLI-level errors.
- [ ] Tests cover stdout, stderr, and exit-code behavior for success and representative failures.

## Blocked by

- .scratch/youtube-voxtral-audio-to-srt/issues/03-add-voxtral-transcription-provider-happy-path.md
