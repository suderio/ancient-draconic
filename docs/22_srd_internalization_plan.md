# Phase 22: SRD Data Internalization (Embedding)

Internalize the baseline SRD data (YAML files) into the `dndsl` executable using Go's `embed` package. This allows the tool to run without an external `data/` directory.

## User Review Required

> [!IMPORTANT]
>
> - **Only YAML files** will be embedded. Image files in the `data/` directory will be ignored to keep the executable size manageable.
> - The `--data-dir` flag will be removed as the baseline data is now internal.
> - The `init` command will be hidden from help output.

## Proposed Changes

### 1. Data Relocation

- Move all `.yaml` files from the root `data/` directory to `internal/data/srd/`.
- Ensure the directory structure is maintained (e.g., `internal/data/srd/monsters/`).
- **Do NOT** copy `.png` or other image files.

### 2. internal/data/loader.go

- **[MODIFY]**: Add `import "embed"` and `//go:embed srd/**/*.yaml` to embed the data.
- **[MODIFY]**: Update the `load()` method to search within the `embed.FS` as the final fallback after checking campaign and world directories.

### 3. cmd/root.go

- **[DELETE]**: Remove the `--data-dir` persistent flag and its binding to Viper.

### 4. cmd/init.go

- **[MODIFY]**: Set `Hidden: true` in the `initCmd` struct.

## Verification Plan

### Automated Tests

- Run `go test ./internal/data/...` to ensure the `Loader` still resolves characters and monsters correctly.

### Manual Verification

- Remove or rename the external `./data` directory.
- Run `dndsl roll by: goblin 1d20` and verify it can still load the goblin's stats from the internalized data.
- Run `dndsl --help` and verify `init` is not listed.
- Verify `dndsl init` still runs if called explicitly (optional for debug).
