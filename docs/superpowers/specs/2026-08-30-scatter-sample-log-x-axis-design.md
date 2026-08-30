# Scatter Sample Logarithmic X Axis

**Date:** 2026-08-30

## Summary

Change the checked-in scatter sample to use a logarithmic horizontal axis so
files with smaller sizes are less tightly clustered.

## Changes

- Add `xScale: log` to `samples/scatter/code-visualizer.yml`.
- Update `samples/scatter/README.md` to identify the X axis as logarithmic and
  explain that file sizes are spread according to their order of magnitude.
- Regenerate `samples/scatter/code-visualizer.png` and
  `samples/scatter/code-visualizer.svg` with the existing
  `task samples-scatter` command.

The scatter implementation, command-line interface, documentation thumbnail,
and other samples remain unchanged. The documentation thumbnail already uses a
logarithmic X axis.

## Validation

Run `task samples-scatter` successfully and confirm that its only resulting
sample changes are the configuration, README, PNG, and SVG files listed above.
