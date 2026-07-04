package raster

import (
	"image"
	"math"
)

// renderRadialGradientPixels fills pixels in [x0,x1)×[y0,y1) with a radial gradient.
// fx,fy is the gradient focus in image coordinates; invScale maps distance to t ∈ [0,1].
// If clipR > 0, pixels outside the circle centred at (clipCx, clipCy) with radius clipR
// are skipped. Pass clipR=0 to paint the full rectangle without clipping.
func renderRadialGradientPixels(
	img *image.RGBA,
	x0, y0, x1, y1 int,
	fx, fy float64,
	invScale float64,
	lerp gradientLerp,
	clipCx, clipCy, clipR float64,
) {
	hasClip := clipR > 0
	r2 := clipR * clipR

	for py := y0; py < y1; py++ {
		dy := float64(py) + 0.5 - fy
		dy2 := dy * dy

		var cdy2 float64
		if hasClip {
			cdy := float64(py) + 0.5 - clipCy
			cdy2 = cdy * cdy
		}

		for px := x0; px < x1; px++ {
			if hasClip {
				cdx := float64(px) + 0.5 - clipCx
				if cdx*cdx+cdy2 > r2 {
					continue
				}
			}

			dx := float64(px) + 0.5 - fx
			dist := math.Sqrt(dx*dx + dy2)
			img.SetRGBA(px, py, lerp.at(min(dist*invScale, 1.0)))
		}
	}
}
