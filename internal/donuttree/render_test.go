package donuttree

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
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

	calls := renderCalls(t, RenderToCanvas(layout, root, 600, 600, is, LabelMetrics{Size: filesystem.FileLines}))

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
		LabelMetrics{Size: filesystem.FileLines},
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
		LabelMetrics{Size: filesystem.FileLines},
	))
	for _, call := range callsNamed(withBorder, "DrawPolygon") {
		g.Expect(call.BorderWidth).To(Equal(1.0))
	}
}

func TestRenderToCanvas_UsesContrastSafeSectorLabelInks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	const fillMetric = metric.Name("label-contrast")
	dark := donutDirectory("dark", 100)
	dark.SetQuantity(fillMetric, 0)
	light := donutDirectory("light", 100)
	light.SetQuantity(fillMetric, 100)
	root := donutDirectory("root", 200)
	root.Dirs = []*model.Directory{dark, light}
	fill := inks.NumericInk(fillMetric, []float64{0, 100}, palette.GetPalette(palette.Neutral))
	is := Inks{ShapeInks: inks.ShapeInks{
		Fill:   fill,
		Border: inks.FixedInk(donutFallbackBorder),
	}}

	calls := renderCalls(t, RenderToCanvas(
		Layout(root, 600, filesystem.FileLines), root, 600, 600, is, LabelMetrics{},
	))
	glyphColours := make(map[string]color.RGBA)
	for _, call := range callsNamed(calls, "DrawText") {
		if call.Text == "d" || call.Text == "l" {
			glyphColours[call.Text] = call.Fill
		}
	}

	darkExpected := canvas.TextColourFor(is.Fill.Dip(inks.MetricValueForDirectory(dark, is.Fill)))
	lightExpected := canvas.TextColourFor(is.Fill.Dip(inks.MetricValueForDirectory(light, is.Fill)))
	g.Expect(glyphColours).To(HaveKeyWithValue("d", darkExpected))
	g.Expect(glyphColours).To(HaveKeyWithValue("l", lightExpected))
	g.Expect(darkExpected).NotTo(Equal(lightExpected))
}

func TestSectorPoints_FollowsAnnularBoundarySampling(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  0.2,
		SweepAngle:  math.Pi / 3,
		InnerRadius: 40,
		OuterRadius: 80,
	}
	center := canvas.Position{X: 120, Y: 160}

	steps := max(2, int(math.Ceil(node.SweepAngle/(2*math.Pi)*64)))
	boundarySamples := steps + 1
	points := sectorPoints(node, center)

	g.Expect(points).To(HaveLen(2*boundarySamples + 1))
	g.Expect(points[:len(points)-1]).To(HaveLen(2 * boundarySamples))
	g.Expect(points[len(points)-1].X).To(BeNumerically("~", points[0].X, 0.000001))
	g.Expect(points[len(points)-1].Y).To(BeNumerically("~", points[0].Y, 0.000001))

	minimumSweep := node
	minimumSweep.SweepAngle = math.Pi / 128
	minimumSteps := max(2, int(math.Ceil(minimumSweep.SweepAngle/(2*math.Pi)*64)))
	g.Expect(sectorPoints(minimumSweep, center)).To(HaveLen(2*(minimumSteps+1) + 1))

	for step := range boundarySamples {
		outerAngle := node.StartAngle + node.SweepAngle*float64(step)/float64(steps)
		outerExpected := polarPosition(center, node.OuterRadius, outerAngle)
		g.Expect(points[step].X).To(BeNumerically("~", outerExpected.X, 0.000001))
		g.Expect(points[step].Y).To(BeNumerically("~", outerExpected.Y, 0.000001))

		innerAngle := node.EndAngle() - node.SweepAngle*float64(step)/float64(steps)
		innerExpected := polarPosition(center, node.InnerRadius, innerAngle)
		innerPoint := points[boundarySamples+step]
		g.Expect(innerPoint.X).To(BeNumerically("~", innerExpected.X, 0.000001))
		g.Expect(innerPoint.Y).To(BeNumerically("~", innerExpected.Y, 0.000001))
	}
}

func TestRenderToCanvas_WritesRecognizablePNGAndSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	const width, height = 360, 240
	root := donutRoot()
	layout := Layout(root, width, filesystem.FileLines)
	is := BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", "")
	cv := RenderToCanvas(layout, root, width, height, is, LabelMetrics{Size: filesystem.FileLines})
	outputDir := donutOutputDir(t)

	pngPath := filepath.Join(outputDir, "donut.png")
	g.Expect(cv.Render(pngPath)).To(Succeed())
	png, err := os.Open(pngPath)
	g.Expect(err).NotTo(HaveOccurred())

	defer png.Close()

	pngInfo, format, err := image.DecodeConfig(png)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
	g.Expect(pngInfo.Width).To(Equal(width))
	g.Expect(pngInfo.Height).To(Equal(height))
	pngStat, err := png.Stat()
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(pngStat.Size()).To(BeNumerically(">", 0))

	svgPath := filepath.Join(outputDir, "donut.svg")
	g.Expect(cv.Render(svgPath)).To(Succeed())
	data, err := os.ReadFile(svgPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).NotTo(BeEmpty())

	decoder := xml.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	g.Expect(err).NotTo(HaveOccurred())

	start, ok := token.(xml.StartElement)
	g.Expect(ok).To(BeTrue())
	g.Expect(start.Name.Local).To(Equal("svg"))
	attributes := make(map[string]string, len(start.Attr))
	for _, attribute := range start.Attr {
		attributes[attribute.Name.Local] = attribute.Value
	}
	g.Expect(attributes).To(HaveKeyWithValue("width", strconv.Itoa(width)))
	g.Expect(attributes).To(HaveKeyWithValue("height", strconv.Itoa(height)))
}

func TestRenderStage_SetsDrawingBoundsBeforeRenderingLegend(t *testing.T) {
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
	cfg.Legend = &config.Legend{Position: new("bottom-center")}
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
	g.Expect(BuildLegendStage(common, state)).To(Succeed())
	g.Expect(LayoutStage(common, state)).To(Succeed())
	g.Expect(RenderStage(common, state)).To(Succeed())

	g.Expect(common.Canvas.DrawingMinY()).To(Equal(common.DrawingBounds.MinY))
	g.Expect(common.Canvas.DrawingMaxY()).To(Equal(common.DrawingBounds.MaxY))

	calls := renderCalls(t, common.Canvas)

	var legendBackground *mock.Call

	for index := range calls {
		call := &calls[index]
		if call.Method == "DrawRectangle" && call.BorderWidth == 1 {
			legendBackground = call

			break
		}
	}

	if legendBackground == nil {
		t.Fatal("expected legend background")
	}

	g.Expect(legendBackground.Pos.Y).To(BeNumerically(">=", common.DrawingBounds.MinY))
	g.Expect(legendBackground.Pos.Y + legendBackground.Size.Height).
		To(BeNumerically("<=", common.DrawingBounds.MaxY))
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
