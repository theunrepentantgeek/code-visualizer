# Enforce Radial Vector Construction

## Scope

Replace one manual polar-to-Cartesian conversion in
`internal/radialtree/render.go` with the established
`geometry.NewRadialVector` abstraction. The change is internal and must
preserve rendered output and observable behavior.

## Evidence

`addExternalLabel` computes a label radius and manually builds a vector with
`math.Cos` and `math.Sin`. `internal/geometry/ABSTRACTIONS.md` defines
`NewRadialVector(angle, length)` as the required construction for an
angle-and-radius displacement. The radial-tree layout already follows this
pattern. The renderer helper is unexported and has no serialized, persisted,
or error-contract boundary.

## Design

Set `labelDisplacement` with
`geometry.NewRadialVector(node.Angle, labelRadius)`. `NewRadialVector`
performs the identical multiplication and trigonometric calculations, so the
translated label position, anchor selection, rotation, and canvas output stay
unchanged.

No new type, helper, configuration, or public API is needed. Existing
radial-tree rendering tests cover PNG and SVG output and label-enabled tree
rendering.

## Validation

Run formatting and the `internal/radialtree` Go tests after the replacement.
