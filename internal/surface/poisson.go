package surface

import (
	"math"
	"math/rand/v2"
	"reflect"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

const attemptsPerActivePoint = 30

type gridCell struct {
	x int
	y int
}

type poissonGrid struct {
	bounds   Rect
	cellSize float64
	cells    map[gridCell][]Sample
}

// PoissonSamples returns infill samples generated with Bridson Poisson-disk sampling.
func PoissonSamples(region Region, originals []Sample, minimumDistance float64, seed uint64) []Sample {
	if !isValidRegion(region) ||
		minimumDistance <= 0 ||
		!isFinite(minimumDistance) ||
		minimumDistance*minimumDistance == 0 {
		return nil
	}

	bounds := region.Bounds()
	grid := newPoissonGrid(bounds, minimumDistance)

	active := make([]Sample, 0, len(originals)+1)
	for _, original := range originals {
		if !isFiniteSample(original) {
			continue
		}

		grid.insert(original)
		active = append(active, original)
	}

	//nolint:gosec // A seeded PCG intentionally makes sampled surfaces deterministic.
	random := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	samples := make([]Sample, 0)

	if len(active) == 0 {
		initial, found := initialSample(region, bounds, random)
		if !found {
			return samples
		}

		grid.insert(initial)
		active = append(active, initial)
		samples = append(samples, initial)
	}

	return sampleInfill(region, &grid, active, samples, minimumDistance, random)
}

func sampleInfill(
	region Region,
	grid *poissonGrid,
	active, samples []Sample,
	minimumDistance float64,
	random *rand.Rand,
) []Sample {
	for len(active) > 0 {
		activeIndex := random.IntN(len(active))
		activeSample := active[activeIndex]
		candidate, accepted := sampleCandidate(region, *grid, activeSample, minimumDistance, random)

		if !accepted {
			active[activeIndex] = active[len(active)-1]
			active = active[:len(active)-1]

			continue
		}

		grid.insert(candidate)
		active = append(active, candidate)
		samples = append(samples, candidate)
	}

	return samples
}

func sampleCandidate(
	region Region,
	grid poissonGrid,
	activeSample Sample,
	minimumDistance float64,
	random *rand.Rand,
) (Sample, bool) {
	for range attemptsPerActivePoint {
		candidate := annulusCandidate(activeSample, minimumDistance, random)
		if !region.Contains(candidate.Position.X, candidate.Position.Y) ||
			grid.hasNearby(candidate, minimumDistance) {
			continue
		}

		return candidate, true
	}

	return Sample{}, false
}

func isValidRegion(region Region) bool {
	if region == nil || isTypedNilRegion(region) {
		return false
	}

	return validBounds(region.Bounds())
}

func isTypedNilRegion(region Region) bool {
	return isNilInterfaceValue(region)
}

func isNilInterfaceValue(value any) bool {
	if value == nil {
		return true
	}

	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflectValue.IsNil()
	default:
		return false
	}
}

func validBounds(bounds Rect) bool {
	return isFinite(bounds.MinX) &&
		isFinite(bounds.MinY) &&
		isFinite(bounds.MaxX) &&
		isFinite(bounds.MaxY) &&
		bounds.MinX < bounds.MaxX &&
		bounds.MinY < bounds.MaxY
}

func initialSample(region Region, bounds Rect, random *rand.Rand) (Sample, bool) {
	for range attemptsPerActivePoint {
		candidate := Sample{
			Position: geometry.Point{
				X: bounds.MinX + random.Float64()*(bounds.MaxX-bounds.MinX),
				Y: bounds.MinY + random.Float64()*(bounds.MaxY-bounds.MinY),
			},
		}
		if region.Contains(candidate.Position.X, candidate.Position.Y) {
			return candidate, true
		}
	}

	return Sample{}, false
}

func annulusCandidate(sample Sample, minimumDistance float64, random *rand.Rand) Sample {
	angle := 2 * math.Pi * random.Float64()
	radius := minimumDistance * math.Sqrt(1+3*random.Float64())

	return Sample{
		Position: sample.Position.Translate(geometry.Vector{
			X: radius * math.Cos(angle),
			Y: radius * math.Sin(angle),
		}),
	}
}

func newPoissonGrid(bounds Rect, minimumDistance float64) poissonGrid {
	return poissonGrid{
		bounds:   bounds,
		cellSize: minimumDistance / math.Sqrt2,
		cells:    make(map[gridCell][]Sample),
	}
}

func (g poissonGrid) insert(sample Sample) {
	cell := g.cellFor(sample)
	g.cells[cell] = append(g.cells[cell], sample)
}

func (g poissonGrid) hasNearby(candidate Sample, minimumDistance float64) bool {
	cell := g.cellFor(candidate)
	cellRadius := int(math.Ceil(minimumDistance / g.cellSize))
	minimumDistanceSquared := minimumDistance * minimumDistance

	for x := cell.x - cellRadius; x <= cell.x+cellRadius; x++ {
		for y := cell.y - cellRadius; y <= cell.y+cellRadius; y++ {
			for _, sample := range g.cells[gridCell{x: x, y: y}] {
				if candidate.Position.DistanceSquaredTo(sample.Position) < minimumDistanceSquared {
					return true
				}
			}
		}
	}

	return false
}

func (g poissonGrid) cellFor(sample Sample) gridCell {
	return gridCell{
		x: int(math.Floor((sample.Position.X - g.bounds.MinX) / g.cellSize)),
		y: int(math.Floor((sample.Position.Y - g.bounds.MinY) / g.cellSize)),
	}
}

func isFiniteSample(sample Sample) bool {
	return sample.Position.Valid()
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
