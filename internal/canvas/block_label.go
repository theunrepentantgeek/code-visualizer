package canvas

import (
	"image/color"
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

const (
	greekedLineHeight = 5.0
	omittedLineHeight = 2.0
)

// BlockLabel is a centered multi-line label constrained to a rectangular area.
type BlockLabel struct {
	Bounds       geometry.Rect
	Lines        []string
	Ink          color.RGBA
	PreserveText bool
}

// AddBlockLabel adds a centered multi-line label sized to fit the given bounds.
func (c *Canvas) AddBlockLabel(layer Layer, label BlockLabel, format ImageFormat) {
	lines := compactLabelLines(label.Lines)
	width, height := label.Bounds.Width(), label.Bounds.Height()

	if len(lines) == 0 || width <= 0 || height <= 0 {
		return
	}

	layout, ok := fitBlockLabel(lines, width, height)
	if !ok {
		return
	}

	if format != FormatSVG && !label.PreserveText {
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
		Ink:      inks.FixedInk(label.Ink),
		FontSize: fontSize,
		Anchor:   AnchorMiddle,
	}
	centerX := label.Bounds.Min.X + label.Bounds.Width()/2.0
	top := label.Bounds.Min.Y + (label.Bounds.Height()-totalHeight)/2.0

	for i, line := range lines {
		c.AddText(layer, Text{
			Spec: spec,
			Position: geometry.NewPoint(
				centerX,
				top+lineHeight*(float64(i)+0.5),
			),
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
		Stroke:      inks.FixedInk(label.Ink),
		StrokeWidth: max(1.0, lineHeight/2.0),
	}
	centerX := label.Bounds.Min.X + label.Bounds.Width()/2.0
	top := label.Bounds.Min.Y + (label.Bounds.Height()-totalHeight)/2.0

	for i, width := range widths {
		lineWidth := min(label.Bounds.Width(), max(width, lineHeight*4.0))
		y := top + lineHeight*(float64(i)+0.5)
		c.AddLine(layer, Line{
			Spec: spec,
			From: geometry.NewPoint(centerX-lineWidth/2.0, y),
			To:   geometry.NewPoint(centerX+lineWidth/2.0, y),
		})
	}
}
