# No-colour encoding sentinel

## Purpose

Make intentional absence of a colour encoding explicit at every visualization
call site without changing rendering behavior.

## Design

`internal/viz` exports `NoColourEncoding`, a package-level
`ColourEncoding{}` value. It represents an unset colour channel, retaining
the existing zero-value semantics: `NoColourEncoding.IsSet()` is false.

All production and test uses of an empty `viz.ColourEncoding{}` literal are
replaced with this sentinel. Resolved, non-empty encodings remain unchanged.

## Validation

The existing `viz` tests gain coverage for the sentinel's unset invariant.
Affected visualization package tests and the complete Go test suite confirm
that the explicit representation preserves rendering behavior.
