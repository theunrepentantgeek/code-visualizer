# Alluvial Sample Design

## Purpose

Demonstrate the Alluvial Diagram with the CodeViz repository's own evolution:
each release column represents the first commit that introduced a visualization
type. The sample makes directory-level growth across those milestones visible
and remains reproducible from committed repository history.

## Release Tags

Create annotated `v0.x` tags in chronological order. `v0.1` marks the first
visualization introduction, and every later tag marks the first commit that
introduces one additional visualization type. The series includes the Alluvial
Diagram as its latest milestone.

Tags are additive repository metadata only: existing tags and commits are not
rewritten. The Taskfile owns the ordered tag list so regeneration fails clearly
when a required milestone is unavailable.

## Sample Artifacts

Add `samples/alluvial/` containing an Alluvial-specific configuration and
committed PNG and SVG output. The configuration:

- uses the ordered `v0.x` release series;
- selects a stable directory metric;
- uses explicit expansion only where needed for a readable, bounded view; and
- uses the existing stable footer convention for reproducible output.

Add `samples-alluvial` to the Taskfile and make `samples` invoke it with the
existing visualization sample tasks. Extend `docs:gallery` to copy the PNG to
the gallery directory.

## Gallery Documentation

Add an Alluvial Diagram section to the gallery that embeds the generated PNG
and links to its SVG counterpart. Explain that each flow width represents the
chosen metric in that tag's snapshot, rather than the delta between tags, and
that the columns follow the visualization-introduction release tags.

## Validation

`task samples-alluvial` regenerates both artifacts from the tagged history.
`task docs:gallery` copies the gallery image. The focused command test covers
tag resolution; the existing alluvial PNG/SVG goldens continue to protect
rendering geometry independently of the sample artifacts.
