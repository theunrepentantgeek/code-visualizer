// Package spiral implements data types and layout algorithms for spiral timeline visualizations.
package spiral

import (
	"image/color"
	"time"

	"github.com/bevan/code-visualizer/internal/viz"
)

// LabelMode is an alias for [viz.LabelMode].
type LabelMode = viz.LabelMode

const (
	LabelAll  = viz.LabelAll
	LabelLaps = viz.LabelLaps
	LabelNone = viz.LabelNone
)

// SpiralNode is a positioned visual element on the rendered spiral timeline.
// X and Y are absolute pixel coordinates on the canvas.
type SpiralNode struct {
	X, Y         float64     // pixel position on canvas
	DiscRadius   float64     // radius in pixels (from size metric)
	Angle        float64     // angle in radians (clockwise from 12-o'clock / north)
	SpiralRadius float64     // distance from canvas centre to this point
	TimeStart    time.Time   // start of this time bucket (inclusive)
	TimeEnd      time.Time   // end of this time bucket (exclusive)
	Label        string      // time label (e.g. "2pm", "Apr 29")
	ShowLabel    bool        // whether to render label
	FillColour   color.RGBA  // fill colour (zero value means use default)
	BorderColour *color.RGBA // border colour (nil means use default)
}
