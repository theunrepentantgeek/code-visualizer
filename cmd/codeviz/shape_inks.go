package main

import "github.com/theunrepentantgeek/code-visualizer/internal/canvas"

// shapeInks holds the Ink instances for any visualization render pass.
type shapeInks struct {
	fill   canvas.Ink
	border canvas.Ink
}
