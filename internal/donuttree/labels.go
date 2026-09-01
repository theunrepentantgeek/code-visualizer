package donuttree

import (
	"math"
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
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

func labelSampleLines(metrics LabelMetrics) []string {
	lines := []string{"directory-name"}
	if metrics.Size != "" {
		lines = append(lines, string(metrics.Size))
	}

	if metrics.IncludeFill && metrics.Fill != "" {
		lines = append(lines, string(metrics.Fill))
	}

	if metrics.IncludeBorder && metrics.Border != "" {
		lines = append(lines, string(metrics.Border))
	}

	return lines
}

func buildDirectoryLabel(dir *model.Directory, metrics LabelMetrics) []string {
	if dir == nil {
		return nil
	}

	lines := []string{dir.Name}
	if component, ok := directoryMetricLabel(metrics.Size, dir); ok {
		lines = append(lines, component)
	}

	if metrics.IncludeFill {
		if component, ok := directoryMetricLabel(metrics.Fill, dir); ok {
			lines = append(lines, component)
		}
	}

	if metrics.IncludeBorder {
		if component, ok := directoryMetricLabel(metrics.Border, dir); ok {
			lines = append(lines, component)
		}
	}

	return lines
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

func addSectorLabel(cv *canvas.Canvas, node DonutNode, center geometry.Point, lines []string, ink inks.Ink) {
	fontSize := sectorLabelFontSize(node, lines)
	if fontSize == 0 {
		return
	}

	midpoint := node.StartAngle + node.SweepAngle/2
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	blockCenter := center.Translate(geometry.Vector{
		X: midRadius * math.Cos(midpoint),
		Y: midRadius * math.Sin(midpoint),
	})

	rotation := midpoint + math.Pi/2
	if isLowerHalf(midpoint) {
		rotation += math.Pi
	}

	_, lineHeight := textlayout.MeasureStrings(lines, fontSize)
	spec := &canvas.TextSpec{
		Ink:      ink,
		FontSize: fontSize,
		Anchor:   canvas.AnchorMiddle,
		Rotation: rotation,
	}

	for index, line := range lines {
		offset := (float64(index) - float64(len(lines)-1)/2) * lineHeight
		cv.AddText(canvas.LayerOverlay, canvas.Text{
			Spec: spec,
			Position: blockCenter.Translate(geometry.Vector{
				X: -offset * math.Sin(rotation),
				Y: offset * math.Cos(rotation),
			}),
			Content: line,
		})
	}
}

func isLowerHalf(angle float64) bool {
	angle = math.Mod(angle, 2*math.Pi)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	return angle > 0 && angle < math.Pi
}

func sectorLabelFontSize(node DonutNode, lines []string) float64 {
	if len(lines) == 0 || node.SweepAngle <= 0 || node.OuterRadius <= node.InnerRadius {
		return 0
	}

	widths, lineHeight := textlayout.MeasureStrings(lines, donutDefaultLabelFontSize)
	if lineHeight <= 0 {
		return 0
	}

	maxWidth := 0.0
	for _, width := range widths {
		maxWidth = max(maxWidth, width)
	}

	if maxWidth <= 0 {
		return 0
	}

	ringWidth := node.OuterRadius - node.InnerRadius
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	availableArcLength := midRadius * node.SweepAngle
	blockHeight := lineHeight * float64(len(lines))

	fontSize := min(
		donutDefaultLabelFontSize,
		donutDefaultLabelFontSize*availableArcLength/maxWidth,
		donutDefaultLabelFontSize*ringWidth/blockHeight,
	)
	if fontSize < donutMinimumLabelFontSize {
		return 0
	}

	return fontSize
}
