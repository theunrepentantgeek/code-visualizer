package treemap

// TreemapRectangle is a positioned visual element in the rendered treemap.
type TreemapRectangle struct {
	X           float64
	Y           float64
	W           float64
	H           float64
	Label       string
	ShowLabel   bool
	IsDirectory bool
	Children    []TreemapRectangle
}
