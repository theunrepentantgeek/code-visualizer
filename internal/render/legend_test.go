package render

import (
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/metric"
)

func TestComputeLegendHeight_Nil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(ComputeLegendHeight(nil)).To(Equal(0))
}

func TestComputeLegendHeight_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{}
	g.Expect(ComputeLegendHeight(info)).To(Equal(0))
}

func TestComputeLegendHeight_OneRow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{MetricName: "file-size", Kind: metric.Quantity, Colours: testColours(3)},
		},
	}

	h := ComputeLegendHeight(info)
	// legendPaddingTop(8) + 1*legendRowHeight(30) + 0*legendRowGap + legendPaddingBottom(6) = 44
	g.Expect(h).To(Equal(44))
}

func TestComputeLegendHeight_TwoRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{MetricName: "fill", Kind: metric.Quantity, Colours: testColours(3)},
			{MetricName: "border", Kind: metric.Classification, Colours: testColours(4)},
		},
	}

	h := ComputeLegendHeight(info)
	// legendPaddingTop(8) + 2*legendRowHeight(30) + 1*legendRowGap(6) + legendPaddingBottom(6) = 80
	g.Expect(h).To(Equal(80))
}

func TestDrawLegendBand_Nil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dc := gg.NewContext(800, 100)
	err := DrawLegendBand(dc, nil, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestDrawLegendBand_NumericQuantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(4),
				Breakpoints: []float64{100, 500, 1000},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify we can save the output as a valid PNG.
	out := filepath.Join(t.TempDir(), "legend-quantity.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_NumericMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "freshness",
				Kind:        metric.Measure,
				Colours:     testColours(5),
				Breakpoints: []float64{0.25, 0.5, 0.75, 1.0},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())

	out := filepath.Join(t.TempDir(), "legend-measure.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_Categorical(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName: "file-type",
				Kind:       metric.Classification,
				Colours:    testColours(3),
				Categories: []string{".go", ".rs", ".py"},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())

	out := filepath.Join(t.TempDir(), "legend-categorical.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_TwoRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(4),
				Breakpoints: []float64{100, 500, 1000},
			},
			{
				MetricName: "file-type",
				Kind:       metric.Classification,
				Colours:    testColours(3),
				Categories: []string{".go", ".rs", ".py"},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())

	out := filepath.Join(t.TempDir(), "legend-two-rows.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_EmptyColours(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{MetricName: "empty", Kind: metric.Quantity, Colours: nil},
		},
	}

	dc := gg.NewContext(800, 50)
	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFormatBreakpoint_Quantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(formatBreakpoint(100, metric.Quantity)).To(Equal("100"))
	g.Expect(formatBreakpoint(1500, metric.Quantity)).To(Equal("1500"))
	g.Expect(formatBreakpoint(0, metric.Quantity)).To(Equal("0"))
}

func TestFormatBreakpoint_Measure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(formatBreakpoint(1.0, metric.Measure)).To(Equal("1"))
	g.Expect(formatBreakpoint(0.25, metric.Measure)).To(Equal("0.25"))
	g.Expect(formatBreakpoint(3.14159, metric.Measure)).To(Equal("3.1"))
}

// testColours returns n distinct colours for testing.
func testColours(n int) []color.RGBA {
	base := []color.RGBA{
		{R: 0x33, G: 0x66, B: 0xCC, A: 0xFF},
		{R: 0x66, G: 0xCC, B: 0x66, A: 0xFF},
		{R: 0xCC, G: 0x66, B: 0x33, A: 0xFF},
		{R: 0xCC, G: 0xCC, B: 0x33, A: 0xFF},
		{R: 0x99, G: 0x33, B: 0xCC, A: 0xFF},
		{R: 0x33, G: 0xCC, B: 0xCC, A: 0xFF},
	}

	result := make([]color.RGBA, n)
	for i := range n {
		result[i] = base[i%len(base)]
	}

	return result
}

// assertValidPNG opens the file and checks it decodes as PNG.
func assertValidPNG(g Gomega, path string) {
	f, err := os.Open(path)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}
