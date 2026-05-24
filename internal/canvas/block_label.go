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

	fontSize, widths, lineHeight, totalHeight, ok := fitBlockLabel(lines, label.W, label.H)
	if !ok {
		return
	}

	if format != FormatSVG {
		switch {
		case lineHeight <= omittedLineHeight:
			return
		case lineHeight <= greekedLineHeight:
			c.addGreekedBlockLabel(layer, label, widths, lineHeight, totalHeight)
			return
		}
	}

	c.addTextBlockLabel(layer, label, lines, fontSize, lineHeight, totalHeight)
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

func fitBlockLabel(lines []string, maxWidth, maxHeight float64) (
	fontSize float64,
	widths []float64,
	lineHeight float64,
	totalHeight float64,
	ok bool,
) {
	upper := min(maxWidth, maxHeight/float64(len(lines)))
	if upper <= 0 {
		return 0, nil, 0, 0, false
	}

	low := 0.0
	high := upper
	for range 14 {
		mid := (low + high) / 2.0
		if mid <= 0 {
			break
		}

		candidateWidths, candidateLineHeight, candidateTotalHeight := measureBlockLabel(lines, mid)
		if candidateTotalHeight <= maxHeight && slices.Max(candidateWidths) <= maxWidth {
			low = mid
			widths = candidateWidths
			lineHeight = candidateLineHeight
			totalHeight = candidateTotalHeight
		} else {
			high = mid
		}
	}

	if low <= 0 {
		return 0, nil, 0, 0, false
	}

	return low, widths, lineHeight, totalHeight, true
}

func measureBlockLabel(lines []string, fontSize float64) ([]float64, float64, float64) {
	widths := make([]float64, len(lines))
	lineHeight := 0.0
	for i, line := range lines {
		width, measuredLineHeight := textlayout.MeasureString(line, fontSize)
		widths[i] = width
		lineHeight = max(lineHeight, measuredLineHeight)
	}

	return widths, lineHeight, lineHeight * float64(len(lines))
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
