# code-visualizer Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-06

## Active Technologies
- Docker, Bash, JSON (devcontainer config); Go 1.26 (project language) + `mcr.microsoft.com/devcontainers/go:2-1.26` base image (002-add-devcontainer)
- Go 1.26+ + Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega (assertions), fogleman/gg (PNG rendering) (003-use-goldie)
- Go 1.26.1 + Kong (CLI), go-git (git metadata), fogleman/gg (PNG rendering), eris (error wrapping), Gomega (test assertions), Goldie v2 (golden-file snapshots) (004-exclude-binary-lines)
- N/A (stateless CLI) (004-exclude-binary-lines)
- Go 1.26.1 + golangci-lint v2.8.0, nilaway (Uber nil-safety analyzer), envsubst (gettext-base) (005-add-linting)

- Go 1.26+ + Kong (CLI parsing), fogleman/gg (PNG rendering), go-git (git metadata) (001-cli-treemap-viz)

## Project Structure

```text
cmd/
  codeviz/
    main.go
internal/
  metric/
  palette/
  render/
  scan/
  treemap/
```

## Commands

- `task build` — Build the codeviz binary
- `task test` — Run all tests
- `task lint` — Run golangci-lint
- `task fmt` — Format with gofumpt
- `task tidy` — Format, mod tidy, lint fix
- `task ci` — Build, test, lint

## Code Style

Go 1.26+: Follow standard conventions, gofumpt formatting, eris error wrapping

## Recent Changes
- 005-add-linting: Added Go 1.26.1 + golangci-lint v2.8.0, nilaway (Uber nil-safety analyzer), envsubst (gettext-base)
- 004-exclude-binary-lines: Added Go 1.26.1 + Kong (CLI), go-git (git metadata), fogleman/gg (PNG rendering), eris (error wrapping), Gomega (test assertions), Goldie v2 (golden-file snapshots)
- 003-use-goldie: Added Go 1.26+ + Goldie v2 (`github.com/sebdah/goldie/v2`), Gomega (assertions), fogleman/gg (PNG rendering)


<!-- MANUAL ADDITIONS START -->

## Agent Workflow Rules

### Running `task lint` / `task ci`

`task lint` runs golangci-lint with `--verbose` on purpose: when issues appear, the surrounding INFO lines are needed to diagnose linter config, exclusion rules, and analyzer stages. Do **not** strip `--verbose`.

Because that verbose output is high-volume and low-value when lint passes, **always run `task lint` and `task ci` via an `Explore` (or equivalent) subagent**, asking it to return only:

- exit status,
- the count and identity of failing linters / failing tests,
- the offending file:line and message for each issue,
- a one-line note if no issues.

Reserve direct `run_in_terminal` invocations of `task lint` / `task ci` for the rare case you need the full transcript.

### Continuous execution

- Never end a turn with only a status / recap message. End a turn only when (a) user input is required, (b) a blocker exists that you cannot resolve, or (c) every todo is completed.
- After a commit, the next action is the next tool call (next task, `task ci`, `git push`, etc.) — not prose.
- Prefer direct edits over subagents for sub-file-sized changes; dispatch subagents for parallelizable work, broad searches, or noisy commands (see above).

<!-- MANUAL ADDITIONS END -->
