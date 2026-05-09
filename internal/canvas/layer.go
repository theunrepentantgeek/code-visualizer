package canvas

// Layer controls the z-ordering of shapes on the canvas.
// Lower values are drawn first (behind higher values).
// The 10-unit gaps between constants leave room for future intermediate layers.
type Layer int

const (
	// LayerBackground is for canvas background fills.
	LayerBackground Layer = 0
	// LayerStructure is for edges, guide tracks, and directory borders.
	LayerStructure Layer = 10
	// LayerContent is for file rectangles, file discs, and data shapes.
	LayerContent Layer = 20
	// LayerOverlay is for labels, legends, and annotations.
	LayerOverlay Layer = 30
)
