// Package spiral implements data types and layout algorithms for spiral timeline visualizations.
package spiral

import (
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// SpiralNode is a positioned visual element on the rendered spiral timeline.
// Position is an absolute pixel location on the canvas.
type SpiralNode struct {
	Position     geometry.Point // pixel position on canvas
	DiscRadius   float64        // radius in pixels (from size metric)
	Angle        float64        // angle in radians (clockwise from 12-o'clock / north)
	SpiralRadius float64        // distance from canvas centre to this point
	TimeStart    time.Time      // start of this time bucket (inclusive)
	TimeEnd      time.Time      // end of this time bucket (exclusive)
}
