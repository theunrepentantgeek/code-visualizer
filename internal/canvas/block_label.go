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

// fitBlockLabel finds the largest font size that fits within (maxWidth × maxHeight).
//
// TrueType glyph metrics scale proportionally with point size, so measuring
// all lines once at a reference size lets us compute the tight-fitting size
// directly — no 14-step binary search required.
func fitBlockLabel(lines []string, maxWidth, maxHeight float64) (fittedBlockLabel, bool) {
	if maxWidth <= 0 || maxHeight <= 0 {
		return fittedBlockLabel{}, false
	}

	// Measure every line once at a comfortable reference size.
	const refSize = 12.0

	refWidths, refLineH := textlayout.MeasureStrings(lines, refSize)
	if refLineH <= 0 {
		return fittedBlockLabel{}, false
	}

	nLines := float64(len(lines))
	maxRefWidth := slices.Max(refWidths)

	// Both width and height scale linearly with font size.
	// scaleFromH: largest scale so that (refLineH * nLines * scale) ≤ maxHeight
	// scaleFromW: largest scale so that (maxRefWidth * scale) ≤ maxWidth
	scaleFromH := maxHeight / (refLineH * nLines)

	var scaleFromW float64
	if maxRefWidth > 0 {
		scaleFromW = maxWidth / maxRefWidth
	} else {
		scaleFromW = scaleFromH // all lines empty; height is the only constraint
	}

	scale := min(scaleFromW, scaleFromH)
	if scale <= 0 {
		return fittedBlockLabel{}, false
	}

	fontSize := refSize * scale
	lineHeight := refLineH * scale
	widths := make([]float64, len(refWidths))

	for i, w := range refWidths {
		widths[i] = w * scale
	}

	return fittedBlockLabel{
		fontSize:    fontSize,
		widths:      widths,
		lineHeight:  lineHeight,
		totalHeight: lineHeight * nLines,
	}, true
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
