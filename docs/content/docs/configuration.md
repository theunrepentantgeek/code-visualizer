---
title: Configuration
weight: 9
---

`codeviz` reads an optional configuration file (`.yaml`, `.yml`, or `.json`)
supplied with the `--config` flag. This page documents the available keys.
Configuration parsing is strict: unknown keys and malformed values are rejected,
and the error identifies the line containing the problem.

{{< callout type="info" >}}
This reference is being expanded. For the authoritative list of flags, run
`codeviz <visualization> --help`, and see the [Usage]({{< relref "/docs/usage" >}}) page.
{{< /callout >}}

## Alluvial

Configure an alluvial release comparison under `alluvial`. `references` is an
ordered list with at least two tag, commit ID, or date references. Tags are the
typical release input. The selected `metric` supplies each directory's
per-snapshot flow width; it is not calculated as a change between references.

```yaml
alluvial:
  references:
    - tag:v1.0
    - tag:v2.0
  metric: file-lines
  fill:
    metric: file-lines.delta
    palette: temperature
  expand:
    - cmd
    - internal
  constantBands: merge
```

CLI `--reference` values replace the configured `references` list while
preserving their order. An empty reference selects the repository's `HEAD`
commit. Repeat `--expand` to show only a selected directory's direct children.
For an alluvial `fill.metric`, `.delta` compares the final reference with the
first, while `.stepdelta` compares each later reference with its predecessor;
the first `.stepdelta` value is unavailable and displays as `-`.
`--include` and `--exclude` remain normal file filters: they bound which
directories can be represented, do not imply hierarchy expansion, and do not
create an `Other` aggregate.

`constantBands` is optional and accepts `hide`, `mute`, or `merge`. A constant
band has the same width-metric value at every configured reference. `hide`
removes those bands, `mute` keeps them as unlabeled light-grey context, and
`merge` combines adjacent constant bands in path order before rendering them
the same way. If omitted, constant bands retain the normal palette and labels.
