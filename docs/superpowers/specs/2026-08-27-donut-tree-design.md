# Donut Tree Design

## Purpose

Add `codeviz donut-tree`, a hierarchical donut visualization of a repository's
folder structure. Each depth of the directory tree occupies a concentric ring;
each folder occupies an annular sector nested within its parent's sector. Files
are not rendered, but their metrics contribute to folder aggregates.

## Command and Configuration

Add a `donut-tree` CLI command and a matching `donutTree` configuration section
with YAML and JSON support. It accepts the common rendering, filtering, title,
footer, and legend options used by other visualizations, plus:

| Option | Required | Default | Meaning |
| --- | --- | --- | --- |
| `--size` | Yes | none | Numeric metric that determines each folder sector's angular share. |
| `--fill` | No | effective size metric | Metric and optional palette for sector fill. |
| `--border` | No | none | Metric and optional palette for sector border. |

The command validates that `size` is numeric and that configured metric
specifications are valid. Configuration values merge with CLI values following
the existing override pattern.

## Metrics and Aggregation

The visualization operates on directories only. The requested size, fill, and
border metrics are loaded for files and aggregated onto directories by the
existing aggregation stage. A directory's sector size is its selected size
metric aggregated over its complete subtree.

An omitted fill metric resolves to the size metric. An omitted border produces
no visible border. Explicit fill and border metrics resolve to directory values
using the established aggregation rules: quantities sum, measures average, and
classifications use the mode. Existing metric descriptors, palette resolution,
ink construction, and legend behavior are reused. The legend reports the
effective directory metrics, including the effective size metric.

## Architecture

Implement `internal/donuttree` as a dedicated visualization package. It owns:

- Visualization-specific state and metric resolution.
- A layout tree describing directory sectors, depth, angular bounds, and label
  placement.
- Annular-sector rendering, curved labels, and donut-specific inks.
- Pipeline stages that build inks and legends, lay out sectors, render, and log
  the result.

The command follows the established visualization lifecycle:

1. Validate paths and merge configuration.
2. Build filters; resolve and register requested metrics.
3. Scan the filesystem, run providers, and compute aggregations.
4. Resolve dimensions and reserved title/footer bounds.
5. Build inks and a legend, lay out sectors, render, then write the canvas.

It reuses `model.Directory`, shared pipeline and stages, canvas backends,
metrics, palettes, inks, legends, and title/footer rendering. The radial-tree
package is unchanged because disc-and-edge layout is a separate concern from
annular-sector layout.

## Layout and Rendering

The scanned root is a centered anchor labeled with the target directory name.
Its direct child folders fill the innermost ring. Each deeper directory level
fills the next uniform-width ring. Ring width is derived from the available
square drawing area after title and footer reservations.

A parent sector's complete angular interval is allocated among its direct child
folders. Positive size values receive the remaining sweep proportionally to
their aggregated sizes. Empty or zero-size children remain visible through an
equal minimum angular allocation. This prevents invalid geometry and preserves
the full folder hierarchy.

Each directory sector is an annular path. Its fill comes from the effective fill
metric. Its stroke is rendered only when the border metric is configured. The
canvas rendering must work consistently for raster and SVG output.

## Labels

Every visible sector attempts to render a centered label along the arc at the
ring's midpoint. The label contains:

- The folder name.
- The effective size metric value.
- The effective fill metric value when fill was explicitly configured.
- The effective border metric value when border was explicitly configured.

The renderer scales a label down to fit the available arc length, but never
below 6 px. If the label cannot fit at 6 px, it is omitted while its sector
remains rendered. Labels orient so they remain readable on either side of the
circle.

## Edge Cases and Errors

Metric resolution or provider failures stop the command through the existing
contextual pipeline error path. An empty repository renders only the root
anchor. Trees with zero-valued directory metrics render valid minimum sectors.
No configured border means no stroke rather than a fallback border color.

## Testing and Documentation

Add tests for:

- CLI/config override behavior and size/fill/border validation.
- Requested metrics and effective defaults, including directory aggregations.
- Sector hierarchy, parent bounds, proportional allocation, and zero/empty
  directory cases.
- Curved-label content, orientation, fitting, the 6 px minimum, and omission.
- Fill and no-border behavior.
- PNG and SVG golden snapshots.

Add command help, visualization documentation, sample configuration, sample
generation wiring, and generated donut-tree images. The documentation states
that files are not drawn and explains defaults for fill and border.
