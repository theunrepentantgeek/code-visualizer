package surface

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestDiagBoundaryCoverage(t *testing.T) {
	annulus := Annulus{CX: 500, CY: 500, InnerRadius: 100, OuterRadius: 300}

	random := rand.New(rand.NewPCG(1, 2))

	originals := make([]Point, 0, 300)
	for range 300 {
		angle := random.Float64() * 2 * math.Pi
		radius := 100 + random.Float64()*200
		originals = append(originals, Point{
			X:     500 + radius*math.Cos(angle),
			Y:     500 + radius*math.Sin(angle),
			Value: radius,
		})
	}

	triangles := Build(annulus, originals, 7)
	t.Logf("triangles: %d", len(triangles))

	used := make(map[[2]float64]bool)
	for _, triangle := range triangles {
		for _, point := range triangle.Points {
			used[[2]float64{point.X, point.Y}] = true
		}
	}

	loops := BoundaryLoops(annulus, MaxTriangleEdge)
	for loopIndex, loop := range loops {
		count := 0

		for _, point := range loop {
			if used[[2]float64{point.X, point.Y}] {
				count++
			}
		}

		t.Logf("loop %d: %d of %d boundary samples used", loopIndex, count, len(loop))
	}

	// how many boundary points fail the radius test?
	outerFail := 0
	for _, point := range loops[0] {
		dx := point.X - 500
		dy := point.Y - 500

		if dx*dx+dy*dy > 300*300 {
			outerFail++
		}
	}

	t.Logf("outer samples failing radius test: %d of %d", outerFail, len(loops[0]))
}
