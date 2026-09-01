// Package mock provides a recording canvas Backend for use in tests.
// It implements model.Backend by appending each drawing call to a slice
// that tests can inspect.
package mock

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

// Call records a single drawing operation dispatched to a Backend.
type Call struct {
	Method      string
	Pos         geometry.Point
	To          geometry.Point
	Size        model.Size
	Points      []geometry.Point
	Loops       [][]geometry.Point
	Fill        color.RGBA
	Border      color.RGBA
	RawFill     model.Fill
	RawBorder   model.Fill
	Text        string
	FontSize    float64
	Anchor      model.TextAnchor
	Rotation    float64
	StrokeWidth float64
	BorderWidth float64
}

// Backend records all drawing calls for test assertions.
type Backend struct {
	Calls      []Call
	FinishPath string
	FinishErr  error
}

// NewBackend constructs an empty recording Backend.
func NewBackend() *Backend {
	return &Backend{}
}

func (m *Backend) DrawRectangle(
	pos geometry.Point, size model.Size, fill, border model.Fill, borderWidth float64,
) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawRectangle",
		Pos:         pos,
		Size:        size,
		Fill:        model.SolidColor(fill),
		Border:      model.SolidColor(border),
		RawFill:     fill,
		RawBorder:   border,
		BorderWidth: borderWidth,
	})
}

func (m *Backend) DrawDisc(
	center geometry.Point, _ float64, fill, border model.Fill, borderWidth float64,
) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawDisc",
		Pos:         center,
		Fill:        model.SolidColor(fill),
		Border:      model.SolidColor(border),
		RawFill:     fill,
		RawBorder:   border,
		BorderWidth: borderWidth,
	})
}

func (m *Backend) DrawPolygon(
	points []geometry.Point, fill, border model.Fill, borderWidth float64,
) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawPolygon",
		Points:      slices.Clone(points),
		Fill:        model.SolidColor(fill),
		Border:      model.SolidColor(border),
		RawFill:     fill,
		RawBorder:   border,
		BorderWidth: borderWidth,
	})
}

func (m *Backend) DrawFilledPath(loops [][]geometry.Point, fill color.RGBA) {
	cloned := make([][]geometry.Point, len(loops))
	for index, loop := range loops {
		cloned[index] = slices.Clone(loop)
	}

	m.Calls = append(m.Calls, Call{
		Method: "DrawFilledPath",
		Loops:  cloned,
		Fill:   fill,
	})
}

func (m *Backend) DrawLine(from, to geometry.Point, stroke color.RGBA, strokeWidth float64) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawLine",
		Pos:         from,
		To:          to,
		Fill:        stroke,
		StrokeWidth: strokeWidth,
	})
}

func (m *Backend) DrawPath(points []geometry.Point, stroke color.RGBA, strokeWidth float64) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawPath",
		Points:      slices.Clone(points),
		Fill:        stroke,
		StrokeWidth: strokeWidth,
	})
}

func (m *Backend) DrawText(
	pos geometry.Point, text string, ink color.RGBA, fontSize float64, anchor model.TextAnchor, rotation float64,
) {
	m.Calls = append(m.Calls, Call{
		Method:   "DrawText",
		Pos:      pos,
		Text:     text,
		Fill:     ink,
		FontSize: fontSize,
		Anchor:   anchor,
		Rotation: rotation,
	})
}

func (m *Backend) DrawArcText(center geometry.Point, _ float64, text string, ink color.RGBA, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method: "DrawArcText",
		Pos:    center,
		Text:   text,
		Fill:   ink,
	})
}

func (m *Backend) Finish(outputPath string) error {
	m.FinishPath = outputPath

	return m.FinishErr
}
