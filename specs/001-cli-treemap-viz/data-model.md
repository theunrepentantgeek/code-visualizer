# Data Model: CLI Treemap Visualization

**Branch**: `001-cli-treemap-viz` | **Date**: 2026-04-04

## Entities

### FileNode

Represents a single file discovered during directory scanning.

| Field       | Type             | Description                                                   |
| ----------- | ---------------- | ------------------------------------------------------------- |
| Path        | `string`         | Absolute or relative path from scan root                      |
| Name        | `string`         | Base filename (e.g., `main.go`)                               |
| Extension   | `string`         | File extension without dot (e.g., `go`), empty string if none |
| Size        | `int64`          | File size in bytes (from `os.FileInfo`)                       |
| LineCount   | `int`            | Number of lines (0 for binary files or if unavailable)        |
| FileType    | `string`         | Extension-based classification; `"no-extension"` when absent  |
| Age         | `*time.Duration` | Duration since first git commit; nil if not a git repo        |
| Freshness   | `*time.Duration` | Duration since most recent git commit; nil if not a git repo  |
| AuthorCount | `*int`           | Number of distinct git committers; nil if not a git repo      |
| IsBinary    | `bool`           | True if git classifies this file as binary                    |

**Notes**:
- Git-derived fields (`Age`, `Freshness`, `AuthorCount`) are pointer types to distinguish "not computed" (nil) from zero-value.
- `IsBinary` determined by go-git's content inspection or `.gitattributes`; defaults to `false` in non-git directories.

### DirectoryNode

Represents a directory in the scanned tree. Recursive structure.

| Field | Type              | Description                              |
| ----- | ----------------- | ---------------------------------------- |
| Path  | `string`          | Absolute or relative path from scan root |
| Name  | `string`          | Directory name (leaf component)          |
| Files | `[]FileNode`      | Direct child files                       |
| Dirs  | `[]DirectoryNode` | Direct child directories                 |

**Relationships**:
- A `DirectoryNode` contains zero or more `FileNode` children and zero or more `DirectoryNode` children.
- The root `DirectoryNode` represents the user-specified scan target.

### MetricName

Custom string type constraining valid metric identifiers.

| Value            | Type        | Description                       | Git Required |
| ---------------- | ----------- | --------------------------------- | ------------ |
| `file-size`      | Numeric     | Size in bytes                     | No           |
| `file-lines`     | Numeric     | Line count                        | No           |
| `file-type`      | Categorical | Extension-based classification    | No           |
| `file-age`       | Numeric     | Duration since first commit       | Yes          |
| `file-freshness` | Numeric     | Duration since most recent commit | Yes          |
| `author-count`   | Numeric     | Distinct committer count          | Yes          |

**Validation Rules**:
- Only numeric metrics may be used as the size metric (FR-004).
- All metrics may be used for fill or border colour.
- Git-required metrics produce an error if the target is not a git repository (FR-011).

### PaletteName

Custom string type constraining valid palette identifiers.

| Value            | Steps | Ordering    | Description                                   |
| ---------------- | ----- | ----------- | --------------------------------------------- |
| `categorization` | 12    | Unordered   | Visually distinct colours for discrete groups |
| `temperature`    | 11    | Bipolar     | Dark blue → white → bright red                |
| `good-bad`       | 13    | Progressive | Red → orange → yellow → green                 |
| `neutral`        | 9     | Linear      | Black → white (monochromatic)                 |

### ColourPalette

Runtime representation of a palette.

| Field   | Type           | Description                                                   |
| ------- | -------------- | ------------------------------------------------------------- |
| Name    | `PaletteName`  | Palette identifier                                            |
| Colours | `[]color.RGBA` | Ordered sequence of discrete colour steps                     |
| Ordered | `bool`         | True for sequential/diverging palettes; false for categorical |

**Invariant**: `len(Colours)` equals the palette's defined step count.

### MetricDefaultPalette

Maps each metric to its default palette (FR-010a).

| Metric           | Default Palette  |
| ---------------- | ---------------- |
| `file-size`      | `neutral`        |
| `file-lines`     | `neutral`        |
| `file-age`       | `temperature`    |
| `file-freshness` | `temperature`    |
| `author-count`   | `good-bad`       |
| `file-type`      | `categorization` |

### BucketBoundaries

Result of quantile-based bucketing for a set of metric values.

| Field      | Type        | Description                                                |
| ---------- | ----------- | ---------------------------------------------------------- |
| Boundaries | `[]float64` | N-1 breakpoints for N palette steps, rounded to 2 sig figs |
| Min        | `float64`   | Observed minimum value                                     |
| Max        | `float64`   | Observed maximum value                                     |
| StepCount  | `int`       | Number of palette steps (equals palette colour count)      |

**Behaviour**:
- Boundaries are computed via quantile distribution.
- Boundary values rounded to 2 significant figures for human comprehensibility.
- Duplicate boundaries after rounding are deduplicated; affected palette steps are merged.

### TreemapRectangle

A positioned visual element in the rendered treemap.

| Field        | Type                 | Description                                                 |
| ------------ | -------------------- | ----------------------------------------------------------- |
| X            | `float64`            | Left edge position in pixels                                |
| Y            | `float64`            | Top edge position in pixels                                 |
| W            | `float64`            | Width in pixels                                             |
| H            | `float64`            | Height in pixels                                            |
| FillColour   | `color.RGBA`         | Fill colour from palette mapping                            |
| BorderColour | `*color.RGBA`        | Border colour from palette mapping; nil if no border metric |
| Label        | `string`             | File or directory name                                      |
| ShowLabel    | `bool`               | Whether the rectangle is large enough to display the label  |
| IsDirectory  | `bool`               | True for directory header bars                              |
| Children     | `[]TreemapRectangle` | Nested rectangles (for directory groups)                    |

### RenderConfiguration

The user's complete set of rendering parameters.

| Field         | Type           | Description                                            |
| ------------- | -------------- | ------------------------------------------------------ |
| TargetPath    | `string`       | Directory to scan                                      |
| OutputPath    | `string`       | Path for the output PNG file                           |
| SizeMetric    | `MetricName`   | Metric determining rectangle area (must be numeric)    |
| FillMetric    | `MetricName`   | Metric for fill colour (defaults to SizeMetric)        |
| FillPalette   | `PaletteName`  | Palette for fill colour (defaults to metric's default) |
| BorderMetric  | `*MetricName`  | Metric for border colour; nil means no borders         |
| BorderPalette | `*PaletteName` | Palette for border colour; nil when no border metric   |
| Verbose       | `bool`         | Enable debug-level logging                             |
| OutputFormat  | `string`       | `"text"` or `"json"` for diagnostic output             |
| Width         | `int`          | Image width in pixels (default 1920)                   |
| Height        | `int`          | Image height in pixels (default 1080)                  |

## State Transitions

No state machine — this is a stateless CLI tool. The data flow is:

```
RenderConfiguration
    → scan(TargetPath) → DirectoryNode tree with FileNode leaves
    → computeMetrics(tree) → FileNodes populated with metric values
    → bucket(values, palette) → BucketBoundaries
    → layout(tree, sizeMetric) → TreemapRectangle tree
    → mapColours(rectangles, fill/border metrics, palettes, buckets) → coloured TreemapRectangles
    → render(rectangles) → PNG file at OutputPath
```

## Entity Relationship Diagram

```
RenderConfiguration ──uses──▶ MetricName
                    ──uses──▶ PaletteName

DirectoryNode ──contains──▶ FileNode (0..*)
              ──contains──▶ DirectoryNode (0..*)

FileNode ──measured-by──▶ MetricName (1..6 values)

MetricName ──default-palette──▶ PaletteName

PaletteName ──defines──▶ ColourPalette

ColourPalette + MetricValues ──bucketed-into──▶ BucketBoundaries

TreemapRectangle ──derived-from──▶ FileNode | DirectoryNode
                 ──coloured-by──▶ ColourPalette + BucketBoundaries
                 ──contains──▶ TreemapRectangle (0..*)
```
