# SVG Three-Decimal Precision Design

## Goal

Improve SVG rendering when users zoom into visualizations by serializing every
floating-point value in the SVG backend with three decimal places.

## Scope

Update `internal/canvas/svg/backend.go` so all values derived from `float64`
inputs use fixed `%.3f` formatting. This includes:

- coordinates, dimensions, radii, and path points;
- stroke widths, font sizes, and rotation values;
- radial-gradient focus percentages and their cache keys; and
- alpha channel values emitted by `rgba(...)`.

Values that are inherently integral or literal remain unchanged, including
canvas dimensions, RGB channels, element IDs, arc flags, and fixed percentage
constants.

## Implementation

Change the existing format verbs in place rather than adding a formatting
helper. The backend has a single serialization file, and direct format strings
keep each SVG element readable while avoiding an abstraction that would not
reduce meaningful duplication.

The output policy is fixed-width: whole values are emitted with trailing
zeroes, such as `10.000`. This makes the precision rule consistent and
deterministic across SVG elements.

## Testing

Add focused SVG backend assertions using non-round values so the tests prove
that each category of floating-point output is rounded to and emitted with
exactly three decimal places. Update existing expectations and golden SVG
fixtures whose serialized values change.

Run the targeted SVG backend tests first, then the repository CI task to cover
all generated SVG snapshots and project checks.

## Compatibility

The SVG structure and rendering semantics remain unchanged. Files will be
slightly larger because numeric values gain decimal digits, and snapshot output
will change, but the CLI and backend interfaces are unaffected.
