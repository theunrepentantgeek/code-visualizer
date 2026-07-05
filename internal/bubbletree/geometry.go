package bubbletree

import "math"

// ---------------------------------------------------------------------------
// Enclosing circle — Welzl's algorithm adapted for circles
// ---------------------------------------------------------------------------

type enclosure struct{ x, y, radius float64 }

// computeEnclosing returns the minimum enclosing circle of all nodes.
func computeEnclosing(nodes []BubbleNode) enclosure {
	if len(nodes) == 0 {
		return enclosure{}
	}

	if len(nodes) == 1 {
		return enclosure{nodes[0].X, nodes[0].Y, nodes[0].Radius}
	}

	circles := make([]enclosure, len(nodes))
	for i, n := range nodes {
		circles[i] = enclosure{n.X, n.Y, n.Radius}
	}

	return welzl(circles, [3]enclosure{}, 0, len(circles))
}

func welzl(pts []enclosure, boundary [3]enclosure, boundaryLen, n int) enclosure {
	if n == 0 || boundaryLen == 3 {
		return trivialEnclosing(boundary[:boundaryLen])
	}

	p := pts[n-1]
	d := welzl(pts, boundary, boundaryLen, n-1)

	if encloses(d, p) {
		return d
	}

	// p must lie on the boundary — recurse with it added.
	boundary[boundaryLen] = p

	return welzl(pts, boundary, boundaryLen+1, n-1)
}

// encloses reports whether outer fully contains inner (circle-in-circle test).
func encloses(outer, inner enclosure) bool {
	dx := inner.x - outer.x
	dy := inner.y - outer.y
	// Avoid math.Sqrt: sqrt(dist²)+r_inner <= r_outer+ε  ⟺  dist² <= (r_outer+ε-r_inner)²
	rhs := outer.radius + 1e-6 - inner.radius
	if rhs < 0 {
		return false
	}

	return dx*dx+dy*dy <= rhs*rhs
}

func trivialEnclosing(boundary []enclosure) enclosure {
	switch len(boundary) {
	case 0:
		return enclosure{}
	case 1:
		return boundary[0]
	case 2:
		return enclosingTwo(boundary[0], boundary[1])
	case 3:
		return enclosingThree(boundary[0], boundary[1], boundary[2])
	}

	return enclosure{} // unreachable
}

func enclosingTwo(a, b enclosure) enclosure {
	dx := b.x - a.x
	dy := b.y - a.y
	d := math.Sqrt(dx*dx + dy*dy)

	// One circle contains the other.
	if d+a.radius <= b.radius {
		return b
	}

	if d+b.radius <= a.radius {
		return a
	}

	r := (d + a.radius + b.radius) / 2

	// t ranges from 0 (at a) to 1 (at b).
	t := 0.5 + (b.radius-a.radius)/(2*d)

	return enclosure{
		x:      a.x + dx*t,
		y:      a.y + dy*t,
		radius: r,
	}
}

// enclosingThree solves for the minimum circle enclosing three boundary circles
// using the algebraic elimination approach.
func enclosingThree(a, b, c enclosure) enclosure {
	x1, y1, r1 := a.x, a.y, a.radius
	x2, y2, r2 := b.x, b.y, b.radius
	x3, y3, r3 := c.x, c.y, c.radius

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

	return enclosure{eu + fu*r, ev + fv*r, r}
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
func enclosingThreeFallback(a, b, c enclosure) enclosure {
	ab := enclosingTwo(a, b)
	ac := enclosingTwo(a, c)
	bc := enclosingTwo(b, c)

	if encloses(ab, c) {
		return ab
	}

	if encloses(ac, b) {
		return ac
	}

	if encloses(bc, a) {
		return bc
	}

	// Last resort: return the largest.
	best := ab
	if ac.radius > best.radius {
		best = ac
	}

	if bc.radius > best.radius {
		best = bc
	}

	return best
}
