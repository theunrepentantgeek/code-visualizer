package model

import (
	"image/color"
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

// LegendOrientation controls whether swatches are stacked vertically
// or laid out horizontally.
type LegendOrientation string

const (
	LegendOrientationVertical   LegendOrientation = "vertical"
	LegendOrientationHorizontal LegendOrientation = "horizontal"
)

// LegendEntryKind distinguishes numeric (continuous gradient) from
// categorical (discrete label) legend entries.
type LegendEntryKind int

const (
	// LegendEntryNumeric is for Quantity/Measure metrics with colour gradients.
	LegendEntryNumeric LegendEntryKind = iota
	// LegendEntryCategorical is for Classification metrics with labelled swatches.
	LegendEntryCategorical
)

// LegendData holds fully resolved rendering data for a legend overlay.
type LegendData struct {
	Position    LegendPosition
	Orientation LegendOrientation
	Entries     []LegendEntryData
}

// LegendEntryData describes one metric section within the legend.
type LegendEntryData struct {
	Title    string // e.g., "Fill: file-size"
	Kind     LegendEntryKind
	Swatches []LegendSwatch
}

// LegendSwatch pairs a colour with an optional label.
// For numeric entries the label is the breakpoint value at the divider
// (empty string on the last swatch). For categorical entries every
// swatch has a label.
type LegendSwatch struct {
	Colour color.RGBA
	Label  string
}

// Legend rendering constants — shared by all backends.
const (
	LegendPadding    = 12.0
	LegendMargin     = 16.0
	SwatchSize       = 28.0
	SwatchGap        = 4.0
	LabelGap         = 6.0
	EntryGap         = 14.0
	LegendFontSize   = 12.0
	TitleFontSize    = 13.0
	LegendLineHeight = 16.0
)
