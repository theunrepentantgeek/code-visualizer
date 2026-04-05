# Research: Use Goldie for Golden File Testing

**Date**: 2026-04-05  
**Feature**: 003-use-goldie

## R1: Goldie v2 Binary File Support

**Decision**: Goldie v2 performs byte-level comparison by default using `bytes.Equal`, which works correctly for binary files including PNG images. No special configuration is needed for binary comparison.

**Rationale**: Goldie's `Assert` method compares `[]byte` slices directly. The default `EqualFn` uses `bytes.Equal` for raw byte comparison. This is identical in behavior to the current pixel-by-pixel comparison (since identical bytes produce identical pixels), but simpler and faster (no need to decode PNG and iterate pixels).

**Alternatives considered**:
- Custom `WithEqualFn` for pixel-level comparison: Rejected — byte-level comparison is stricter and sufficient since the same rendering code produces deterministic output. If byte-level comparison passes, pixel-level comparison necessarily passes.

## R2: Golden File Directory Layout

**Decision**: Use `WithFixtureDir("testdata")` to keep golden files in the existing `internal/render/testdata/` directory. Use `WithNameSuffix(".png")` to match existing file naming (Goldie defaults to `.golden` suffix).

**Rationale**: Preserves existing directory structure and avoids a breaking migration of golden file locations. The 6 existing PNG files remain in place.

**Alternatives considered**:
- Goldie default `testdata/fixtures/` with `.golden` suffix: Rejected — would require renaming all 6 existing golden files and updating `.gitignore` or similar. No benefit over current layout.
- Separate fixture directory per test: Rejected — `useTestNameForDir` would scatter files across subdirectories. Current flat layout in `testdata/` is simpler for 6 files.

## R3: Update Mechanism

**Decision**: Use Goldie's built-in `-update` flag and `GOLDIE_UPDATE` environment variable. Update the Taskfile `update-golden-files` task to use `GOLDIE_UPDATE=1` instead of the custom `UPDATE_GOLDEN=1`.

**Rationale**: Goldie registers `-update` as a standard `flag.Bool`. The `GOLDIE_UPDATE` env var is also supported natively via `truthy(os.Getenv("GOLDIE_UPDATE"))`. This replaces the custom `UPDATE_GOLDEN` env var with a standard mechanism.

**Alternatives considered**:
- Keep `UPDATE_GOLDEN` alongside Goldie: Rejected — maintaining two update mechanisms creates confusion. The custom env var must be retired.
- Use only `-update` flag (no env var): Rejected — `GOLDIE_UPDATE` env var is useful in CI/Taskfile contexts where passing test flags is less ergonomic.

## R4: Goldie v2 API Usage Pattern

**Decision**: Use `goldie.New(t, options...)` to create a tester instance, then call `g.Assert(t, name, actualData)` where `actualData` is the raw PNG bytes read from the rendered output file.

**Rationale**: This is Goldie's standard API pattern. The test renders to a temp file, reads the bytes, and passes them to `Assert`. Goldie handles reading the golden file, comparing, and updating (when `-update` or `GOLDIE_UPDATE` is set).

**Alternatives considered**:
- `g.AssertJson` / `g.AssertWithTemplate`: Not applicable — these are for structured text data, not binary files.

## R5: Migration Impact on Test Code

**Decision**: The `goldenPaletteTest` helper function (~40 lines) will be replaced with ~5 lines using Goldie's API. The 4 `TestGoldenFile_*` test functions remain unchanged in structure (they still call a helper). Only the helper's implementation changes.

**Rationale**: Minimal diff, maximum reduction in custom code. The test names, test structure, and coverage remain identical.

**Alternatives considered**:
- Inline Goldie calls in each test function: Rejected — the helper pattern keeps tests DRY and consistent.
