# Research: Configure Linting (005-add-linting)

**Date**: 2026-04-06

## R-001: Custom golangci-lint Build with nilaway Plugin

**Decision**: Use `golangci-lint custom` to build a custom binary (`golangci-lint-custom`) that embeds the nilaway plugin as a module-based linter plugin.

**Rationale**: nilaway is a static analysis tool from Uber that detects potential nil panics in Go code. It is not bundled with standard golangci-lint and must be included via the plugin/custom build mechanism. The reference project (go-vcr-tidy) uses this same approach successfully.

**Alternatives considered**:
- Running nilaway as a standalone tool separately → rejected because it would require a separate invocation and wouldn't integrate with golangci-lint's filtering/exclusion system.
- Using `nogo` with Bazel → rejected; project uses `go build` / Task, not Bazel.

## R-002: Custom Build Template Configuration

**Decision**: Create `.devcontainer/.custom-gcl.template.yml` using `envsubst` for `$TOOL_DEST` substitution, matching the go-vcr-tidy pattern.

**Rationale**: The template pattern with `envsubst` allows the same configuration to work in both devcontainer mode (`/usr/local/bin`) and local mode (`hack/tools`). The go-vcr-tidy reference uses this exact pattern.

**Template format** (adapted from go-vcr-tidy):
```yaml
version: v2.8.0
name: golangci-lint-custom
destination: $TOOL_DEST
plugins:
  - module: "go.uber.org/nilaway"
    import: "go.uber.org/nilaway/cmd/gclplugin"
    version: latest
```

**Key difference**: Version should match the golangci-lint version installed by the script (v2.8.0 in code-visualizer vs v2.6.2 in go-vcr-tidy).

## R-003: Linter Configuration Import from go-vcr-tidy

**Decision**: Import the go-vcr-tidy `.golangci.yml` configuration, adapting project-specific settings (module paths in nilaway `include-pkgs`, revive dot-imports packages, etc.).

**Rationale**: The go-vcr-tidy configuration enables a comprehensive set of ~60+ linters covering code quality, security, style, and nil-safety. Reusing a proven configuration avoids trial-and-error.

**Adaptations needed**:
- `nilaway.settings.include-pkgs`: Change from `github.com/theunrepentantgeek/crddoc` to the code-visualizer module path
- `revive.rules.dot-imports.arguments`: Keep gomega, remove ginkgo (not used in this project)
- `wrapcheck.extra-ignore-sigs`: Keep eris wrappers (eris is used in this project)
- Exclusion paths: Keep `third_party$`, `builtin$`, `examples$`
- `issues.max-issues-per-linter`: Keep at 10 initially, can adjust after initial run

## R-004: Taskfile Modifications

**Decision**: Modify the `lint` and `tidy` tasks in `Taskfile.yml` to invoke `golangci-lint-custom` instead of `golangci-lint`.

**Rationale**: Follows the same pattern as go-vcr-tidy. The custom binary is a drop-in replacement that supports all the same flags plus the nilaway plugin.

**Changes required**:
- `lint` task: `golangci-lint run ./...` → `golangci-lint-custom run --verbose`
- `tidy` task: `golangci-lint run --fix ./...` → `golangci-lint-custom run --fix --verbose`

## R-005: install-dependencies.sh Modifications

**Decision**: Add a custom linter build step to `install-dependencies.sh` after the standard golangci-lint installation.

**Rationale**: The custom build depends on `golangci-lint` being installed first (it uses it to build the custom binary). The build step uses `envsubst` to expand `$TOOL_DEST` in the template, runs `golangci-lint custom`, then cleans up the temporary file.

**Key considerations**:
- `envsubst` must be available — it's part of `gettext-base` which is typically available on Debian-based images. The devcontainer base image (`mcr.microsoft.com/devcontainers/go:2-1.26`) should include it or it should be installed in the Dockerfile.
- The `should-install` check uses `$TOOL_DEST/golangci-lint-custom` as the sentinel file.
- `SCRIPT_DIR` must be set to locate the template (matches go-vcr-tidy pattern).

## R-006: Fixing Existing Lint Issues

**Decision**: Fix issues at the root cause rather than suppress warnings, per the issue and spec requirements.

**Rationale**: Starting with a clean baseline is essential. The new linters will likely find issues in existing code related to nil checks, error handling, function length, complexity, etc.

**Approach**:
- Run the expanded linter on the codebase
- Categorize findings
- Fix issues that represent genuine code quality improvements
- Suppress only where the linter is clearly wrong (e.g., false positives) with `//nolint` directives including required explanation
