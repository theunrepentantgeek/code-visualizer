package scatter

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

func renderScatterFile(name, category string, lines, size int64) *model.File {
	f := &model.File{Name: name, Path: name}
	f.SetClassification(filesystem.FileType, category)
	f.SetQuantity(filesystem.FileLines, lines)
	f.SetQuantity(filesystem.FileSize, size)

	return f
}

func TestRenderToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Files: []*model.File{
		renderScatterFile("main.go", "go", 120, 100),
		renderScatterFile("readme.md", "md", 40, 60),
	}}
	dataset := CollectDataset(
		root,
		viz.GrainFile,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)
	layout := Layout(
		dataset,
		800,
		600,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
	)
	inks := BuildInks(dataset, stages.RequestedMetrics{}, filesystem.FileSize, palette.Temperature, "", "")
	cv := RenderToCanvas(layout, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "scatter.png")
	g.Expect(cv.Render(out)).To(Succeed())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderToCanvas_SVGIncludesAxisTitlesAndLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{Files: []*model.File{
		renderScatterFile("main.go", "go", 120, 100),
		renderScatterFile("readme.md", "md", 40, 60),
	}}
	dataset := CollectDataset(
		root,
		viz.GrainFile,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
		filesystem.FileSize,
	)
	layout := Layout(
		dataset,
		800,
		600,
		AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification},
		AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity},
	)
	inks := BuildInks(dataset, stages.RequestedMetrics{}, filesystem.FileSize, palette.Temperature, "", "")
	cv := RenderToCanvas(layout, 800, 600, inks)

	out := filepath.Join(t.TempDir(), "scatter.svg")
	g.Expect(cv.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(bytes.Contains(data, []byte("file-type"))).To(BeTrue())
	g.Expect(bytes.Contains(data, []byte("file-lines"))).To(BeTrue())
	g.Expect(bytes.Contains(data, []byte("main"))).To(BeTrue())

	dec := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := dec.Token()
		if xmlErr != nil {
			break
		}

		if se, ok := tok.(xml.StartElement); ok {
			rootElement = se.Name.Local

			break
		}
	}

	g.Expect(rootElement).To(Equal("svg"))
}

func TestMetricValueForPoint_DirectoryUsesDirectoryMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	const sizeMetric = metric.Name("file-size.sum")
	dir := &model.Directory{Name: "src"}
	dir.SetQuantity(sizeMetric, 300)
	ink := inks.NumericInk(sizeMetric, []float64{100, 300}, palette.GetPalette(palette.Temperature))

	value := metricValueForPoint(ScatterPoint{Directory: dir}, ink)

	g.Expect(value).To(Equal(inks.QuantityValue(300)))
}
