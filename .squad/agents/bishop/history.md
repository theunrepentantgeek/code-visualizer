# Bishop — History

## Core Context

- **Project:** A Go CLI tool (`codeviz`) that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Artificer — abstractions, interfaces, types, design patterns, code smells, structural integrity
- **Joined:** 2026-04-15
- **Requested by:** Bevan Arps

## Project Knowledge

- Language: Go 1.26+
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`
- Key packages: `cmd/codeviz/` (entry), `internal/metric/`, `internal/palette/`, `internal/render/`, `internal/scan/`, `internal/treemap/`, `internal/provider/`
- Build: `task build` → `bin/codeviz`
- Test: `task test` (`go test ./... -count=1`)
- Lint: `task lint` (golangci-lint v2 with nilaway, wsl_v5, revive, wrapcheck, gci)
- Format: `task fmt` (gofumpt); imports ordered by gci
- Error handling: eris wrapping throughout
- Test assertions: Gomega (not testify); golden files via Goldie v2
- Go is already a pattern-friendly language: interfaces are implicit, types are first-class, package boundaries enforce encapsulation
- Key structural seams to watch: metric/palette/render boundaries, how scan feeds treemap, how types flow from scan → treemap → render

## Learnings

<!-- Append learnings below -->

### PR #39 Review (2026-04-15)

**Reviewed:** Provider interface extension with `Scope()` and `Description()`, plus 9 new folder metrics.

**Structural decisions affirmed:**
- `Scope` as typed string (not iota) is correct for Open/Closed extensibility
- Interface remains cohesive at 7 methods — no segregation needed
- `Description` as `string` is appropriate (not primitive obsession)
- Dependencies as `[]metric.Name` with runtime validation is pragmatic
- New `internal/provider/folder/` package is well-bounded

**Pattern identified for future:**
- Folder aggregation pattern (sum/max/min/mean over tree) appears in 9 providers
- Currently WET — could extract `aggregateSum()`, `aggregateMax()`, `aggregateMean()` helpers
- Not blocking, but watch for more folder metrics tipping toward DRY

**Review output:** Orchestration log at `.squad/orchestration-log/2026-04-15T04:50:46Z-bishop.md`

### Full Structural Audit (2026-07-07)

**Scope:** Complete review of `cmd/codeviz/` and all `internal/` packages.

**Issues filed (11 total):**
- #152 — Extract shared command workflow (highest impact, ~1,500 lines duplicated across 4 commands)
- #153 — Extract shared base config struct (data clump across 4 viz config types)
- #154 — Extract shared LabelMode type (triplicated across radialtree/bubbletree/spiral)
- #155 — Replace 7 git provider wrappers with declarative registration (WET boilerplate)
- #156 — Extract MetricBag from File/Directory (duplicated metric storage, ~130 lines)
- #157 — Deduplicate luminance calculation (render vs palette)
- #158 — Unify raster/SVG rendering paths (duplicated across 8 renderer files + legend)
- #159 — Move git history loading out of spiral layout package (boundary violation)
- #160 — LegendEntry union type permits invalid states
- #161 — Extract shared progress ticker pattern (triplicated goroutine lifecycle)
- #162 — Consider splitting Provider interface (ISP)

**Key structural observations:**
- The cmd/ layer is the primary pain point: ~2,576 lines across 4 commands with ~60% duplication
- The render package is the second hotspot: ~2,300 lines with raster/SVG duplication throughout
- The model layer has clean semantics but duplicated implementation (metric storage)
- Provider/git has good centralised helpers (`loadGitMetric`) but 7 near-identical wrapper files
- Config package acknowledges its own duplication via `//nolint:dupl` comments
- The spiral package is the only layout package with a data-access dependency (boundary smell)
- The codebase is well-structured overall — issues are about duplication and missing abstractions, not about fundamental design problems

**Priority ranking:**
1. #152 (command workflow) — highest leverage, eliminates most duplication
2. #158 (render unification) — second highest, prevents duplication scaling with new viz types
3. #155 (git providers) — quick win, enables cleaner provider model
4. #153 (config base) — quick win, removes acknowledged duplication
5. #156 (MetricBag) — clean extraction, low risk
6. #154 (LabelMode) — trivial but prevents drift
7. #157 (luminance) — trivial fix
8. #159 (spiral boundary) — important for testability
9. #160 (LegendEntry) — moderate, prevents nil bugs
10. #161 (progress ticker) — small cleanup
11. #162 (provider ISP) — consider alongside #155
