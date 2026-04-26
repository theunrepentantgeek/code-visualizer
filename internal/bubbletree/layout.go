// Package bubbletree implements data types and layout algorithms
// for circle-packing bubble tree visualizations.
package bubbletree

import (
	"cmp"
	"math"
	"slices"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
)

const (
	minFileRadius    = 2.0  // minimum circle radius for any file node
	siblingPadding   = 3.0  // gap between sibling circles at the same level
	parentPadding    = 6.0  // inset from parent circle edge (space for labels)
	LabelReservation = 14.0 // extra radius for directories that show a label
)

// Layout builds a bubble tree from root, positioning circles to fit within
// width × height pixels. sizeMetric controls the relative size of file circles
// and labels controls which labels are shown.
func Layout(root *model.Directory, width, height int, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	if root == nil {
		return BubbleNode{}
	}

	node := layoutDir(root, sizeMetric, labels)
	scaleToFit(&node, float64(width), float64(height))

	return node
}

// layoutDir recursively builds a BubbleNode for dir. Children are packed
// using the front-chain algorithm and enclosed. All coordinates are relative
// to the parent centre (local frame).
func layoutDir(dir *model.Directory, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	children := make([]BubbleNode, 0, len(dir.Dirs)+len(dir.Files))

	for _, d := range dir.Dirs {
		children = append(children, layoutDir(d, sizeMetric, labels))
	}

	for _, f := range dir.Files {
		children = append(children, layoutFile(f, sizeMetric, labels))
	}

	node := BubbleNode{
		Path:        dir.Path,
		Label:       dir.Name,
		IsDirectory: true,
		ShowLabel:   labels == LabelAll || labels == LabelFoldersOnly,
	}

	if len(children) == 0 {
		node.Radius = minFileRadius

		return node
	}

	// Sort by radius descending — improves packing density.
	slices.SortFunc(children, func(a, b BubbleNode) int {
		return cmp.Compare(b.Radius, a.Radius)
	})

	packCircles(children)

	enc := computeEnclosing(children)

	// Re-centre so the enclosing circle's centre becomes local origin.
	for i := range children {
		children[i].X -= enc.x
		children[i].Y -= enc.y
	}

	node.Radius = enc.radius + parentPadding
	if node.ShowLabel {
		node.Radius += LabelReservation
	}

	node.Children = children

	return node
}

func layoutFile(f *model.File, sizeMetric metric.Name, labels LabelMode) BubbleNode {
	r := math.Sqrt(fileMetricValue(f, sizeMetric))
	if r < minFileRadius {
		r = minFileRadius
	}

	return BubbleNode{
		Radius:    r,
		Path:      f.Path,
		Label:     f.Name,
		ShowLabel: labels == LabelAll,
	}
}

// fileMetricValue returns the metric value for f as a float64.
// Quantity is checked first, then Measure. Returns 0 if absent.
func fileMetricValue(f *model.File, m metric.Name) float64 {
	if q, ok := f.Quantity(m); ok {
		return float64(q)
	}

	if v, ok := f.Measure(m); ok {
		return v
	}

	return 0
}

// ---------------------------------------------------------------------------
// Front-chain circle packing
// ---------------------------------------------------------------------------

type point struct{ x, y float64 }

type frontNode struct {
	idx        int
	prev, next *frontNode
}

func linkNodes(a, b *frontNode) {
	a.next = b
	b.prev = a
}

// packCircles positions circles using a front-chain packing algorithm.
// On entry each circle must have its Radius set; on exit X and Y are set
// in a local coordinate frame centred roughly on the packing.
func packCircles(circles []BubbleNode) {
	n := len(circles)
	if n == 0 {
		return
	}

	placeInitialCircles(circles)

	if n <= 3 {
		return
	}

	chain := make([]frontNode, n)
	initFrontChain(chain)

	for i := 3; i < n; i++ {
		bestPos, bestAfter := findBestPlacement(circles, i, chain)

		if bestAfter != nil {
			circles[i].X, circles[i].Y = bestPos.x, bestPos.y

			// Insert into chain between bestAfter and bestAfter.next.
			chain[i].prev = bestAfter
			chain[i].next = bestAfter.next
			bestAfter.next.prev = &chain[i]
			bestAfter.next = &chain[i]
		} else {
			placeFallback(circles, i)
		}
	}
}

// placeInitialCircles positions the first min(len(circles), 3) circles.
// Circle 0 at origin, circle 1 along x-axis, circle 2 tangent to both.
func placeInitialCircles(circles []BubbleNode) {
	circles[0].X, circles[0].Y = 0, 0

	if len(circles) < 2 {
		return
	}

	circles[1].X = circles[0].Radius + circles[1].Radius + siblingPadding
	circles[1].Y = 0

	if len(circles) < 3 {
		return
	}

	p1, p2, ok := tangentPositions(circles[2].Radius, circles[0], circles[1])
	if ok {
		if p1.x*p1.x+p1.y*p1.y <= p2.x*p2.x+p2.y*p2.y {
			circles[2].X, circles[2].Y = p1.x, p1.y
		} else {
			circles[2].X, circles[2].Y = p2.x, p2.y
		}
	}
}

// initFrontChain initializes the circular linked list for the front chain.
// Nodes are linked as 0 → 2 → 1 → 0.
func initFrontChain(chain []frontNode) {
	for i := range chain {
		chain[i].idx = i
	}

	linkNodes(&chain[0], &chain[2])
	linkNodes(&chain[2], &chain[1])
	linkNodes(&chain[1], &chain[0])
}

// findBestPlacement scans the front chain to find the position closest to the
// origin where circle i can be placed tangent to an adjacent pair without
// overlapping any previously placed circle.
func findBestPlacement(circles []BubbleNode, i int, chain []frontNode) (point, *frontNode) {
	bestDist := math.MaxFloat64

	var bestPos point

	var bestAfter *frontNode

	start := &chain[0]
	cur := start

	for {
		pos, ok := bestTangentPosition(circles, i, cur)
		if ok {
			d := pos.x*pos.x + pos.y*pos.y
			if d < bestDist {
				bestDist = d
				bestPos = pos
				bestAfter = cur
			}
		}

		cur = cur.next
		if cur == start {
			break
		}
	}

	return bestPos, bestAfter
}

// bestTangentPosition returns the non-overlapping tangent position closest to
// the origin for placing circle i between the adjacent pair (cur, cur.next).
func bestTangentPosition(circles []BubbleNode, i int, cur *frontNode) (point, bool) {
	a, b := cur, cur.next
	tp1, tp2, tok := tangentPositions(circles[i].Radius, circles[a.idx], circles[b.idx])

	if !tok {
		return point{}, false
	}

	var best point

	bestDist := math.MaxFloat64
	found := false

	for _, pos := range [2]point{tp1, tp2} {
		if !anyOverlap(pos, circles[i].Radius, circles[:i], a.idx, b.idx) {
			d := pos.x*pos.x + pos.y*pos.y
			if d < bestDist {
				bestDist = d
				best = pos
				found = true
			}
		}
	}

	return best, found
}

// tangentPositions returns the two positions where a circle of radius rc
// can be placed tangent to circles a and b (including siblingPadding).
func tangentPositions(rc float64, a, b BubbleNode) (p1, p2 point, ok bool) {
	da := a.Radius + rc + siblingPadding
	db := b.Radius + rc + siblingPadding

	dx := b.X - a.X
	dy := b.Y - a.Y
	d := math.Sqrt(dx*dx + dy*dy)

	if d < 1e-10 || d > da+db+1e-6 || d < math.Abs(da-db)-1e-6 {
		return point{}, point{}, false
	}

	al := (da*da - db*db + d*d) / (2 * d)
	h2 := da*da - al*al

	if h2 < 0 {
		h2 = 0
	}

	h := math.Sqrt(h2)

	mx := a.X + al*dx/d
	my := a.Y + al*dy/d

	return point{mx + h*dy/d, my - h*dx/d},
		point{mx - h*dy/d, my + h*dx/d},
		true
}

// anyOverlap reports whether a circle at pos with the given radius overlaps
// any already-placed circle except the two tangent anchors.
func anyOverlap(pos point, radius float64, placed []BubbleNode, skipA, skipB int) bool {
	for j := range placed {
		if j == skipA || j == skipB {
			continue
		}

		dx := pos.x - placed[j].X
		dy := pos.y - placed[j].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		minSep := radius + placed[j].Radius + siblingPadding

		if dist < minSep-1e-6 {
			return true
		}
	}

	return false
}

// placeFallback positions circle i on the outer edge of the current packing
// when no valid front-chain tangent position exists.
func placeFallback(circles []BubbleNode, i int) {
	maxDist := 0.0

	for j := range i {
		d := math.Sqrt(circles[j].X*circles[j].X+circles[j].Y*circles[j].Y) + circles[j].Radius
		if d > maxDist {
			maxDist = d
		}
	}

	// Golden angle for even angular distribution.
	goldenAngle := math.Pi * (3 - math.Sqrt(5))

	angle := float64(i) * goldenAngle
	r := maxDist + circles[i].Radius + siblingPadding
	circles[i].X = r * math.Cos(angle)
	circles[i].Y = r * math.Sin(angle)
}

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

	return welzl(circles, nil, len(circles))
}

func welzl(pts []enclosure, boundary []enclosure, n int) enclosure {
	if n == 0 || len(boundary) == 3 {
		return trivialEnclosing(boundary)
	}

	p := pts[n-1]
	d := welzl(pts, boundary, n-1)

	if encloses(d, p) {
		return d
	}

	// p must lie on the boundary — recurse with it added.
	newBoundary := make([]enclosure, len(boundary)+1)
	copy(newBoundary, boundary)
	newBoundary[len(boundary)] = p

	return welzl(pts, newBoundary, n-1)
}

// encloses reports whether outer fully contains inner (circle-in-circle test).
func encloses(outer, inner enclosure) bool {
	dx := inner.x - outer.x
	dy := inner.y - outer.y

	return math.Sqrt(dx*dx+dy*dy)+inner.radius <= outer.radius+1e-6
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

// ---------------------------------------------------------------------------
// Top-down coordinate assignment — scales local layout to pixel canvas
// ---------------------------------------------------------------------------

// scaleToFit assigns absolute pixel coordinates to the entire tree,
// scaling and translating so the root circle fits within width × height.
func scaleToFit(node *BubbleNode, width, height float64) {
	if node.Radius <= 0 {
		node.X = width / 2
		node.Y = height / 2

		return
	}

	scale := math.Min(width, height) / (2 * node.Radius)

	node.X = width / 2
	node.Y = height / 2
	node.Radius *= scale

	applyScale(node, scale)
}

// applyScale recursively converts children from local to absolute coordinates.
func applyScale(parent *BubbleNode, scale float64) {
	for i := range parent.Children {
		child := &parent.Children[i]
		child.X = parent.X + child.X*scale
		child.Y = parent.Y + child.Y*scale
		child.Radius *= scale
		applyScale(child, scale)
	}
}
