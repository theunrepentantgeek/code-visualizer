package raster

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

func TestDrawLegend_EmptyData_DoesNotPanic(t *testing.T) {
	t.Parallel()

	b := New(800, 600)
	b.DrawLegend(model.LegendData{Position: "none"}, 800, 600)
}

func TestDrawLegend_Vertical_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("vertical"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	fi, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if fi != nil {
		g.Expect(fi.Size()).To(BeNumerically(">", 0))
	}
}

func TestDrawLegend_Horizontal_ProducesImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := New(800, 600)
	b.DrawLegend(*makeSampleData("horizontal"), 800, 600)

	out := filepath.Join(t.TempDir(), "legend.png")
	err := b.Finish(out)
	g.Expect(err).NotTo(HaveOccurred())

	fi, statErr := os.Stat(out)
	g.Expect(statErr).NotTo(HaveOccurred())

	if fi != nil {
		g.Expect(fi.Size()).To(BeNumerically(">", 0))
	}
}

func makeSampleData(orientation string) *model.LegendData {
	return &model.LegendData{
		Position:    "bottom-right",
		Orientation: orientation,
		Entries: []model.LegendEntryData{
			{
				Title: "Fill: file-size",
				Kind:  model.LegendEntryNumeric,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 50, G: 50, B: 200, A: 255}, Label: "100"},
					{Colour: color.RGBA{R: 100, G: 100, B: 200, A: 255}, Label: "500"},
					{Colour: color.RGBA{R: 200, G: 200, B: 200, A: 255}, Label: ""},
				},
			},
			{
				Title: "Border: file-type",
				Kind:  model.LegendEntryCategorical,
				Swatches: []model.LegendSwatch{
					{Colour: color.RGBA{R: 0, G: 173, B: 216, A: 255}, Label: "go"},
					{Colour: color.RGBA{R: 222, G: 165, B: 132, A: 255}, Label: "rs"},
				},
			},
		},
	}
}
