package main

import "github.com/theunrepentantgeek/code-visualizer/internal/canvas"

// vizInks holds the Ink instances for any visualization render pass.
type vizInks struct {
	fill   canvas.Ink
	border canvas.Ink
}
