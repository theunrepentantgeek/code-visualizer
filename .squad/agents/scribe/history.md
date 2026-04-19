# Scribe — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Session Logger
- **Joined:** 2026-04-14T09:49:33.774Z

## Learnings

<!-- Append learnings below -->

### Session — Bubble Tree Implementation (2026-07-15) — PR #64

- **Issue:** #33 — Add Bubble visualization
- **Branch:** `squad/33-bubble-visualization`
- **Outcome:** Full bubble tree visualization implemented end-to-end; PR #64 created. CI green (build ✅, tests ✅, lint clean except 2 pre-existing issues).
- **Agents involved:**
  - **Ripley:** Researched architecture, produced bubble tree proposal (Wang 2006 front-chain circle-packing + Welzl's enclosing circle). Wrote decision to inbox.
  - **Dallas:** Built layout engine (`internal/bubbletree/node.go` + `layout.go`) and rendering (`internal/render/bubbletree.go` + `svg_bubble.go`). Wrote layout engine decision to inbox.
  - **Kane:** Built CLI command (`cmd/codeviz/bubbletree_cmd.go`) and config (`internal/config/bubbletree.go`).
  - **Lambert:** Wrote 20 tests (16 layout + 4 render smoke tests).
  - **Parker:** Fixed 15+ lint issues (complexity decomposition, variable naming, nolint annotations).
- **Decisions merged:** Ripley's architecture proposal (already in decisions.md), Dallas's layout engine algorithm & constants (merged from inbox this session).
- **Key files:**
  - `internal/bubbletree/node.go`, `internal/bubbletree/layout.go`, `internal/bubbletree/layout_test.go`
  - `internal/render/bubbletree.go`, `internal/render/svg_bubble.go`, `internal/render/bubbletree_test.go`
  - `cmd/codeviz/bubbletree_cmd.go`, `internal/config/bubbletree.go`
