// Package mock provides a recording canvas Backend for use in tests.
// It implements model.Backend by appending each drawing call to a slice
// that tests can inspect.
package mock

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// Call records a single drawing operation dispatched to a Backend.
type Call struct {
	Method      string
	Pos         model.Position
	Size        model.Size
	Fill        color.RGBA
	Border      color.RGBA
	RawFill     model.Fill
	RawBorder   model.Fill
	Text        string
	FontSize    float64
	Anchor      model.TextAnchor
	Rotation    float64
	StrokeWidth float64
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

func (m *Backend) DrawRectangle(pos model.Position, size model.Size, fill, border model.Fill, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method:    "DrawRectangle",
		Pos:       pos,
		Size:      size,
		Fill:      model.SolidColor(fill),
		Border:    model.SolidColor(border),
		RawFill:   fill,
		RawBorder: border,
	})
}

func (m *Backend) DrawDisc(center model.Position, _ float64, fill, border model.Fill, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method:    "DrawDisc",
		Pos:       center,
		Fill:      model.SolidColor(fill),
		Border:    model.SolidColor(border),
		RawFill:   fill,
		RawBorder: border,
	})
}

func (m *Backend) DrawLine(from, _ model.Position, stroke color.RGBA, strokeWidth float64) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawLine",
		Pos:         from,
		Fill:        stroke,
		StrokeWidth: strokeWidth,
	})
}

func (m *Backend) DrawPath(_ []model.Position, _ color.RGBA, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method: "DrawPath",
	})
}

func (m *Backend) DrawText(
	pos model.Position, text string, ink color.RGBA, fontSize float64, anchor model.TextAnchor, rotation float64,
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

func (m *Backend) DrawArcText(center model.Position, _ float64, text string, ink color.RGBA, _ float64) {
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
