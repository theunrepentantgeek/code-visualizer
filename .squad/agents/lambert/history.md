# Lambert — History

## Core Context

- **Project:** A Go CLI tool that scans file trees and renders treemap visualizations as PNG images with configurable metrics and colour palettes.
- **Role:** Tester
- **Joined:** 2026-04-14T09:49:33.773Z

## Learnings

<!-- Append learnings below -->

### 2026-04-14 — radialtree layout tests

Wrote `internal/radialtree/layout_test.go` (white-box, `package radialtree`) with 12 test cases covering:
- Root always placed at centre (0,0)
- Children positioned in a ring at positive radius
- Single file child has positive DiscRadius
- Four equal-weight files produce four distinct angles (no duplicates)
- Nested depth: file radius > subdir radius > root radius (0)
- Larger metric value produces larger DiscRadius
- LabelAll: both root and file ShowLabel == true
- LabelFoldersOnly: root ShowLabel == true, file ShowLabel == false
- LabelNone: both ShowLabel == false
- Empty directory returns without panic
- Root.Label reflects directory Name
- Larger canvasSize produces larger child radii

Followed the exact style from `internal/treemap/layout_test.go`: `t.Parallel()`, `NewGomegaWithT(t)`, nilaway-safe nil guards, no testify.
