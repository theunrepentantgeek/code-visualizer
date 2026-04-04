# Research: CLI Treemap Visualization

**Branch**: `001-cli-treemap-viz` | **Date**: 2026-04-04

## 1. Squarified Treemap Layout Algorithm

**Decision**: Use `github.com/nikolaydubina/treemap` (v1.2.5) — specifically its `layout` sub-package which exposes `layout.Squarify(box, areas) []Box`.

**Rationale**: Most mature Go implementation of the Bruls/Huizing/van Wijk 2000 squarified treemap algorithm. The `layout` package is cleanly separated from the rendering package — use the layout algorithm without coupling to the library's SVG renderer. API is minimal: pass a bounding `Box{X, Y, W, H}` and a slice of area values, receive positioned boxes. Handles edge cases (zero areas, sanity checks). MIT licensed, 171+ stars, tagged stable releases. The parent library's `render` package and `UIBox` tree structure serve as reference for nested directory groups with margins/padding.

**Alternatives Considered**:
- `jeffwilliams/squarify` — Also implements Bruls/Huizing/van Wijk. Published 2015, no tagged versions, lower adoption.
- `MazenAlkhatib/treemap` (v1.1.0) — Very new (Apr 2025), less battle-tested.
- Implement from paper — Well-documented (~2 pages), but nikolaydubina's implementation already handles edge cases and has tests. No reason to reimplement.

## 2. PNG Rendering

**Decision**: Use `github.com/fogleman/gg` (v1.3.0) for 2D rasterisation to PNG.

**Rationale**: Pure Go 2D graphics library built on `image/draw` and `golang.org/x/image`. Provides exactly the primitives needed: `DrawRectangle`, `Fill`, `Stroke`, `SetLineWidth`, `SetHexColor`/`SetColor`, `DrawString`/`DrawStringAnchored`/`DrawStringWrapped`, `MeasureString`, `WordWrap`, `LoadFontFace`, and `SavePNG`/`EncodePNG`. No CGO dependency — fully cross-platform. 1,942+ importers. `MeasureString()` enables the label-fitting logic for FR-008a (omit labels on rectangles too small).

**Alternatives Considered**:
- `image/draw` + `image/png` (stdlib) — Too low-level. Would need to implement rectangle drawing, anti-aliasing, text rendering, and font loading manually.
- `github.com/llgcode/draw2d` — More complex API, less community adoption.
- Cairo/Pango bindings — Requires CGO and system library installation. Violates cross-platform goal.

## 3. Git Metadata Extraction

**Decision**: Use `github.com/go-git/go-git/v5` for pure Go git access.

**Rationale**: Definitive pure-Go git implementation (4,756+ importers, Apache 2.0). For the three git metrics:
- **File age** (first commit): `Repository.Log(&git.LogOptions{FileName: &path})`, iterate to the last (earliest) commit.
- **File freshness** (most recent commit): Same log call, take the first commit returned.
- **Author count**: Iterate file-filtered log, collect unique `commit.Author.Email` values.
- **Binary detection**: Can enumerate tree entries and check file attributes via `.gitattributes` or content inspection.

Removes the external `git` binary dependency entirely, making the tool more portable. Performance should be within the 15-second budget for repos up to 10,000 files.

**Alternatives Considered**:
- Shell out to `git log` via `os/exec` — Output parsing fragility, Windows PATH issues, external dependency.
- `libgit2` bindings (`git2go`) — Requires CGO and system libgit2 installation. Not cross-platform friendly.

## 4. Quantile-Based Bucketing

**Decision**: Implement from scratch using Go's `sort` package and `math` for significant-figure rounding. No external library needed.

**Rationale**: The algorithm is straightforward (~30-50 lines):
1. Sort all metric values.
2. For N palette steps, compute quantile breakpoints at positions `i * len(values) / N` for `i = 1..N-1`.
3. Round each breakpoint to 2 significant figures using `math.Round(v / pow) * pow` where `pow = 10^(floor(log10(abs(v))) - 1)`.
4. Deduplicate boundaries that collapse after rounding.
5. Assign files to buckets via `sort.SearchFloat64s`.

**Alternatives Considered**:
- `gonum/stat` — Has `stat.Quantile()` but pulling entire gonum dependency for one trivial call is unnecessary. Sig-fig rounding must be custom regardless.
- `montanaflynn/stats` — Lightweight but still overkill for sorted-index quantiles.

## 5. Kong CLI Framework

**Decision**: Use `github.com/alecthomas/kong` with struct tags. Single command (no subcommands).

**Rationale**: Key features for this project:
- **`enum` tag**: Constrain metric names to valid values (e.g., `enum:"file-size,file-lines,file-age,file-freshness,author-count,file-type"`).
- **`required`/`optional`**: Mandatory size metric + target path; optional fill/border metric and palette.
- **`default`**: Set default palette per metric configuration.
- **`Validatable` interface**: Enforce rules like "categorical metrics cannot be used as size metric" (FR-004) and "border palette requires border metric".
- **`Run(...) error` method**: Clean execution flow on the CLI struct.

**Alternatives Considered**:
- `cobra` + `pflag` — More boilerplate. Kong's struct-tag approach is more concise; `enum` tag directly solves metric validation.
- `urfave/cli` — Functional-style API, less type-safe.
- `flag` (stdlib) — Too primitive for this use case.

## 6. WCAG 2.1 AA Colour Palettes

**Decision**: Design all four palettes to meet WCAG 2.1 SC 1.4.11 (Non-text Contrast, 3:1 minimum). Use structural dark borders between all rectangles as the primary mechanism for adjacency distinction.

**Rationale**: Each treemap rectangle is a graphical object under SC 1.4.11. The practical approach:
- **All rectangles get a thin dark border** (e.g., `#333333`) regardless of whether a border-colour metric is active. This ensures adjacent rectangles of similar fill are distinguishable. The border-colour metric, when specified, replaces this default structural border.
- **Palettes** designed with adequate luminance stepping between adjacent colours. Use ColorBrewer (Cynthia Brewer's research) as the reference — designed for data visualisation with perceptual uniformity.
- **Text labels**: Use dark text on light fills and light text on dark fills, switching based on relative luminance threshold (~0.5). This meets SC 1.4.3 at 4.5:1 for text.

Note: SC 1.4.11's "essential" exception applies to data visualisations where colour conveys information. The palettes are technically exempt, but meeting the minimums is best practice.

**Alternatives Considered**:
- Relying on fill colour alone to distinguish adjacent rectangles — Fails when adjacent files share a quantile bucket. Borders are necessary.
- `lucasb-eyer/go-colorful` for palette interpolation — Spec requires discrete steps (no interpolation), so hardcoded hex values are simpler. Could still be useful for computing relative luminance in palette validation tests.

## Dependency Summary

| Dependency                         | Version | Purpose                      | License    |
| ---------------------------------- | ------- | ---------------------------- | ---------- |
| `github.com/alecthomas/kong`       | v1.15.0 | CLI argument parsing         | MIT        |
| `github.com/nikolaydubina/treemap` | v1.2.5  | Squarified treemap layout    | MIT        |
| `github.com/fogleman/gg`           | v1.3.0  | 2D PNG rendering             | MIT        |
| `github.com/go-git/go-git/v5`      | v5.17.2 | Git metadata extraction      | Apache 2.0 |
| `github.com/onsi/gomega`           | latest  | Test assertions              | MIT        |
| `github.com/sebdah/goldie/v2`      | latest  | Golden-file snapshot testing | MIT        |
