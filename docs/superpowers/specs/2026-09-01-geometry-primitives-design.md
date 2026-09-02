# Geometry Primitives Design

## Purpose

Introduce a dependency-free `internal/geometry` package and migrate the rendering
system from repeated coordinate pairs and shape arithmetic to five shared
primitives: `Vector`, `Point`, `Size`, `Rect`, and `Circle`.

The work will be delivered as five stacked pull requests. Each pull request
introduces one primitive, tests its complete API, and migrates all suitable
consumers available at that point in the stack. Every pull request must build and
pass tests independently.

This is a behavior-preserving refactor. Existing raster and SVG golden output
must remain byte-for-byte unchanged; geometry corrections and rendering changes
are out of scope.

## Design Principles

- `internal/geometry` depends only on the Go standard library.
- Point and vector semantics remain distinct. Points represent locations;
  vectors represent displacement.
- Types use exported `float64` fields so literals remain concise and the zero
  value remains useful.
- Methods return new values rather than mutating receivers.
- Operations never silently clamp, reorder, normalize, or apply epsilon
  tolerances.
- Undefined operations return a boolean result rather than a plausible zero.
- Domain records keep domain data and contain geometry instead of becoming
  overloaded geometric primitives.
- Values that only resemble geometry, such as angles and scalar axis positions,
  remain domain-specific.

## Pull Request Stack

### PR 1: Vector

Add:

```go
type Vector struct {
    X float64
    Y float64
}
```

Factories and constants:

```go
var ZeroVector = Vector{}
func NewVector(x, y float64) Vector
func NewRadialVector(angle, length float64) Vector
```

Methods:

```go
func (v Vector) Valid() bool
func (v Vector) Add(other Vector) Vector
func (v Vector) Subtract(other Vector) Vector
func (v Vector) Scale(factor float64) Vector
func (v Vector) Dot(other Vector) float64
func (v Vector) Length() float64
func (v Vector) LengthSquared() float64
func (v Vector) Unit() (Vector, bool)
```

`ZeroVector` is the additive identity, exported as a `var` because structs
cannot be declared `const`. `NewRadialVector` converts polar coordinates
(angle in radians, length) to Cartesian components using the same convention
as radial-tree layout: `{X: length * cos(angle), Y: length * sin(angle)}`.

`Valid` requires finite components. `Unit` returns false for a zero or invalid
vector.

Migrate displacement and relative-coordinate concepts. In particular, radial
tree node positions become vectors because they are offsets from the canvas
centre. Offset parameters and intermediate displacement calculations use
`Vector` where this improves semantics without requiring `Point`, which arrives
in the next PR.

### PR 2: Point

Add:

```go
type Point struct {
    X float64
    Y float64
}
```

Methods and functions:

```go
func (p Point) Valid() bool
func (p Point) Translate(displacement Vector) Point
func (p Point) VectorTo(other Point) Vector
func (p Point) DistanceTo(other Point) float64
func (p Point) DistanceSquaredTo(other Point) float64
func Midpoint(a, b Point) Point
func Lerp(a, b Point, fraction float64) Point
```

`Valid` requires finite coordinates. Point-to-point addition is deliberately
absent. `DistanceTo` and `DistanceSquaredTo` return `NaN` when either operand is
invalid, preserving the invalidity rather than producing a useful-looking
distance. `Midpoint` and `Lerp` naturally propagate non-finite input through
floating-point arithmetic.

Migrate absolute positions used by canvas, bubble tree, spiral, scatter, donut
tree, and surface calculations.

The current `surface.Point` is not a geometric point because it also carries an
interpolated value and interpolation state. Replace it with:

```go
type Sample struct {
    Position    geometry.Point
    Value       float64
    unsupported bool
    Original    bool
}
```

Surface algorithms operate on `Sample` records while delegating coordinate
arithmetic to `Sample.Position`.

Normalized gradient focus coordinates remain a separate canvas-domain type:

```go
type GradientPoint struct {
    X float64
    Y float64
}
```

Backends explicitly convert a `GradientPoint` into an absolute
`geometry.Point`. This prevents normalized coordinates from being accidentally
mixed with pixel positions.

The existing `canvas.Position` and `canvas/model.Position` names may be aliases
within this PR to keep the stack reviewable. Consumers should migrate toward
`geometry.Point`, and duplicate definitions or transitional aliases must be
removed by the end of the stack. Because `internal/geometry` has no project
dependencies, importing it cannot create a cycle back into canvas.

### PR 3: Size

Add:

```go
type Size struct {
    Width  float64
    Height float64
}
```

Methods:

```go
func (s Size) Valid() bool
func (s Size) Empty() bool
func (s Size) Area() float64
func (s Size) Scale(factor float64) Size
func (s Size) AspectRatio() (float64, bool)
```

`Valid` requires finite, non-negative dimensions. `Empty` is true for a valid
size when either dimension is zero. `Area` follows ordinary floating-point
arithmetic; callers that require a meaningful area check `Valid` first.
`AspectRatio` returns false for invalid sizes or a zero height.

Migrate paired floating-point render and layout dimensions. Configuration values
remain unchanged because their pointer-valued integer fields represent optional
user input rather than resolved geometry.

As with `Position`, the existing canvas `Size` aliases may provide a transitional
bridge but should not preserve a duplicate implementation.

### PR 4: Rect

Add:

```go
type Rect struct {
    Min Point
    Max Point
}
```

Constructor and methods:

```go
func RectFromPositionSize(position Point, size Size) Rect
func (r Rect) Valid() bool
func (r Rect) Empty() bool
func (r Rect) Width() float64
func (r Rect) Height() float64
func (r Rect) Size() Size
func (r Rect) Center() Point
func (r Rect) Contains(point Point) bool
func (r Rect) Translate(displacement Vector) Rect
func (r Rect) Inset(amount float64) (Rect, bool)
func (r Rect) ExpandToInclude(point Point) (Rect, bool)
func (r Rect) Union(other Rect) (Rect, bool)
```

`Valid` requires finite points ordered `Min.X <= Max.X` and
`Min.Y <= Max.Y`. `Empty` is true for a valid rectangle with zero width or
height. `Contains` includes the boundary and returns false for an invalid
rectangle or point.

`RectFromPositionSize` does not reorder or repair invalid input.
`Inset` returns false if the amount is non-finite or would invert either axis;
negative amounts expand the rectangle. `ExpandToInclude` and `Union` return
false if either operand is invalid. This keeps their results unambiguous and
prevents the methods from silently repairing unordered rectangles.

Migrate:

- surface bounds,
- shared pipeline drawing bounds,
- scatter plot bounds,
- treemap rectangle and label bounds,
- bubble occupied bounds,
- canvas rectangle geometry where the whole rectangle is passed as a unit.

Integer image bounds stay as `image.Rectangle`. Pipeline drawing bounds may
retain explicit integer conversion at canvas allocation and reservation
boundaries, but layout geometry uses `geometry.Rect`.

Treemap visual records continue to contain labels, hierarchy, and directory
chrome. They embed or contain a `geometry.Rect`; they do not become aliases for
it.

### PR 5: Circle

Add:

```go
type Circle struct {
    Center Point
    Radius float64
}
```

Methods:

```go
func (c Circle) Valid() bool
func (c Circle) Contains(point Point) bool
func (c Circle) Encloses(other Circle) bool
func (c Circle) Intersects(other Circle) bool
func (c Circle) Bounds() Rect
func (c Circle) Translate(displacement Vector) Circle
```

`Valid` requires a valid centre and a finite, non-negative radius. Boundary
points count as contained and tangent circles count as intersecting.
Predicates return false for invalid operands. These generic predicates use exact
mathematical comparisons and no hidden tolerance.

Algorithms that intentionally require tolerance, such as minimum enclosing
circle calculations, keep the tolerance visible at the algorithm call site
rather than changing `Circle.Encloses` semantics.

Migrate canvas discs, bubble nodes and enclosing-circle calculations, scatter
points, radial nodes, spiral nodes, and other records where centre and radius
form one geometric concept. Domain records retain metadata, hierarchy, angles,
and metrics alongside a `geometry.Circle`.

Annuli and donut sectors remain domain types. Their centre and constituent
circle calculations use geometry primitives, but the geometry package does not
gain `Annulus` or `Sector` in this series.

## Migration Boundaries

The following values remain outside `internal/geometry`:

- `GradientPoint`, because normalized gradient space is distinct from pixel
  space;
- angles, sweeps, and scalar axis positions;
- optional configuration dimensions;
- integer raster bounds represented by `image.Rectangle`;
- annuli and donut sectors;
- visualization records containing labels, metrics, hierarchy, or model
  references.

The migration replaces coordinate pairs only when they express a reusable
geometric concept. It does not flatten visualization domain models into generic
shapes.

## Compatibility and Dependencies

The PRs form a stack in this order:

```text
Vector → Point → Size → Rect → Circle
```

Each branch is based on the branch immediately before it. No PR may depend on
code introduced later in the stack. Temporary aliases are acceptable when they
make an individual diff substantially easier to review, but the final PR must
remove duplicate primitive definitions and unnecessary compatibility aliases.

Existing public behavior, command-line behavior, configuration formats, exports,
and rendered output remain unchanged.

## Error and Invalid-Value Policy

Primitive methods do not return errors. Geometry values are local computation
values, and invalidity is represented by non-finite or incorrectly ordered
fields.

- `Point.Valid` and `Vector.Valid` require finite components.
- `Size.Valid` requires finite, non-negative dimensions.
- `Rect.Valid` requires finite, ordered endpoints.
- `Circle.Valid` requires a finite centre and finite, non-negative radius.
- Predicates return false for invalid operands.
- Operations that are undefined for valid inputs, such as normalizing a zero
  vector or taking an aspect ratio with zero height, return `(value, false)`.
- Constructors and operations never silently repair invalid values.

Callers that receive external input continue to report errors at their existing
domain boundary. The geometry package does not introduce logging or
application-specific errors.

## Testing

Each primitive receives table-driven unit tests covering:

- the zero value,
- ordinary values,
- boundary inclusion and tangency,
- degenerate but valid geometry,
- negative dimensions and radii,
- unordered rectangle endpoints,
- `NaN` and positive and negative infinity in every field,
- undefined operations and their boolean result,
- algebraic identities such as translation by the zero vector and distance
  symmetry,
- non-mutation of receivers.

Each PR updates all affected visualization, canvas, backend, and surface tests.
The complete existing test suite and golden tests run for every PR. Golden files
must not be regenerated: any changed raster or SVG output is a regression to
investigate.

Focused benchmarks are not required because these small value methods are
expected to inline and the refactor does not change algorithmic complexity.
Existing benchmarks provide sufficient regression coverage.

## Completion Criteria

The stack is complete when:

1. `internal/geometry` owns the five approved primitives and their tested APIs.
2. Suitable production consumers use those primitives instead of repeated
   coordinate and shape fields.
3. `surface.Sample` cleanly separates interpolation state from position.
4. Gradient focus remains explicitly distinct from absolute geometry.
5. Duplicate point, size, rectangle, bounds, and circle implementations have
   been removed where they represent the same concept.
6. Every PR is independently buildable and testable.
7. Existing golden raster and SVG output is unchanged.
