package model

import (
	"image/color"
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
// Position and Orientation use the same string values as canvas.LegendPosition
// and canvas.LegendOrientation (e.g., "bottom-right", "vertical").
type LegendData struct {
	Position    string
	Orientation string
	Entries     []LegendEntryData
}

// LegendEntryData describes one metric section within the legend.
type LegendEntryData struct {
	Title    string // e.g., "Fill: file-size"
	Kind     LegendEntryKind
	Swatches []LegendSwatch
	IsBorder bool // true when swatches represent border colours (render as outlines)
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
