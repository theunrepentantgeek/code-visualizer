package donuttree

import (
	"math"
	"strconv"
	"strings"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	donutDefaultLabelFontSize = 14.0
	donutMinimumLabelFontSize = 6.0
)

// LabelMetrics identifies the metric values included in directory labels.
type LabelMetrics struct {
	Size          metric.Name
	Fill          metric.Name
	Border        metric.Name
	IncludeFill   bool
	IncludeBorder bool
}

func buildDirectoryLabel(dir *model.Directory, metrics LabelMetrics) string {
	if dir == nil {
		return ""
	}

	components := []string{dir.Name}
	if component, ok := directoryMetricLabel(metrics.Size, dir); ok {
		components = append(components, string(metrics.Size)+": "+component)
	}

	if metrics.IncludeFill {
		if component, ok := directoryMetricLabel(metrics.Fill, dir); ok {
			components = append(components, string(metrics.Fill)+": "+component)
		}
	}

	if metrics.IncludeBorder {
		if component, ok := directoryMetricLabel(metrics.Border, dir); ok {
			components = append(components, string(metrics.Border)+": "+component)
		}
	}

	return strings.Join(components, " | ")
}

func directoryMetricLabel(name metric.Name, dir *model.Directory) (string, bool) {
	if name == "" || dir == nil {
		return "", false
	}

	if value, ok := dir.Quantity(name); ok {
		return strconv.FormatInt(value, 10), true
	}

	if value, ok := dir.Measure(name); ok {
		return strconv.FormatFloat(value, 'f', -1, 64), true
	}

	if value, ok := dir.Classification(name); ok {
		return value, true
	}

	return "", false
}

func addSectorLabel(cv *canvas.Canvas, node DonutNode, center canvas.Position, label string, ink inks.Ink) {
	fontSize := sectorLabelFontSize(node, label)
	if fontSize == 0 {
		return
	}

	radius := (node.InnerRadius + node.OuterRadius) / 2
	glyphs := stringsToRunes(label)
	midpoint := node.StartAngle + node.SweepAngle/2

	lowerHalf := isLowerHalf(midpoint)
	if lowerHalf {
		reverseStrings(glyphs)
	}

	widths, _ := textlayout.MeasureStrings(glyphs, fontSize)

	totalWidth := 0.0
	for _, width := range widths {
		totalWidth += width
	}

	cursor := midpoint - totalWidth/(2*radius)
	spec := &canvas.TextSpec{Ink: ink, FontSize: fontSize, Anchor: canvas.AnchorMiddle}

	for index, glyph := range glyphs {
		angle := cursor + widths[index]/(2*radius)

		rotation := angle + math.Pi/2
		if lowerHalf {
			rotation += math.Pi
		}

		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec:    &canvas.TextSpec{Ink: spec.Ink, FontSize: spec.FontSize, Anchor: spec.Anchor, Rotation: rotation},
			X:       center.X + radius*math.Cos(angle),
			Y:       center.Y + radius*math.Sin(angle),
			Content: glyph,
		})
		cursor += widths[index] / radius
	}
}

func sectorLabelFontSize(node DonutNode, label string) float64 {
	if label == "" || node.SweepAngle <= 0 {
		return 0
	}

	radius := (node.InnerRadius + node.OuterRadius) / 2
	if radius <= 0 {
		return 0
	}

	textWidth, _ := textlayout.MeasureString(label, donutDefaultLabelFontSize)
	if textWidth <= 0 {
		return 0
	}

	availableArcLength := radius * node.SweepAngle

	fontSize := min(donutDefaultLabelFontSize, donutDefaultLabelFontSize*availableArcLength/textWidth)
	if fontSize < donutMinimumLabelFontSize {
		return 0
	}

	return fontSize
}

func stringsToRunes(text string) []string {
	glyphs := make([]string, 0, len(text))
	for _, glyph := range text {
		glyphs = append(glyphs, string(glyph))
	}

	return glyphs
}

func isLowerHalf(angle float64) bool {
	angle = math.Mod(angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	return angle > 0 && angle < math.Pi
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
