package treemap

import "image/color"

// TreemapRectangle is a positioned visual element in the rendered treemap.
type TreemapRectangle struct {
	X            float64
	Y            float64
	W            float64
	H            float64
	FillColour   color.RGBA
	BorderColour *color.RGBA
	Label        string
	ShowLabel    bool
	IsDirectory  bool
	Children     []TreemapRectangle
}
