package raster

import "image/color"

// gradientLerp precomputes float64 channel values and deltas for linearly
// interpolating between two RGBA colours. Create with newGradientLerp; call
// at(t) for any t ∈ [0,1].
//
// The precomputation happens once per gradient, not once per pixel, keeping
// the hot inner rendering loops free of repeated uint8→float64 conversions.
type gradientLerp struct {
	cr, cg, cb, ca float64
	dr, dg, db, da float64
}

func newGradientLerp(center, edge color.RGBA) gradientLerp {
	cr := float64(center.R)
	cg := float64(center.G)
	cb := float64(center.B)
	ca := float64(center.A)

	return gradientLerp{
		cr: cr, cg: cg, cb: cb, ca: ca,
		dr: float64(edge.R) - cr,
		dg: float64(edge.G) - cg,
		db: float64(edge.B) - cb,
		da: float64(edge.A) - ca,
	}
}

// at returns the interpolated colour at t, where t=0 gives center and t=1 gives edge.
// t is clamped to [0,1] by the caller.
func (l *gradientLerp) at(t float64) color.RGBA {
	return color.RGBA{
		R: uint8(l.cr + l.dr*t),
		G: uint8(l.cg + l.dg*t),
		B: uint8(l.cb + l.db*t),
		A: uint8(l.ca + l.da*t),
	}
}
