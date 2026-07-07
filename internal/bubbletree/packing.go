package bubbletree

import "math"

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
		// Avoid math.Sqrt: dist < minSep-ε  ⟺  dist² < (minSep-ε)²  (when minSep-ε > 0)
		minSep := radius + placed[j].Radius + siblingPadding - 1e-6
		if minSep > 0 && dx*dx+dy*dy < minSep*minSep {
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
