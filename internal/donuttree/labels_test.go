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

func TestAddSectorLabel_RendersLinesOnConcentricArcsWithCommonMidpoint(t *testing.T) {
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
	addSectorLabel(cv, node, center, []string{"src", "120", "go"}, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	glyphs := callsNamed(backend.Calls, "DrawText")
	g.Expect(glyphs).To(HaveLen(8))

	midpoint := node.StartAngle + node.SweepAngle/2

	for _, row := range [][]mock.Call{glyphs[:3], glyphs[3:6], glyphs[6:]} {
		firstAngle := math.Atan2(row[0].Pos.Y-center.Y, row[0].Pos.X-center.X)
		lastAngle := math.Atan2(row[len(row)-1].Pos.Y-center.Y, row[len(row)-1].Pos.X-center.X)
		g.Expect((firstAngle + lastAngle) / 2).To(BeNumerically("~", midpoint, 0.02))
	}

	radii := []float64{
		math.Hypot(glyphs[0].Pos.X-center.X, glyphs[0].Pos.Y-center.Y),
		math.Hypot(glyphs[3].Pos.X-center.X, glyphs[3].Pos.Y-center.Y),
		math.Hypot(glyphs[6].Pos.X-center.X, glyphs[6].Pos.Y-center.Y),
	}
	g.Expect(radii[0]).To(BeNumerically("<", radii[1]))
	g.Expect(radii[1]).To(BeNumerically("<", radii[2]))
}

func TestAddSectorLabel_InvertsLowerRightGlyphOrderAndRotation(t *testing.T) {
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
	addSectorLabel(cv, node, center, []string{"abc"}, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	glyphs := callsNamed(backend.Calls, "DrawText")

	g.Expect(strings.Join([]string{glyphs[0].Text, glyphs[1].Text, glyphs[2].Text}, "")).To(Equal("cba"))
	angle := math.Atan2(glyphs[0].Pos.Y-center.Y, glyphs[0].Pos.X-center.X)
	g.Expect(glyphs[0].Rotation).To(BeNumerically("~", angle+3*math.Pi/2, 0.001))
}

func TestAddSectorLabel_DoesNotInvertUpperLeftGlyphOrderOrRotation(t *testing.T) {
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
	addSectorLabel(cv, node, center, []string{"abc"}, inks.FixedInk(donutLabelColour))

	backend := mock.NewBackend()
	g.Expect(cv.RenderTo(backend)).To(Succeed())
	glyphs := callsNamed(backend.Calls, "DrawText")

	g.Expect(strings.Join([]string{glyphs[0].Text, glyphs[1].Text, glyphs[2].Text}, "")).To(Equal("abc"))
	angle := math.Atan2(glyphs[0].Pos.Y-center.Y, glyphs[0].Pos.X-center.X)
	g.Expect(math.Mod(glyphs[0].Rotation-(angle+math.Pi/2), 2*math.Pi)).To(BeNumerically("~", 0, 0.001))
}

func TestSectorLabelFontSize_FitsAllRowsAndRejectsInsufficientRadialSpace(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	medium := DonutNode{SweepAngle: math.Pi / 3, InnerRadius: 40, OuterRadius: 60}
	tooThin := DonutNode{SweepAngle: math.Pi, InnerRadius: 10, OuterRadius: 12}

	g.Expect(sectorLabelFontSize(medium, []string{"medium", "label"})).To(BeNumerically(">=", 6))
	g.Expect(sectorLabelFontSize(medium, []string{"medium", "label"})).To(BeNumerically("<", 14))
	g.Expect(sectorLabelFontSize(tooThin, []string{"one", "two"})).To(BeZero())
}
