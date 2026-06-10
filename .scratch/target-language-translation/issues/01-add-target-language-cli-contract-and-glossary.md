# Add Target Language CLI contract and glossary

Status: ready-for-agent

## Parent

.scratch/target-language-translation/PRD.md

## What to build

Add the target-language CLI contract without changing existing no-translation behavior. The CLI should accept a product-defined `Target Language` code when translation is requested, reject unsupported or regional language values before provider work starts, and introduce the domain language needed by later translation slices.

## Acceptance criteria

- [ ] `--target-language` is accepted by the CLI.
- [ ] The accepted Target Language codes are `ar`, `bn`, `br`, `ca`, `cs`, `da`, `de`, `el`, `en`, `es`, `fa`, `fi`, `fr`, `gu`, `he`, `hi`, `hr`, `id`, `it`, `ja`, `kn`, `ko`, `lo`, `mr`, `ms`, `ne`, `nl`, `no`, `pl`, `pt`, `pa`, `ro`, `ru`, `sr`, `sv`, `ta`, `te`, `th`, `tl`, `tr`, `uk`, `ur`, `vi`, `zh`.
- [ ] Unsupported values, uppercase values, malformed values, and regional values such as `fr-FR` are rejected before Media Source preparation or transcription starts.
- [ ] Existing commands without `--target-language` behave as before.
- [ ] The glossary defines `Target Language`, `Translation Provider`, and `Subtitle Source` without expanding `Media Source` to include SRT files.
- [ ] App-level tests cover accepted and rejected Target Language behavior.

## Blocked by

None - can start immediately
