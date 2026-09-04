package bubbletree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// Enclosing circle — Welzl's algorithm adapted for circles
// ---------------------------------------------------------------------------

// welzlTolerance is the enclosure tolerance used by Welzl's algorithm to
// absorb floating-point rounding when deciding whether one circle already
// encloses another. It is kept local to this algorithm rather than folded
// into geometry.Circle.Encloses, which performs exact comparisons.
const welzlTolerance = 1e-6

// computeEnclosing returns the minimum enclosing circle of all nodes.
func computeEnclosing(nodes []BubbleNode) geometry.Circle {
	if len(nodes) == 0 {
		return geometry.Circle{}
	}

	if len(nodes) == 1 {
		return nodes[0].Geometry
	}

	circles := make([]geometry.Circle, len(nodes))
	for i, n := range nodes {
		circles[i] = n.Geometry
	}

	return welzl(circles, [3]geometry.Circle{}, 0, len(circles))
}

func welzl(pts []geometry.Circle, boundary [3]geometry.Circle, boundaryLen, n int) geometry.Circle {
	if n == 0 || boundaryLen == 3 {
		return trivialEnclosing(boundary[:boundaryLen])
	}

	p := pts[n-1]
	d := welzl(pts, boundary, boundaryLen, n-1)

	if enclosesWithin(d, p, welzlTolerance) {
		return d
	}

	// p must lie on the boundary — recurse with it added.
	boundary[boundaryLen] = p

	return welzl(pts, boundary, boundaryLen+1, n-1)
}

// enclosesWithin reports whether outer fully contains inner (circle-in-circle
// test), allowing tolerance of slack to absorb floating-point rounding. This
// is deliberately separate from geometry.Circle.Encloses, which uses exact
// comparisons with no hidden tolerance.
//
//nolint:unparam // tolerance is kept explicit so callers can see and vary the enclosure slack
func enclosesWithin(outer, inner geometry.Circle, tolerance float64) bool {
	if !outer.Valid() || !inner.Valid() {
		return false
	}

	return outer.Center.DistanceTo(inner.Center)+inner.Radius <=
		outer.Radius+tolerance
}

func trivialEnclosing(boundary []geometry.Circle) geometry.Circle {
	switch len(boundary) {
	case 0:
		return geometry.Circle{}
	case 1:
		return boundary[0]
	case 2:
		return enclosingTwo(boundary[0], boundary[1])
	case 3:
		return enclosingThree(boundary[0], boundary[1], boundary[2])
	}

	return geometry.Circle{} // unreachable
}

func enclosingTwo(a, b geometry.Circle) geometry.Circle {
	delta := a.Center.VectorTo(b.Center)
	d := math.Sqrt(delta.LengthSquared())

	// One circle contains the other.
	if d+a.Radius <= b.Radius {
		return b
	}

	if d+b.Radius <= a.Radius {
		return a
	}

	r := (d + a.Radius + b.Radius) / 2

	// t ranges from 0 (at a) to 1 (at b).
	t := 0.5 + (b.Radius-a.Radius)/(2*d)

	return geometry.NewCircle(a.Center.Translate(delta.Scale(t)), r)
}

// enclosingThree solves for the minimum circle enclosing three boundary circles
// using the algebraic elimination approach.
func enclosingThree(a, b, c geometry.Circle) geometry.Circle {
	x1, y1, r1 := a.Center.X, a.Center.Y, a.Radius
	x2, y2, r2 := b.Center.X, b.Center.Y, b.Radius
	x3, y3, r3 := c.Center.X, c.Center.Y, c.Radius

	s1 := x1*x1 + y1*y1 - r1*r1
	s2 := x2*x2 + y2*y2 - r2*r2
	s3 := x3*x3 + y3*y3 - r3*r3

	a1, b1, c1 := 2*(x1-x2), 2*(y1-y2), 2*(r2-r1)
	d1 := s1 - s2
	a2, b2, c2 := 2*(x1-x3), 2*(y1-y3), 2*(r3-r1)
	d2 := s1 - s3

	det := a1*b2 - a2*b1
	if math.Abs(det) < 1e-10 {
		return enclosingThreeFallback(a, b, c)
	}

	// Express u, v as linear functions of r:
	//   u = eu + fu*r
	//   v = ev + fv*r
	eu := (b2*d1 - b1*d2) / det
	fu := (b1*c2 - b2*c1) / det
	ev := (a1*d2 - a2*d1) / det
	fv := (a2*c1 - a1*c2) / det

	// Substitute into (u-x1)² + (v-y1)² = (r-r1)² to get a quadratic in r.
	u0 := eu - x1
	v0 := ev - y1

	qa := fu*fu + fv*fv - 1
	qb := 2 * (u0*fu + v0*fv + r1)
	qc := u0*u0 + v0*v0 - r1*r1

	minR := math.Max(r1, math.Max(r2, r3))

	r, ok := solveQuadraticForRadius(qa, qb, qc, minR)
	if !ok {
		return enclosingThreeFallback(a, b, c)
	}

	return geometry.NewCircle(geometry.NewPoint(eu+fu*r, ev+fv*r), r)
}

// solveQuadraticForRadius solves qa*r² + qb*r + qc = 0 for the smallest
// root >= minR. Returns (root, true) on success or (0, false) when no
// valid root exists.
func solveQuadraticForRadius(qa, qb, qc, minR float64) (float64, bool) {
	disc := qb*qb - 4*qa*qc
	if disc < 0 {
		disc = 0
	}

	if math.Abs(qa) < 1e-10 {
		if math.Abs(qb) < 1e-10 {
			return 0, false
		}

		r := -qc / qb
		if r < minR {
			return 0, false
		}

		return r, true
	}

	root1 := (-qb + math.Sqrt(disc)) / (2 * qa)
	root2 := (-qb - math.Sqrt(disc)) / (2 * qa)

	switch {
	case root1 >= minR && root2 >= minR:
		return math.Min(root1, root2), true
	case root1 >= minR:
		return root1, true
	case root2 >= minR:
		return root2, true
	default:
		return 0, false
	}
}

// enclosingThreeFallback returns the smallest pairwise enclosing circle
// that contains all three circles. Used when the algebraic solution is degenerate.
func enclosingThreeFallback(a, b, c geometry.Circle) geometry.Circle {
	ab := enclosingTwo(a, b)
	ac := enclosingTwo(a, c)
	bc := enclosingTwo(b, c)

	if enclosesWithin(ab, c, welzlTolerance) {
		return ab
	}

	if enclosesWithin(ac, b, welzlTolerance) {
		return ac
	}

	if enclosesWithin(bc, a, welzlTolerance) {
		return bc
	}

	// Last resort: return the largest.
	best := ab
	if ac.Radius > best.Radius {
		best = ac
	}

	if bc.Radius > best.Radius {
		best = bc
	}

	return best
}
