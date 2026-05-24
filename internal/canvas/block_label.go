package canvas

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
)

const (
	greekedLineHeight = 5.0
	omittedLineHeight = 2.0
)

// BlockLabel is a centered multi-line label constrained to a rectangular area.
type BlockLabel struct {
	X, Y, W, H float64
	Lines      []string
	Ink        color.RGBA
}

// AddBlockLabel adds a centered multi-line label sized to fit the given bounds.
func (c *Canvas) AddBlockLabel(layer Layer, label BlockLabel, format ImageFormat) {
	lines := compactLabelLines(label.Lines)
	if len(lines) == 0 || label.W <= 0 || label.H <= 0 {
		return
	}

	layout, ok := fitBlockLabel(lines, label.W, label.H)
	if !ok {
		return
	}

	if format != FormatSVG {
		switch {
		case layout.lineHeight <= omittedLineHeight:
			return
		case layout.lineHeight <= greekedLineHeight:
			c.addGreekedBlockLabel(layer, label, layout.widths, layout.lineHeight, layout.totalHeight)

			return
		}
	}

	c.addTextBlockLabel(layer, label, lines, layout.fontSize, layout.lineHeight, layout.totalHeight)
}

func compactLabelLines(lines []string) []string {
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			compact = append(compact, line)
		}
	}

	return compact
}

type fittedBlockLabel struct {
	fontSize    float64
	widths      []float64
	lineHeight  float64
	totalHeight float64
}

func fitBlockLabel(lines []string, maxWidth, maxHeight float64) (fittedBlockLabel, bool) {
	upper := min(maxWidth, maxHeight/float64(len(lines)))
	if upper <= 0 {
		return fittedBlockLabel{}, false
	}

	low := 0.0
	high := upper
	best := fittedBlockLabel{}

	for range 14 {
		mid := (low + high) / 2.0
		if mid <= 0 {
			break
		}

		candidate := measureBlockLabel(lines, mid)
		if candidate.totalHeight <= maxHeight && slices.Max(candidate.widths) <= maxWidth {
			low = mid
			candidate.fontSize = mid
			best = candidate
		} else {
			high = mid
		}
	}

	if low <= 0 {
		return fittedBlockLabel{}, false
	}

	return best, true
}

func measureBlockLabel(lines []string, fontSize float64) fittedBlockLabel {
	widths := make([]float64, len(lines))
	lineHeight := 0.0

	for i, line := range lines {
		width, measuredLineHeight := textlayout.MeasureString(line, fontSize)
		widths[i] = width
		lineHeight = max(lineHeight, measuredLineHeight)
	}

	return fittedBlockLabel{
		widths:      widths,
		lineHeight:  lineHeight,
		totalHeight: lineHeight * float64(len(lines)),
	}
}

func (c *Canvas) addTextBlockLabel(
	layer Layer,
	label BlockLabel,
	lines []string,
	fontSize, lineHeight, totalHeight float64,
) {
	spec := &TextSpec{
		Ink:      FixedInk(label.Ink),
		FontSize: fontSize,
		Anchor:   AnchorMiddle,
	}
	centerX := label.X + label.W/2.0
	top := label.Y + (label.H-totalHeight)/2.0

	for i, line := range lines {
		c.AddText(layer, Text{
			Spec:    spec,
			X:       centerX,
			Y:       top + lineHeight*(float64(i)+0.5),
			Content: line,
		})
	}
}

func (c *Canvas) addGreekedBlockLabel(
	layer Layer,
	label BlockLabel,
	widths []float64,
	lineHeight, totalHeight float64,
) {
	spec := &LineSpec{
		Stroke:      FixedInk(label.Ink),
		StrokeWidth: max(1.0, lineHeight/2.0),
	}
	centerX := label.X + label.W/2.0
	top := label.Y + (label.H-totalHeight)/2.0

	for i, width := range widths {
		lineWidth := min(label.W, max(width, lineHeight*4.0))
		y := top + lineHeight*(float64(i)+0.5)
		c.AddLine(layer, Line{
			Spec: spec,
			X1:   centerX - lineWidth/2.0,
			Y1:   y,
			X2:   centerX + lineWidth/2.0,
			Y2:   y,
		})
	}
}
