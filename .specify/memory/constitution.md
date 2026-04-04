<!--
  Sync Impact Report
  ==================
  Version change: 0.0.0 → 1.0.0 (MAJOR: initial ratification)
  Modified principles: N/A (initial)
  Added sections:
    - Core Principles (8 principles)
    - Technology Constraints
    - Development Workflow
    - Performance Standards
    - Governance
  Removed sections: None
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ compatible (Constitution Check section present)
    - .specify/templates/spec-template.md ✅ compatible (requirements/stories structure aligned)
    - .specify/templates/tasks-template.md ✅ compatible (phase structure supports TDD workflow)
  Follow-up TODOs: None
-->

# Code Visualizer Constitution

## Core Principles

### I. Test-First (NON-NEGOTIABLE)

- TDD is mandatory for all production code: tests MUST be written
  before implementation.
- Red-Green-Refactor cycle MUST be strictly followed.
- Tests use Gomega for assertions and Goldie for golden-file
  snapshot testing.
- No feature code merges without corresponding passing tests.
- Integration tests MUST cover CLI output, Fyne rendering
  contracts, and cross-component data flow.

### II. API-First Design

- All public interfaces MUST be designed as explicit Go
  interfaces or typed function signatures before implementation.
- CLI commands MUST define their contract (flags, arguments,
  stdin/stdout formats) before any handler code is written.
- Internal package boundaries MUST expose clean, minimal APIs;
  unexported internals are the default.
- Breaking API changes MUST be documented and versioned.

### III. Type Safety

- Go's type system MUST be leveraged fully; use of `any` or
  `interface{}` requires written justification in a code comment.
- Custom types MUST be preferred over primitive aliases when they
  convey domain semantics (e.g., `type NodeID string`).
- All exported functions MUST return typed errors or sentinel
  error values; bare `error` strings are discouraged.
- Struct fields MUST use appropriate types — no stringly-typed
  configuration.

### IV. Simplicity / YAGNI

- Start with the simplest implementation that satisfies the
  current requirement. Do not build for speculative futures.
- No abstraction layers, indirection, or generics unless there
  are at least two concrete consumers.
- Premature optimization is prohibited; optimize only after
  profiling identifies a bottleneck.
- Every dependency added to `go.mod` MUST be justified; prefer
  the standard library when feasible.

### V. Performance

- Visualization rendering MUST complete within published time
  budgets (defined per visualization type in specs).
- Memory consumption MUST be profiled for repositories exceeding
  10,000 source files.
- CLI operations MUST remain responsive: startup under 500ms,
  progress feedback for operations exceeding 2 seconds.
- Fyne UI MUST maintain 60 fps during pan/zoom interactions on
  rendered graphs.

### VI. Accessibility

- The Fyne interactive UI MUST support keyboard navigation for
  all primary actions.
- Color choices MUST meet WCAG 2.1 AA contrast ratios (4.5:1
  for normal text, 3:1 for large text/graphics).
- Visualizations MUST NOT rely solely on color to distinguish
  elements; shape, pattern, or label differentiation is required.
- CLI output MUST support both human-readable and
  machine-parseable (JSON) formats.

### VII. Observability

- Structured logging via Go's `slog` package is required for all
  CLI and background operations.
- Log levels MUST be meaningful: `DEBUG` for development tracing,
  `INFO` for operational milestones, `ERROR` for actionable
  failures.
- CLI commands MUST support a `--verbose` / `-v` flag to expose
  debug-level output.
- Errors MUST include sufficient context (operation, input, cause)
  to diagnose without a debugger.

### VIII. Documentation

- All exported packages MUST have a package-level doc comment
  explaining purpose and usage.
- All public API changes MUST include updated documentation
  before merge.
- User-facing features MUST include usage examples in docs/.
- CLI help text MUST be comprehensive: every command, flag, and
  argument documented with examples.
- Architecture decisions MUST be recorded when they deviate from
  the obvious approach.

## Technology Constraints

- **Language**: Go (latest stable release).
- **UI Framework**: Fyne for the interactive visualization UX.
- **CLI Framework**: Kong for command-line argument parsing.
- **Testing**: Gomega (assertions) + Goldie (golden-file
  snapshots). Standard `go test` runner.
- **Build Orchestration**: Taskfile (`task`) for build, test,
  lint, and release workflows.
- **Formatting/Linting**: `gofmt` and `golangci-lint` MUST pass
  with zero warnings before merge.
- **Module Structure**: Single Go module at repository root.
  Internal packages under `internal/`; shared code under `pkg/`
  only when consumed by both CLI and Fyne entrypoints.
- **Version Control**: Git with conventional commit messages.

## Development Workflow

- All changes MUST be submitted via pull request against `main`.
- PRs MUST pass CI checks: `task lint`, `task test`, `task build`
  before review.
- Every PR MUST include a description linking to the relevant
  spec or task ID.
- Code review is required from at least one other contributor
  (or self-review with documented rationale for solo maintainer
  periods).
- Feature branches MUST follow the naming convention
  `###-feature-name` (sequential numbering).
- Commits MUST follow conventional commit format:
  `type(scope): description`.

## Performance Standards

- **CLI cold start**: MUST complete in under 500ms for help/
  version commands.
- **Small repo visualization** (<1,000 files): MUST render in
  under 5 seconds.
- **Medium repo visualization** (1,000–10,000 files): MUST render
  in under 30 seconds.
- **Large repo visualization** (>10,000 files): MUST provide
  incremental/streaming rendering with progress feedback.
- **Fyne UI frame rate**: MUST sustain 60 fps during interactive
  exploration (pan, zoom, select).
- **Memory**: CLI process MUST NOT exceed 512 MB RSS for repos
  under 10,000 files.
- Performance regressions MUST be caught by benchmark tests run
  in CI.

## Governance

- This constitution supersedes all other project practices and
  conventions. In case of conflict, the constitution prevails.
- Amendments require:
  1. A written proposal describing the change and rationale.
  2. Update to this document with incremented version number.
  3. Propagation of changes to all dependent templates and docs.
- Versioning follows semantic versioning:
  - **MAJOR**: Principle removal or incompatible redefinition.
  - **MINOR**: New principle or materially expanded guidance.
  - **PATCH**: Clarifications, wording, or non-semantic fixes.
- All PRs and code reviews MUST verify compliance with these
  principles.
- Complexity beyond what the constitution permits MUST be
  justified in writing within the relevant spec or plan.

**Version**: 1.1.0 | **Ratified**: 2026-04-04 | **Last Amended**: 2026-04-04
