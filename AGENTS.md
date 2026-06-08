# AGENTS.md

Guidance for OpenCode, Codex, and other coding agents working in this repo.

## Project

`video-to-srt` is a Go CLI that turns a Media Source into a SRT file using a Transcription Provider.

## Before Changing Code

- Read `CONTEXT.md` and use its domain terms exactly.
- Check `docs/adr/` if the change touches architecture or provider behavior.
- Keep changes scoped; do not rewrite unrelated code or generated artifacts.
- Preserve user changes in the worktree.

## Commands

- Test: `go test ./...`
- Run: `go run ./cmd/video-to-srt <media-source>`

## Git

- Use Conventional Commits for commit messages.

## Repo Notes

- CLI entrypoint: `cmd/video-to-srt/main.go`
- Core orchestration: `internal/app/`
- Media Source handling: `internal/source/`
- Subtitle formatting: `internal/subtitles/`
- Transcription Providers: `internal/transcription/`

## Issues And PRDs

Local issue tracking lives under `.scratch/`.

- PRDs: `.scratch/<feature-slug>/PRD.md`
- Issues: `.scratch/<feature-slug>/issues/<NN>-<slug>.md`
- Triage labels follow `docs/agents/triage-labels.md`

See `docs/agents/issue-tracker.md` and `docs/agents/domain.md` for details.
