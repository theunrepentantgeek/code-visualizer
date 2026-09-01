package bubbletree

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// ---------------------------------------------------------------------------
// Front-chain circle packing
// ---------------------------------------------------------------------------

type frontNode struct {
	idx        int
	prev, next *frontNode
}

func linkNodes(a, b *frontNode) {
	a.next = b
	b.prev = a
}

// packCircles positions circles using a front-chain packing algorithm.
// On entry each circle must have its Radius set; on exit Position is set
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
			circles[i].Position = bestPos //nolint:gosec // G602 false positive: i < len(circles)

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
	circles[0].Position = geometry.Point{}

	if len(circles) < 2 {
		return
	}

	circles[1].Position = geometry.Point{X: circles[0].Radius + circles[1].Radius + siblingPadding}

	if len(circles) < 3 {
		return
	}

	p1, p2, ok := tangentPositions(circles[2].Radius, circles[0], circles[1])
	if ok {
		if p1.DistanceSquaredTo(geometry.Point{}) <= p2.DistanceSquaredTo(geometry.Point{}) {
			circles[2].Position = p1
		} else {
			circles[2].Position = p2
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
func findBestPlacement(circles []BubbleNode, i int, chain []frontNode) (geometry.Point, *frontNode) {
	bestDist := math.MaxFloat64

	var bestPos geometry.Point

	var bestAfter *frontNode

	start := &chain[0]
	cur := start

	for {
		pos, ok := bestTangentPosition(circles, i, cur)
		if ok {
			d := pos.DistanceSquaredTo(geometry.Point{})
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
func bestTangentPosition(circles []BubbleNode, i int, cur *frontNode) (geometry.Point, bool) {
	a, b := cur, cur.next
	tp1, tp2, tok := tangentPositions(circles[i].Radius, circles[a.idx], circles[b.idx])

	if !tok {
		return geometry.Point{}, false
	}

	var best geometry.Point

	bestDist := math.MaxFloat64
	found := false

	for _, pos := range [2]geometry.Point{tp1, tp2} {
		if !anyOverlap(pos, circles[i].Radius, circles[:i], a.idx, b.idx) {
			d := pos.DistanceSquaredTo(geometry.Point{})
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
func tangentPositions(rc float64, a, b BubbleNode) (p1, p2 geometry.Point, ok bool) {
	da := a.Radius + rc + siblingPadding
	db := b.Radius + rc + siblingPadding

	delta := a.Position.VectorTo(b.Position)
	d := math.Sqrt(delta.LengthSquared())

	if d < 1e-10 || d > da+db+1e-6 || d < math.Abs(da-db)-1e-6 {
		return geometry.Point{}, geometry.Point{}, false
	}

	al := (da*da - db*db + d*d) / (2 * d)
	h2 := da*da - al*al

	if h2 < 0 {
		h2 = 0
	}

	h := math.Sqrt(h2)

	midpoint := a.Position.Translate(geometry.Vector{
		X: al * delta.X / d,
		Y: al * delta.Y / d,
	})
	offset := geometry.Vector{X: h * delta.Y / d, Y: -h * delta.X / d}

	return midpoint.Translate(offset), midpoint.Translate(offset.Scale(-1)), true
}

// anyOverlap reports whether a circle at pos with the given radius overlaps
// any already-placed circle except the two tangent anchors.
func anyOverlap(pos geometry.Point, radius float64, placed []BubbleNode, skipA, skipB int) bool {
	for j := range placed {
		if j == skipA || j == skipB {
			continue
		}

		// Avoid math.Sqrt: dist < minSep-ε  ⟺  dist² < (minSep-ε)²  (when minSep-ε > 0)
		minSep := radius + placed[j].Radius + siblingPadding - 1e-6
		if minSep > 0 && pos.DistanceSquaredTo(placed[j].Position) < minSep*minSep {
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
		offset := geometry.Point{}.VectorTo(circles[j].Position)
		d := math.Sqrt(offset.LengthSquared()) + circles[j].Radius
		if d > maxDist {
			maxDist = d
		}
	}

	// Golden angle for even angular distribution.
	goldenAngle := math.Pi * (3 - math.Sqrt(5))

	angle := float64(i) * goldenAngle
	r := maxDist + circles[i].Radius + siblingPadding
	circles[i].Position = geometry.Point{X: r * math.Cos(angle), Y: r * math.Sin(angle)}
}
