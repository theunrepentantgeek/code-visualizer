package render

import (
	"image/color"

	"github.com/fogleman/gg"

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

// ReserveLegendSpace computes the width and height reductions needed to
// reserve space for the legend within the canvas. Returns zeros if info is
// nil, position is "none", or there are no entries.
//
// For center positions the carve-out direction is fixed (center-left/right
// reduce width; top/bottom-center reduce height). For corner positions the
// orientation decides: a vertical (tall) legend carves out side space; a
// horizontal (wide) legend carves out top/bottom space.
func ReserveLegendSpace(info *LegendInfo) (widthReduction, heightReduction float64) {
	if info == nil || info.Position == LegendPositionNone || len(info.Entries) == 0 {
		return 0, 0
	}

	dc := gg.NewContext(1, 1)
	w, h := measureLegend(dc, info)

	switch info.Position {
	case LegendPositionCenterLeft, LegendPositionCenterRight:
		return w + 2*legendMargin, 0
	case LegendPositionTopCenter, LegendPositionBottomCenter:
		return 0, h + 2*legendMargin
	default:
		// Corner positions: let the orientation decide.
		if info.Orientation == LegendOrientationVertical {
			return w + 2*legendMargin, 0
		}

		return 0, h + 2*legendMargin
	}
}

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
