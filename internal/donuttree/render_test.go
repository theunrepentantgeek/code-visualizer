package donuttree

import (
	"bytes"
	"encoding/xml"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func donutDirectory(name string, lines int64) *model.Directory {
	dir := &model.Directory{Name: name}
	dir.SetQuantity(filesystem.FileLines, lines)

	return dir
}

func donutRoot() *model.Directory {
	src := donutDirectory("src", 120)
	src.Dirs = []*model.Directory{donutDirectory("api", 80)}
	src.Files = []*model.File{{Name: "main.go"}}

	root := donutDirectory("project", 120)
	root.Dirs = []*model.Directory{src}
	root.Files = []*model.File{{Name: "README.md"}}

	return root
}

func renderCalls(t *testing.T, cv *canvas.Canvas) []mock.Call {
	t.Helper()

	backend := mock.NewBackend()
	g := NewGomegaWithT(t)
	g.Expect(cv.RenderTo(backend)).To(Succeed())

	return backend.Calls
}

func callsNamed(calls []mock.Call, method string) []mock.Call {
	result := make([]mock.Call, 0)

	for _, call := range calls {
		if call.Method == method {
			result = append(result, call)
		}
	}

	return result
}

func donutOutputDir(t *testing.T) string {
	t.Helper()
	path := filepath.Join(".", "."+strings.ReplaceAll(t.Name(), "/", "-"))
	g := NewGomegaWithT(t)
	g.Expect(os.RemoveAll(path)).To(Succeed())
	g.Expect(os.Mkdir(path, 0o755)).To(Succeed())
	t.Cleanup(func() { _ = os.RemoveAll(path) })

	return path
}

func TestRenderToCanvas_RendersOneSectorPerDirectoryAndOneRootAnchor(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 600, filesystem.FileLines)
	is := BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", "")

	calls := renderCalls(t, RenderToCanvas(layout, root, 600, 600, is))

	g.Expect(callsNamed(calls, "DrawPolygon")).To(HaveLen(2))
	g.Expect(callsNamed(calls, "DrawDisc")).To(HaveLen(1))
	g.Expect(callsNamed(calls, "DrawText")).To(ContainElement(HaveField("Text", "project")))
}

func TestRenderToCanvas_UsesNoBorderUnlessConfigured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 600, filesystem.FileLines)

	withoutBorder := renderCalls(t, RenderToCanvas(
		layout, root, 600, 600,
		BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", ""),
	))
	for _, call := range callsNamed(withoutBorder, "DrawPolygon") {
		g.Expect(call.BorderWidth).To(BeZero())
	}

	root.SetQuantity(metric.Name("file-freshness.sum"), 1)
	root.Dirs[0].SetQuantity(metric.Name("file-freshness.sum"), 1)
	root.Dirs[0].Dirs[0].SetQuantity(metric.Name("file-freshness.sum"), 1)

	withBorder := renderCalls(t, RenderToCanvas(
		layout, root, 600, 600,
		BuildInks(
			root, stages.CollectRequestedMetrics(metric.Name("file-freshness.sum")),
			filesystem.FileLines, palette.Neutral,
			"file-freshness.sum", palette.GoodBad,
		),
	))
	for _, call := range callsNamed(withBorder, "DrawPolygon") {
		g.Expect(call.BorderWidth).To(Equal(1.0))
	}
}

func TestRenderToCanvas_WritesRecognizablePNGAndSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 360, filesystem.FileLines)
	is := BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", "")
	cv := RenderToCanvas(layout, root, 360, 360, is)
	outputDir := donutOutputDir(t)

	pngPath := filepath.Join(outputDir, "donut.png")
	g.Expect(cv.Render(pngPath)).To(Succeed())
	png, err := os.Open(pngPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer png.Close()

	_, format, err := image.DecodeConfig(png)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))

	svgPath := filepath.Join(outputDir, "donut.svg")
	g.Expect(cv.Render(svgPath)).To(Succeed())
	data, err := os.ReadFile(svgPath)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	g.Expect(err).NotTo(HaveOccurred())

	start, ok := token.(xml.StartElement)
	g.Expect(ok).To(BeTrue())
	g.Expect(start.Name.Local).To(Equal("svg"))
}

func TestRenderStage_KeepsConfiguredDimensionsAfterTitleAndFooterReservation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	cfg := config.New()
	width, height := 800, 480
	cfg.ImageSize.Width = &width
	cfg.ImageSize.Height = &height
	size := "file-lines"
	cfg.DonutTree.Size = &size
	cfg.Title = &config.Title{Text: new("Donut")}
	common := &stages.CommonState{
		Root:       root,
		RootConfig: cfg,
		Width:      width,
		Height:     height,
	}
	g.Expect(stages.InitDrawingBounds(common)).To(Succeed())
	g.Expect(stages.ReserveTitleBounds(common)).To(Succeed())
	g.Expect(stages.ReserveFooterBounds(common)).To(Succeed())

	state := &State{
		SizeMetric:  filesystem.FileLines,
		FillMetric:  filesystem.FileLines,
		FillPalette: palette.Neutral,
	}

	g.Expect(BuildInksStage(common, state)).To(Succeed())
	g.Expect(LayoutStage(common, state)).To(Succeed())
	g.Expect(RenderStage(common, state)).To(Succeed())

	outputDir := donutOutputDir(t)
	output := filepath.Join(outputDir, "stage.png")
	g.Expect(common.Canvas.Render(output)).To(Succeed())
	file, err := os.Open(output)
	g.Expect(err).NotTo(HaveOccurred())

	defer file.Close()

	info, format, err := image.DecodeConfig(file)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
	g.Expect(info.Width).To(Equal(width))
	g.Expect(info.Height).To(Equal(height))
}
