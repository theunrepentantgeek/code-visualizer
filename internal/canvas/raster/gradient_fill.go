package raster

import (
	"image"
	"math"
)

// radialClip describes an optional circular clipping region. When r == 0 the clip is disabled.
type radialClip struct {
	cx, cy, r float64
}

// renderRadialGradientPixels fills pixels in rect with a radial gradient.
// fx,fy is the gradient focus in image coordinates; invScale maps distance to t ∈ [0,1].
// If clip.r > 0, pixels outside the circle are skipped.
func renderRadialGradientPixels(
	img *image.RGBA,
	rect image.Rectangle,
	fx, fy float64,
	invScale float64,
	lerp gradientLerp,
	clip radialClip,
) {
	hasClip := clip.r > 0
	r2 := clip.r * clip.r

	for py := rect.Min.Y; py < rect.Max.Y; py++ {
		dy := float64(py) + 0.5 - fy
		dy2 := dy * dy

		var cdy2 float64

		if hasClip {
			cdy := float64(py) + 0.5 - clip.cy
			cdy2 = cdy * cdy
		}

		for px := rect.Min.X; px < rect.Max.X; px++ {
			if hasClip {
				cdx := float64(px) + 0.5 - clip.cx
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
