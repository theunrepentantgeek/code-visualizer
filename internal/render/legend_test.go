package render

import (
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/fogleman/gg"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
	"github.com/bevan/code-visualizer/internal/radialtree"
	"github.com/bevan/code-visualizer/internal/treemap"
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

// ---- Integration tests: legend with renderers ----

func TestComputeLegendHeight_ThreeRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{MetricName: "a", Kind: metric.Quantity, Colours: testColours(3)},
			{MetricName: "b", Kind: metric.Measure, Colours: testColours(3)},
			{MetricName: "c", Kind: metric.Classification, Colours: testColours(3)},
		},
	}

	h := ComputeLegendHeight(info)
	// legendPaddingTop(8) + 3*legendRowHeight(30) + 2*legendRowGap(6) + legendPaddingBottom(6) = 116
	g.Expect(h).To(Equal(116))
}

func TestDrawLegendBand_ThreeRows(t *testing.T) {
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
				MetricName:  "freshness",
				Kind:        metric.Measure,
				Colours:     testColours(3),
				Breakpoints: []float64{0.25, 0.75},
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

	out := filepath.Join(t.TempDir(), "legend-three-rows.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_SingleColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName: "uniform",
				Kind:       metric.Quantity,
				Colours:    testColours(1),
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())

	out := filepath.Join(t.TempDir(), "legend-single-colour.png")
	g.Expect(dc.SavePNG(out)).To(Succeed())
	assertValidPNG(g, out)
}

func TestDrawLegendBand_VeryLongMetricName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	longName := strings.Repeat("a", 50)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  longName,
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{10, 20},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(800, h)
	dc.SetColor(color.White)
	dc.Clear()

	// Must not panic or error even with overflow label.
	err := DrawLegendBand(dc, info, 0, 0, 800)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestDrawLegendBand_NarrowWidth(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "narrow",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{10, 20},
			},
		},
	}

	h := ComputeLegendHeight(info)
	dc := gg.NewContext(100, h)
	dc.SetColor(color.White)
	dc.Clear()

	// Width < legendLabelWidth, swatch area ≤ 0 — should not panic.
	err := DrawLegendBand(dc, info, 0, 0, 100)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRender_Treemap_WithLegend_TallerImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			makeFile("a.go", "go", 100),
			makeFile("b.go", "go", 200),
		},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)

	// Render without legend.
	outNoLegend := filepath.Join(t.TempDir(), "treemap-no-legend.png")
	err := Render(rects, 400, 300, nil, outNoLegend)
	g.Expect(err).NotTo(HaveOccurred())

	cfgNoLegend := decodePNGConfig(g, outNoLegend)

	// Render with legend.
	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{150},
			},
		},
	}

	outWithLegend := filepath.Join(t.TempDir(), "treemap-with-legend.png")
	err = Render(rects, 400, 300, legend, outWithLegend)
	g.Expect(err).NotTo(HaveOccurred())

	cfgWithLegend := decodePNGConfig(g, outWithLegend)

	g.Expect(cfgWithLegend.Width).To(Equal(cfgNoLegend.Width))
	g.Expect(cfgWithLegend.Height).To(BeNumerically(">", cfgNoLegend.Height),
		"treemap with legend should be taller than without")
}

func TestRender_Radial_WithLegend_TallerImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100), makeFile("b.go", "go", 200)},
	}

	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)

	// Without legend.
	outNo := filepath.Join(t.TempDir(), "radial-no-legend.png")
	err := RenderRadial(&node, 400, nil, outNo)
	g.Expect(err).NotTo(HaveOccurred())

	cfgNo := decodePNGConfig(g, outNo)

	// With legend.
	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{150},
			},
		},
	}

	outWith := filepath.Join(t.TempDir(), "radial-with-legend.png")
	err = RenderRadial(&node, 400, legend, outWith)
	g.Expect(err).NotTo(HaveOccurred())

	cfgWith := decodePNGConfig(g, outWith)

	g.Expect(cfgWith.Height).To(BeNumerically(">", cfgNo.Height),
		"radial with legend should be taller")
}

func TestRender_Bubble_WithLegend_TallerImage(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()

	// Without legend.
	outNo := filepath.Join(t.TempDir(), "bubble-no-legend.png")
	err := RenderBubble(&root, 800, 600, nil, outNo)
	g.Expect(err).NotTo(HaveOccurred())

	cfgNo := decodePNGConfig(g, outNo)

	// With legend.
	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(4),
				Breakpoints: []float64{100, 300, 500},
			},
		},
	}

	outWith := filepath.Join(t.TempDir(), "bubble-with-legend.png")
	err = RenderBubble(&root, 800, 600, legend, outWith)
	g.Expect(err).NotTo(HaveOccurred())

	cfgWith := decodePNGConfig(g, outWith)

	g.Expect(cfgWith.Height).To(BeNumerically(">", cfgNo.Height),
		"bubble with legend should be taller")
}

func TestRender_Treemap_NilLegend_OriginalHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "treemap-nil-legend.png")
	err := Render(rects, 400, 300, nil, out)
	g.Expect(err).NotTo(HaveOccurred())

	cfg := decodePNGConfig(g, out)
	g.Expect(cfg.Height).To(Equal(300))
}

func TestRender_Treemap_SVG_WithLegend_ContainsGroup(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)

	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{50},
			},
		},
	}

	out := filepath.Join(t.TempDir(), "treemap-legend.svg")
	err := Render(rects, 400, 300, legend, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<g "))
	g.Expect(svg).To(ContainSubstring("file-size"))
	g.Expect(svg).To(ContainSubstring("<rect "))
}

func TestRender_Treemap_SVG_NoLegend_NoGroup(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	rects := treemap.Layout(root, 400, 300, filesystem.FileSize)
	out := filepath.Join(t.TempDir(), "treemap-no-legend.svg")
	err := Render(rects, 400, 300, nil, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	// The legend group wrapper uses translate; without a legend, there should
	// be no translate group.
	g.Expect(svg).NotTo(ContainSubstring("translate"))
}

func TestRender_Radial_SVG_WithLegend_ContainsGroup(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeFile("a.go", "go", 100)},
	}

	node := radialtree.Layout(root, 400, filesystem.FileSize, radialtree.LabelAll)

	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{50},
			},
		},
	}

	out := filepath.Join(t.TempDir(), "radial-legend.svg")
	err := RenderRadial(&node, 400, legend, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<g "))
	g.Expect(svg).To(ContainSubstring("file-size"))
}

func TestRender_Bubble_SVG_WithLegend_ContainsGroup(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()

	legend := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName: "file-type",
				Kind:       metric.Classification,
				Colours:    testColours(3),
				Categories: []string{".go", ".rs", ".py"},
			},
		},
	}

	out := filepath.Join(t.TempDir(), "bubble-legend.svg")
	err := RenderBubble(&root, 800, 600, legend, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	g.Expect(svg).To(ContainSubstring("<g "))
	g.Expect(svg).To(ContainSubstring("file-type"))
	g.Expect(svg).To(ContainSubstring(".go"))
}

func TestRender_Bubble_SVG_NoLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := sampleBubbleTree()

	out := filepath.Join(t.TempDir(), "bubble-no-legend.svg")
	err := RenderBubble(&root, 800, 600, nil, out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	svg := string(data)
	// No legend group transform.
	g.Expect(svg).NotTo(MatchRegexp(`<g transform="translate\([^"]*\)">`))
}

func TestBuildNumericLegendRow_ProducesCorrectBucketCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Temperature)
	buckets := metric.ComputeBuckets([]float64{1, 10, 100, 1000}, 5)
	numBuckets := len(buckets.Boundaries) + 1

	row := BuildNumericLegendRow("size", metric.Quantity, buckets, numBuckets, pal)

	g.Expect(row.Colours).To(HaveLen(numBuckets))
	g.Expect(row.Breakpoints).To(Equal(buckets.Boundaries))
}

func TestBuildCategoricalLegendRow_ProducesCorrectCategoryCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pal := palette.GetPalette(palette.Categorization)
	cats := []string{"go", "rs", "py"}

	row := BuildCategoricalLegendRow("type", cats, pal)

	g.Expect(row.Colours).To(HaveLen(3))
	g.Expect(row.Categories).To(Equal(cats))
	g.Expect(row.Kind).To(Equal(metric.Classification))
}

// decodePNGConfig opens a PNG file and returns its image.Config.
func decodePNGConfig(g Gomega, path string) image.Config {
	f, err := os.Open(path)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	cfg, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))

	return cfg
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
