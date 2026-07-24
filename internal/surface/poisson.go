package surface

import (
	"math"
	"math/rand/v2"
	"reflect"
)

const attemptsPerActivePoint = 30

type gridCell struct {
	x int
	y int
}

type poissonGrid struct {
	bounds   Rect
	cellSize float64
	cells    map[gridCell][]Point
}

// Sample returns infill points generated with Bridson Poisson-disk sampling.
func Sample(region Region, originals []Point, minimumDistance float64, seed uint64) []Point {
	if !isValidRegion(region) ||
		minimumDistance <= 0 ||
		!isFinite(minimumDistance) ||
		minimumDistance*minimumDistance == 0 {
		return nil
	}

	bounds := region.Bounds()
	grid := newPoissonGrid(bounds, minimumDistance)
	active := make([]Point, 0, len(originals)+1)
	for _, original := range originals {
		if !isFinitePoint(original) {
			continue
		}

		grid.insert(original)
		active = append(active, original)
	}

	random := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	samples := make([]Point, 0)
	if len(active) == 0 {
		initial, found := initialSample(region, bounds, random)
		if !found {
			return samples
		}

		grid.insert(initial)
		active = append(active, initial)
		samples = append(samples, initial)
	}

	for len(active) > 0 {
		activeIndex := random.IntN(len(active))
		activePoint := active[activeIndex]
		accepted := false

		for range attemptsPerActivePoint {
			candidate := annulusCandidate(activePoint, minimumDistance, random)
			if !region.Contains(candidate.X, candidate.Y) || grid.hasNearby(candidate, minimumDistance) {
				continue
			}

			grid.insert(candidate)
			active = append(active, candidate)
			samples = append(samples, candidate)
			accepted = true
			break
		}

		if !accepted {
			active[activeIndex] = active[len(active)-1]
			active = active[:len(active)-1]
		}
	}

	return samples
}

func isValidRegion(region Region) bool {
	if region == nil {
		return false
	}

	value := reflect.ValueOf(region)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if value.IsNil() {
			return false
		}
	}

	bounds := region.Bounds()
	return isFinite(bounds.MinX) &&
		isFinite(bounds.MinY) &&
		isFinite(bounds.MaxX) &&
		isFinite(bounds.MaxY) &&
		bounds.MinX < bounds.MaxX &&
		bounds.MinY < bounds.MaxY
}

func initialSample(region Region, bounds Rect, random *rand.Rand) (Point, bool) {
	for range attemptsPerActivePoint {
		candidate := Point{
			X: bounds.MinX + random.Float64()*(bounds.MaxX-bounds.MinX),
			Y: bounds.MinY + random.Float64()*(bounds.MaxY-bounds.MinY),
		}
		if region.Contains(candidate.X, candidate.Y) {
			return candidate, true
		}
	}

	return Point{}, false
}

func annulusCandidate(point Point, minimumDistance float64, random *rand.Rand) Point {
	angle := 2 * math.Pi * random.Float64()
	radius := minimumDistance * math.Sqrt(1+3*random.Float64())

	return Point{
		X: point.X + radius*math.Cos(angle),
		Y: point.Y + radius*math.Sin(angle),
	}
}

func newPoissonGrid(bounds Rect, minimumDistance float64) poissonGrid {
	return poissonGrid{
		bounds:   bounds,
		cellSize: minimumDistance / math.Sqrt2,
		cells:    make(map[gridCell][]Point),
	}
}

func (g poissonGrid) insert(point Point) {
	cell := g.cellFor(point)
	g.cells[cell] = append(g.cells[cell], point)
}

func (g poissonGrid) hasNearby(candidate Point, minimumDistance float64) bool {
	cell := g.cellFor(candidate)
	cellRadius := int(math.Ceil(minimumDistance / g.cellSize))
	minimumDistanceSquared := minimumDistance * minimumDistance

	for x := cell.x - cellRadius; x <= cell.x+cellRadius; x++ {
		for y := cell.y - cellRadius; y <= cell.y+cellRadius; y++ {
			for _, point := range g.cells[gridCell{x: x, y: y}] {
				dx := candidate.X - point.X
				dy := candidate.Y - point.Y
				if dx*dx+dy*dy < minimumDistanceSquared {
					return true
				}
			}
		}
	}

	return false
}

func (g poissonGrid) cellFor(point Point) gridCell {
	return gridCell{
		x: int(math.Floor((point.X - g.bounds.MinX) / g.cellSize)),
		y: int(math.Floor((point.Y - g.bounds.MinY) / g.cellSize)),
	}
}

func isFinitePoint(point Point) bool {
	return isFinite(point.X) && isFinite(point.Y)
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
