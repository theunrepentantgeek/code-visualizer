package canvas

import (
	"image/color"
)

// drawCall records a single drawing operation dispatched to the mock backend.
type drawCall struct {
	method string
	pos    Position
	size   Size
	fill   color.RGBA
	border color.RGBA
	text   string
}

// mockBackend records all drawing calls for test assertions.
type mockBackend struct {
	calls      []drawCall
	finishPath string
	finishErr  error
}

func newMockBackend() *mockBackend {
	return &mockBackend{}
}

func (m *mockBackend) DrawRectangle(pos Position, size Size, fill, border color.RGBA, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawRectangle",
		pos:    pos,
		size:   size,
		fill:   fill,
		border: border,
	})
}

func (m *mockBackend) DrawDisc(center Position, _ float64, fill, border color.RGBA, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawDisc",
		pos:    center,
		fill:   fill,
		border: border,
	})
}

func (m *mockBackend) DrawLine(from, _ Position, _ color.RGBA, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawLine",
		pos:    from,
	})
}

func (m *mockBackend) DrawPath(_ []Position, _ color.RGBA, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawPath",
	})
}

func (m *mockBackend) DrawText(
	pos Position, text string, ink color.RGBA, _ float64, _ TextAnchor, _ float64,
) {
	m.calls = append(m.calls, drawCall{
		method: "DrawText",
		pos:    pos,
		text:   text,
		fill:   ink,
	})
}

func (m *mockBackend) DrawArcText(center Position, _ float64, text string, ink color.RGBA, _ float64) {
	m.calls = append(m.calls, drawCall{
		method: "DrawArcText",
		pos:    center,
		text:   text,
		fill:   ink,
	})
}

func (m *mockBackend) Finish(outputPath string) error {
	m.finishPath = outputPath

	return m.finishErr
}
