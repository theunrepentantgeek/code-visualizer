package donuttree

import (
	"math"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
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

func TestAddSectorLabel_RendersConfiguredDirectoryLabelGlyphs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	dir := labelDirectory()
	node := DonutNode{SweepAngle: math.Pi, InnerRadius: 100, OuterRadius: 140}
	center := canvas.Position{X: 200, Y: 200}
	cases := []struct {
		name     string
		metrics  LabelMetrics
		expected string
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
			expected: "src | file-lines.sum: 120 | file-type.mode: go | file-freshness.sum: 5",
		},
		{
			name:     "default size metric",
			metrics:  LabelMetrics{Size: "file-lines.sum"},
			expected: "src | file-lines.sum: 120",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cv := canvas.NewCanvas(400, 400)
			addSectorLabel(
				cv, node, center, buildDirectoryLabel(dir, testCase.metrics),
				inks.FixedInk(donutLabelColour),
			)

			backend := mock.NewBackend()
			g.Expect(cv.RenderTo(backend)).To(Succeed())
			glyphs := callsNamed(backend.Calls, "DrawText")
			text := make([]string, len(glyphs))
			for index, glyph := range glyphs {
				text[index] = glyph.Text
			}
			g.Expect(strings.Join(text, "")).To(Equal(testCase.expected))
		})
	}
}

func TestAddSectorLabel_CentersGlyphsOnMidpointRadius(t *testing.T) {
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
	addSectorLabel(cv, node, center, "unicode: \u65e5", inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	glyphs := callsNamed(backend.Calls, "DrawText")
	g.Expect(glyphs).NotTo(BeEmpty())

	expectedRadius := (node.InnerRadius + node.OuterRadius) / 2
	midpoint := node.StartAngle + node.SweepAngle/2
	firstAngle := math.Atan2(glyphs[0].Pos.Y-center.Y, glyphs[0].Pos.X-center.X)
	lastAngle := math.Atan2(glyphs[len(glyphs)-1].Pos.Y-center.Y, glyphs[len(glyphs)-1].Pos.X-center.X)
	g.Expect((firstAngle + lastAngle) / 2).To(BeNumerically("~", midpoint, 0.02))

	for _, glyph := range glyphs {
		radius := math.Hypot(glyph.Pos.X-center.X, glyph.Pos.Y-center.Y)
		g.Expect(radius).To(BeNumerically("~", expectedRadius, 0.001))
	}
}

func TestAddSectorLabel_InvertsLowerHalfGlyphOrderAndRotation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  math.Pi / 2,
		SweepAngle:  math.Pi / 2,
		InnerRadius: 100,
		OuterRadius: 140,
	}
	center := canvas.Position{X: 200, Y: 200}
	cv := canvas.NewCanvas(400, 400)
	addSectorLabel(cv, node, center, "abc", inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	glyphs := callsNamed(backend.Calls, "DrawText")

	g.Expect(strings.Join([]string{glyphs[0].Text, glyphs[1].Text, glyphs[2].Text}, "")).To(Equal("cba"))
	angle := math.Atan2(glyphs[0].Pos.Y-center.Y, glyphs[0].Pos.X-center.X)
	g.Expect(glyphs[0].Rotation).To(BeNumerically("~", angle+3*math.Pi/2, 0.001))
}

func TestSectorLabelFontSize_FitsMediumArcsAndRejectsTinyArcs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	medium := DonutNode{SweepAngle: math.Pi / 3, InnerRadius: 40, OuterRadius: 60}
	tiny := DonutNode{SweepAngle: 0.01, InnerRadius: 10, OuterRadius: 12}

	g.Expect(sectorLabelFontSize(medium, "medium label")).To(BeNumerically(">=", 6))
	g.Expect(sectorLabelFontSize(medium, "medium label")).To(BeNumerically("<", 14))
	g.Expect(sectorLabelFontSize(tiny, "too small")).To(BeZero())
}
