// Package testsupport provides canvas-related test doubles for use by
// other packages' tests. It is intentionally separated from the canvas
// package so that production code cannot import the mock by mistake.
package testsupport

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// Call records a single drawing operation dispatched to a MockBackend.
type Call struct {
	Method      string
	Pos         canvas.Position
	Size        canvas.Size
	Fill        color.RGBA
	Border      color.RGBA
	RawFill     model.Fill
	RawBorder   model.Fill
	Text        string
	FontSize    float64
	Anchor      canvas.TextAnchor
	Rotation    float64
	StrokeWidth float64
}

// MockBackend records all drawing calls for test assertions.
type MockBackend struct {
	Calls      []Call
	FinishPath string
	FinishErr  error
}

// NewMockBackend constructs an empty MockBackend.
func NewMockBackend() *MockBackend {
	return &MockBackend{}
}

func (m *MockBackend) DrawRectangle(pos canvas.Position, size canvas.Size, fill, border model.Fill, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method:    "DrawRectangle",
		Pos:       pos,
		Size:      size,
		Fill:      solidColor(fill),
		Border:    solidColor(border),
		RawFill:   fill,
		RawBorder: border,
	})
}

func (m *MockBackend) DrawDisc(center canvas.Position, _ float64, fill, border model.Fill, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method:    "DrawDisc",
		Pos:       center,
		Fill:      solidColor(fill),
		Border:    solidColor(border),
		RawFill:   fill,
		RawBorder: border,
	})
}

func (m *MockBackend) DrawLine(from, _ canvas.Position, stroke color.RGBA, strokeWidth float64) {
	m.Calls = append(m.Calls, Call{
		Method:      "DrawLine",
		Pos:         from,
		Fill:        stroke,
		StrokeWidth: strokeWidth,
	})
}

func (m *MockBackend) DrawPath(_ []canvas.Position, _ color.RGBA, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method: "DrawPath",
	})
}

func (m *MockBackend) DrawText(
	pos canvas.Position, text string, ink color.RGBA, fontSize float64, anchor canvas.TextAnchor, rotation float64,
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

func (m *MockBackend) DrawArcText(center canvas.Position, _ float64, text string, ink color.RGBA, _ float64) {
	m.Calls = append(m.Calls, Call{
		Method: "DrawArcText",
		Pos:    center,
		Text:   text,
		Fill:   ink,
	})
}

func (m *MockBackend) Finish(outputPath string) error {
	m.FinishPath = outputPath

	return m.FinishErr
}

func solidColor(f model.Fill) color.RGBA {
	switch v := f.(type) {
	case model.SolidFill:
		return v.Color
	case model.RadialGradientFill:
		return v.Center
	default:
		return color.RGBA{A: 255}
	}
}
