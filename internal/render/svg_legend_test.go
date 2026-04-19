package render

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
)

func TestWriteSVGLegend_Nil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "nil-legend.svg")
	f, err := os.Create(out)
	g.Expect(err).NotTo(HaveOccurred())

	writeSVGLegend(f, nil, 0, 800)
	g.Expect(f.Close()).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	// Nil info writes nothing.
	g.Expect(string(data)).To(BeEmpty())
}

func TestWriteSVGLegend_EmptyRows(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "empty-legend.svg")
	f, err := os.Create(out)
	g.Expect(err).NotTo(HaveOccurred())

	info := &LegendInfo{}
	writeSVGLegend(f, info, 0, 800)
	g.Expect(f.Close()).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(BeEmpty())
}

func TestWriteSVGLegend_NumericRow_ContainsRectAndText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "file-size",
				Kind:        metric.Quantity,
				Colours:     testColours(3),
				Breakpoints: []float64{100, 500},
			},
		},
	}

	svg := renderSVGLegendToString(g, info, 800)

	// Must contain a wrapping <g> group.
	g.Expect(svg).To(ContainSubstring("<g "))
	g.Expect(svg).To(ContainSubstring("</g>"))

	// Must contain swatch rectangles (one per colour).
	g.Expect(strings.Count(svg, "<rect ")).To(Equal(3))

	// Must contain the metric name and breakpoint text.
	g.Expect(svg).To(ContainSubstring("file-size"))
	g.Expect(svg).To(ContainSubstring("100"))
	g.Expect(svg).To(ContainSubstring("500"))
}

func TestWriteSVGLegend_CategoricalRow_ContainsCategoryLabels(t *testing.T) {
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

	svg := renderSVGLegendToString(g, info, 800)

	g.Expect(svg).To(ContainSubstring("file-type"))
	g.Expect(svg).To(ContainSubstring(".go"))
	g.Expect(svg).To(ContainSubstring(".rs"))
	g.Expect(svg).To(ContainSubstring(".py"))
}

func TestWriteSVGLegend_MultipleRows_ContainsAllMetrics(t *testing.T) {
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
				Colours:    testColours(2),
				Categories: []string{".go", ".rs"},
			},
		},
	}

	svg := renderSVGLegendToString(g, info, 800)

	// All three metric names appear.
	g.Expect(svg).To(ContainSubstring("file-size"))
	g.Expect(svg).To(ContainSubstring("freshness"))
	g.Expect(svg).To(ContainSubstring("file-type"))

	// 4+3+2 = 9 swatch rectangles.
	g.Expect(strings.Count(svg, "<rect ")).To(Equal(9))
}

func TestWriteSVGLegend_HTMLEscapesMetricName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName:  "size<&>",
				Kind:        metric.Quantity,
				Colours:     testColours(2),
				Breakpoints: []float64{50},
			},
		},
	}

	svg := renderSVGLegendToString(g, info, 800)

	// Angle brackets and ampersand must be escaped.
	g.Expect(svg).To(ContainSubstring("size&lt;&amp;&gt;"))
	g.Expect(svg).NotTo(ContainSubstring("size<&>"))
}

func TestWriteSVGLegend_SingleColourRow(t *testing.T) {
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

	svg := renderSVGLegendToString(g, info, 800)

	g.Expect(strings.Count(svg, "<rect ")).To(Equal(1))
	g.Expect(svg).To(ContainSubstring("uniform"))
}

func TestWriteSVGLegend_EmptyColoursSkipsRow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{MetricName: "empty", Kind: metric.Quantity, Colours: nil},
		},
	}

	svg := renderSVGLegendToString(g, info, 800)

	// Group wrapper is written, but no rects or metric text because
	// writeSVGLegendRow exits early for empty colours.
	g.Expect(strings.Count(svg, "<rect ")).To(Equal(0))
}

func TestSVGFormatBreakpoint_Quantity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(svgFormatBreakpoint(42, metric.Quantity)).To(Equal("42"))
	g.Expect(svgFormatBreakpoint(0, metric.Quantity)).To(Equal("0"))
	g.Expect(svgFormatBreakpoint(99999, metric.Quantity)).To(Equal("99999"))
}

func TestSVGFormatBreakpoint_Measure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(svgFormatBreakpoint(1.0, metric.Measure)).To(Equal("1"))
	g.Expect(svgFormatBreakpoint(0.25, metric.Measure)).To(Equal("0.25"))
	g.Expect(svgFormatBreakpoint(3.14159, metric.Measure)).To(Equal("3.1"))
}

func TestWriteSVGLegend_ManyCategories(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := make([]string, 20)
	for i := range categories {
		categories[i] = "cat" + string(rune('A'+i%26))
	}

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName: "many",
				Kind:       metric.Classification,
				Colours:    testColours(20),
				Categories: categories,
			},
		},
	}

	svg := renderSVGLegendToString(g, info, 1200)

	g.Expect(strings.Count(svg, "<rect ")).To(Equal(20))
	g.Expect(svg).To(ContainSubstring("catA"))
	g.Expect(svg).To(ContainSubstring("catT"))
}

func TestWriteSVGLegend_NarrowWidth_NoSwatches(t *testing.T) {
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

	// Width smaller than legendLabelWidth (120) — swatch area is ≤ 0.
	svg := renderSVGLegendToString(g, info, 100)

	// Metric label is written but no rects for swatches.
	g.Expect(svg).To(ContainSubstring("narrow"))
	g.Expect(strings.Count(svg, "<rect ")).To(Equal(0))
}

func TestWriteSVGLegend_SwatchStrokeColour(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	info := &LegendInfo{
		Rows: []LegendRow{
			{
				MetricName: "stroke-check",
				Kind:       metric.Quantity,
				Colours: []color.RGBA{
					{R: 0xFF, G: 0, B: 0, A: 0xFF},
					{R: 0, G: 0xFF, B: 0, A: 0xFF},
				},
				Breakpoints: []float64{50},
			},
		},
	}

	svg := renderSVGLegendToString(g, info, 800)

	// Swatches have a grey border.
	g.Expect(svg).To(ContainSubstring(`stroke="#808080"`))
}

// renderSVGLegendToString is a test helper that calls writeSVGLegend and returns
// the generated SVG fragment as a string.
func renderSVGLegendToString(g Gomega, info *LegendInfo, width float64) string {
	dir, err := os.MkdirTemp(".", "svg-legend-test-*")
	g.Expect(err).NotTo(HaveOccurred())

	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "legend.svg")
	f, err := os.Create(path)
	g.Expect(err).NotTo(HaveOccurred())

	writeSVGLegend(f, info, 0, width)
	g.Expect(f.Close()).To(Succeed())

	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	return string(data)
}
