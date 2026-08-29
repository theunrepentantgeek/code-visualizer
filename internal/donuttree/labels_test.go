package donuttree

import (
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func labelDirectory() *model.Directory {
	dir := &model.Directory{Name: "src"}
	dir.SetQuantity("file-lines.sum", 120)
	dir.SetClassification("file-type.mode", "go")
	dir.SetQuantity("file-freshness.sum", 5)

	return dir
}

func TestBuildDirectoryLabel_ReturnsValueOnlyLines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cases := []struct {
		name     string
		metrics  LabelMetrics
		expected []string
	}{
		{
			name: "explicit metrics",
			metrics: LabelMetrics{
				Size:          "file-lines.sum",
				Fill:          "file-type.mode",
				Border:        "file-freshness.sum",
				IncludeFill:   true,
				IncludeBorder: true,
			},
			expected: []string{"src", "120", "go", "5"},
		},
		{
			name:     "default size metric",
			metrics:  LabelMetrics{Size: "file-lines.sum"},
			expected: []string{"src", "120"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g.Expect(buildDirectoryLabel(labelDirectory(), testCase.metrics)).To(Equal(testCase.expected))
		})
	}
}

func TestAddSectorLabel_RendersCompactLinesAlongSectorRadius(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  math.Pi,
		SweepAngle:  math.Pi / 2,
		InnerRadius: 100,
		OuterRadius: 140,
	}
	center := canvas.Position{X: 200, Y: 200}
	cv := canvas.NewCanvas(400, 400)
	addSectorLabel(cv, node, center, []string{"src", "120", "go"}, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	calls := callsNamed(backend.Calls, "DrawText")
	lines := []string{"src", "120", "go"}
	g.Expect(calls).To(HaveLen(len(lines)))

	midpoint := node.StartAngle + node.SweepAngle/2
	midRadius := (node.InnerRadius + node.OuterRadius) / 2
	blockCenter := canvas.Position{
		X: center.X + midRadius*math.Cos(midpoint),
		Y: center.Y + midRadius*math.Sin(midpoint),
	}
	fontSize := sectorLabelFontSize(node, lines)
	_, measuredLineHeight := textlayout.MeasureStrings(lines, fontSize)

	for index, call := range calls {
		offset := (float64(index) - float64(len(lines)-1)/2) * measuredLineHeight
		g.Expect(call.Text).To(Equal(lines[index]))
		g.Expect(call.Anchor).To(Equal(canvas.AnchorMiddle))
		g.Expect(call.Rotation).To(BeNumerically("~", midpoint+math.Pi/2, 0.001))
		g.Expect(call.Pos.X).To(BeNumerically("~", blockCenter.X+offset*math.Cos(midpoint), 0.001))
		g.Expect(call.Pos.Y).To(BeNumerically("~", blockCenter.Y+offset*math.Sin(midpoint), 0.001))
	}
}

func TestAddSectorLabel_FlipsTangentialBaselineOnLowerHalf(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  0,
		SweepAngle:  math.Pi / 2,
		InnerRadius: 100,
		OuterRadius: 140,
	}
	center := canvas.Position{X: 200, Y: 200}
	cv := canvas.NewCanvas(400, 400)
	addSectorLabel(cv, node, center, []string{"src", "120"}, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	calls := callsNamed(backend.Calls, "DrawText")
	g.Expect(calls).To(HaveLen(2))

	midpoint := node.StartAngle + node.SweepAngle/2
	for _, call := range calls {
		g.Expect(call.Rotation).To(BeNumerically("~", midpoint+3*math.Pi/2, 0.001))
	}
}

func TestSectorLabelFontSize_FitsRotatedBlockInBothDimensions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	wideLine := DonutNode{SweepAngle: math.Pi, InnerRadius: 100, OuterRadius: 120}
	narrowArc := DonutNode{SweepAngle: 0.1, InnerRadius: 100, OuterRadius: 140}
	roomy := DonutNode{SweepAngle: math.Pi / 2, InnerRadius: 100, OuterRadius: 140}

	g.Expect(sectorLabelFontSize(wideLine, []string{"wide"})).To(BeNumerically(">=", donutMinimumLabelFontSize))
	g.Expect(sectorLabelFontSize(wideLine, []string{"wide"})).To(BeNumerically("<", donutDefaultLabelFontSize))
	g.Expect(sectorLabelFontSize(narrowArc, []string{"a", "b", "c", "d"})).To(BeZero())
	g.Expect(sectorLabelFontSize(roomy, []string{"src", "120"})).To(Equal(donutDefaultLabelFontSize))
}
