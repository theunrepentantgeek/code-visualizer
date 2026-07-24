package spiral

import (
	"math"

	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

// BuildSurface creates an interpolated metric surface constrained to the spiral band.
func BuildSurface(layout SpiralLayout, values []float64, seed uint64) []surface.Triangle {
	if len(layout.Nodes) < 3 || len(layout.Nodes) != len(values) {
		return nil
	}

	originals := make([]surface.Point, len(layout.Nodes))
	for index, node := range layout.Nodes {
		originals[index] = surface.Point{
			X:     node.X,
			Y:     node.Y,
			Value: values[index],
		}
	}

	region := surface.Annulus{
		CX:          layout.CX,
		CY:          layout.CY,
		InnerRadius: layout.A,
		OuterRadius: layout.A + layout.B*layout.MaxTheta,
	}

	return surface.Build(region, originals, seed)
}

func surfaceSeed(layout SpiralLayout) uint64 {
	seed := uint64(len(layout.Nodes))
	for _, dimension := range [...]float64{
		layout.CX,
		layout.CY,
		layout.A,
		layout.B,
		layout.MaxTheta,
	} {
		seed ^= math.Float64bits(dimension)
		seed *= 1099511628211
	}

	return seed
}
