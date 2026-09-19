# ColourEncoding Enforcement Design

## Goal

Keep a colour channel's metric and palette coupled as `viz.ColourEncoding`
from configuration resolution through visualization-specific ink construction.
Preserve all current output, fallback, error, logging, legend, and zero-value
behaviour.

## Changes

1. Donut tree will resolve its already-normalized directory fill and border
   metrics with `stages.ResolveColourEncoding` instead of constructing
   `viz.ColourEncoding` literals and separately resolving palettes.
2. Radial tree's directory-ink helper will accept fill and border
   `viz.ColourEncoding` values instead of four metric/palette arguments.
3. Scatter's metric-ink helper will accept a `viz.ColourEncoding` rather than
   separate metric and palette arguments.
4. Spiral's bucket-ink helper will accept a `viz.ColourEncoding` rather than
   separate metric and palette arguments. Its bucket value accessors remain
   separate because they describe the source data, not the colour channel.
5. The `internal/viz` abstraction documentation will describe
   `ColourEncoding`, its zero-value invariant, its resolver, and the rule
   against splitting the pair into parallel parameters.

## Delivery and Validation

Each refactor is a separate cohesive commit. Before each commit, format the
changed Go files, run the affected package's unit tests, and build the
repository. The documentation update is a fifth, documentation-only commit.
