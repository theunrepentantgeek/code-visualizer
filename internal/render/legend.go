package render

import (
	"image/color"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
)

// LegendPosition specifies where the legend is placed on the canvas.
type LegendPosition string

const (
	LegendPositionNone         LegendPosition = "none"
	LegendPositionTopLeft      LegendPosition = "top-left"
	LegendPositionTopCenter    LegendPosition = "top-center"
	LegendPositionTopRight     LegendPosition = "top-right"
	LegendPositionCenterRight  LegendPosition = "center-right"
	LegendPositionBottomRight  LegendPosition = "bottom-right"
	LegendPositionBottomCenter LegendPosition = "bottom-center"
	LegendPositionBottomLeft   LegendPosition = "bottom-left"
	LegendPositionCenterLeft   LegendPosition = "center-left"
)

// LegendOrientation controls whether swatches are stacked vertically or
// laid out horizontally.
type LegendOrientation string

const (
	LegendOrientationVertical   LegendOrientation = "vertical"
	LegendOrientationHorizontal LegendOrientation = "horizontal"
)

// DefaultOrientation returns the default orientation for a given position.
// Left/right positions default to vertical; center positions to horizontal.
func DefaultOrientation(pos LegendPosition) LegendOrientation {
	switch pos {
	case LegendPositionTopCenter, LegendPositionBottomCenter:
		return LegendOrientationHorizontal
	default:
		return LegendOrientationVertical
	}
}

// LegendEntry describes one metric shown in the legend.
type LegendEntry struct {
	Role       string // "Fill", "Border", "Size"
	MetricName string // e.g., "file-size", "file-type"
	Kind       metric.Kind

	// For Quantity/Measure metrics:
	Buckets    *metric.BucketBoundaries
	NumBuckets int
	Palette    palette.ColourPalette

	// For Classification metrics:
	Categories []CategorySwatch
}

// CategorySwatch pairs a category label with its colour.
type CategorySwatch struct {
	Label  string
	Colour color.RGBA
}

// LegendInfo holds everything needed to render a legend.
type LegendInfo struct {
	Position    LegendPosition
	Orientation LegendOrientation
	Entries     []LegendEntry
}

// maxCategorySwatches limits how many categories are shown before truncating.
const maxCategorySwatches = 10

const (
	legendPadding    = 12.0 // padding inside legend box
	legendMargin     = 16.0 // margin from canvas edge
	swatchSize       = 28.0 // square swatch dimension (~2× text height)
	swatchGap        = 4.0  // gap between adjacent swatches
	labelGap         = 6.0  // gap between swatch and label text
	entryGap         = 14.0 // gap between separate legend entries
	legendFontSize   = 12.0 // legend text size
	titleFontSize    = 13.0 // entry title text size
	legendLineHeight = 16.0 // approximate text line height
)

// legendOrigin computes the top-left (x, y) for the legend box.
func legendOrigin(
	pos LegendPosition,
	canvasW, canvasH float64,
	legendW, legendH float64,
) (ox, oy float64) {
	switch pos {
	case LegendPositionTopLeft:
		return legendMargin, legendMargin
	case LegendPositionTopCenter:
		return (canvasW - legendW) / 2, legendMargin
	case LegendPositionTopRight:
		return canvasW - legendW - legendMargin, legendMargin
	case LegendPositionCenterRight:
		return canvasW - legendW - legendMargin, (canvasH - legendH) / 2
	case LegendPositionBottomRight:
		return canvasW - legendW - legendMargin, canvasH - legendH - legendMargin
	case LegendPositionBottomCenter:
		return (canvasW - legendW) / 2, canvasH - legendH - legendMargin
	case LegendPositionCenterLeft:
		return legendMargin, (canvasH - legendH) / 2
	default:
		return legendMargin, canvasH - legendH - legendMargin
	}
}
