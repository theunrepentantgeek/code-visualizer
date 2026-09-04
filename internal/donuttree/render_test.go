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
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
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

func TestRenderToCanvas_UsesNarrowedRingGeometryForSectorsAndLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 600, filesystem.FileLines)
	is := BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", "")

	calls := renderCalls(t, RenderToCanvas(layout, root, 600, 600, is, LabelMetrics{}))
	polygons := callsNamed(calls, "DrawPolygon")
	expectedOuterRadius := layout.AnchorRadius * 1.9

	g.Expect(polygons).NotTo(BeEmpty())
	g.Expect(layout.Center.DistanceTo(polygons[0].Points[0])).
		To(BeNumerically("~", expectedOuterRadius, 0.000001))

	var directoryLabel *mock.Call

	for index := range calls {
		if calls[index].Method == "DrawText" && calls[index].Text == "src" {
			directoryLabel = &calls[index]

			break
		}
	}

	if directoryLabel == nil {
		t.Fatal("expected src directory label")

		return
	}

	expectedMidRadius := layout.AnchorRadius * 1.45
	g.Expect(layout.Center.DistanceTo(directoryLabel.Pos)).
		To(BeNumerically("~", expectedMidRadius, 0.000001))
}

func TestBuildLegendStage_AddsArcLabelSampleLines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Legend = &config.Legend{Position: new("bottom-right")}
	cfg.DonutTree.Fill = &config.MetricSpec{Metric: "file-type"}
	cfg.DonutTree.Border = &config.MetricSpec{Metric: "file-size"}
	state := &State{
		SizeMetric:   "file-lines.sum",
		FillMetric:   "file-type.mode",
		BorderMetric: "file-size.sum",
		Inks: Inks{ShapeInks: inks.ShapeInks{
			Fill:   inks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			Border: inks.FixedInk(color.RGBA{A: 255}),
		}},
	}

	g.Expect(BuildLegendStage(&stages.CommonState{RootConfig: cfg}, state)).To(Succeed())
	g.Expect(state.LegendConfig).NotTo(BeNil())

	if state.LegendConfig == nil {
		t.Fatal("expected legend config")
	}

	g.Expect(state.LegendConfig.LabelSample).To(Equal(legend.LabelSample{
		Shape: legend.LabelSampleArc,
		Lines: []string{"directory-name", "file-lines.sum", "file-type.mode", "file-size.sum"},
	}))
}

func TestBuildLegendStage_OmitsDerivedMetricsFromLabelSample(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Legend = &config.Legend{Position: new("bottom-right")}
	state := &State{
		SizeMetric:   "file-lines.sum",
		FillMetric:   "file-lines.sum",
		BorderMetric: "file-size.sum",
		Inks: Inks{ShapeInks: inks.ShapeInks{
			Fill:   inks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			Border: inks.FixedInk(color.RGBA{A: 255}),
		}},
	}

	g.Expect(BuildLegendStage(&stages.CommonState{RootConfig: cfg}, state)).To(Succeed())
	g.Expect(state.LegendConfig).NotTo(BeNil())

	if state.LegendConfig == nil {
		t.Fatal("expected legend config")
	}

	g.Expect(state.LegendConfig.LabelSample.Lines).To(Equal([]string{"directory-name", "file-lines.sum"}))
}

func TestRenderToCanvas_OmitsBorderPolygonsUnlessConfigured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	root := donutRoot()
	layout := Layout(root, 600, filesystem.FileLines)

	polygons := callsNamed(renderCalls(t, RenderToCanvas(
		layout, root, 600, 600,
		BuildInks(root, stages.RequestedMetrics{}, filesystem.FileLines, palette.Neutral, "", ""),
		LabelMetrics{Size: filesystem.FileLines},
	)), "DrawPolygon")

	g.Expect(polygons).To(HaveLen(2))

	for _, call := range polygons {
		g.Expect(call.BorderWidth).To(BeZero())
	}
}

func TestRenderToCanvas_InsetsMetricBordersInsideAdjacentSectors(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	left := donutDirectory("left", 100)
	right := donutDirectory("right", 100)
	root := donutDirectory("root", 200)
	root.Dirs = []*model.Directory{left, right}

	const borderMetric = metric.Name("file-freshness.sum")
	left.SetQuantity(borderMetric, 1)
	right.SetQuantity(borderMetric, 2)

	layout := Layout(root, 600, filesystem.FileLines)
	is := BuildInks(
		root,
		stages.CollectRequestedMetrics(borderMetric),
		filesystem.FileLines,
		palette.Neutral,
		borderMetric,
		palette.GoodBad,
	)
	polygons := callsNamed(renderCalls(t, RenderToCanvas(
		layout, root, 600, 600,
		is,
		LabelMetrics{Size: filesystem.FileLines},
	)), "DrawPolygon")

	g.Expect(polygons).To(HaveLen(4))

	for index := 0; index < len(polygons); index += 2 {
		fill := polygons[index]
		border := polygons[index+1]

		g.Expect(fill.BorderWidth).To(BeZero())
		g.Expect(fill.Points).To(Equal(sectorPoints(layout.Children[index/2], layout.Center)))
		g.Expect(border.Fill.A).To(BeZero())
		g.Expect(border.BorderWidth).To(Equal(donutSectorBorderWidth))
	}

	leftBorder := polygons[1].Points
	rightBorder := polygons[3].Points
	leftEnd := leftBorder[len(leftBorder)/2-1]
	rightStart := rightBorder[0]
	g.Expect(leftEnd.DistanceTo(rightStart)).
		To(BeNumerically("~", donutSectorBorderWidth, 0.000001))
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
	labelColours := make(map[string]color.RGBA)

	for _, call := range callsNamed(calls, "DrawText") {
		if call.Text == "dark" || call.Text == "light" {
			labelColours[call.Text] = call.Fill
		}
	}

	darkExpected := canvas.TextColourFor(is.Fill.Dip(inks.MetricValueForDirectory(dark, is.Fill)))
	lightExpected := canvas.TextColourFor(is.Fill.Dip(inks.MetricValueForDirectory(light, is.Fill)))

	g.Expect(labelColours).To(HaveKeyWithValue("dark", darkExpected))
	g.Expect(labelColours).To(HaveKeyWithValue("light", lightExpected))
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
	center := geometry.Point{X: 120, Y: 160}

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

func TestInsetSectorPoints_KeepsBorderStrokeInsideSector(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  0,
		SweepAngle:  math.Pi / 3,
		InnerRadius: 40,
		OuterRadius: 80,
	}
	center := geometry.Point{X: 120, Y: 160}
	points := insetSectorPoints(node, center, donutSectorBorderWidth)
	halfWidth := donutSectorBorderWidth / 2
	outerCount := sectorSteps(node.SweepAngle) + 1

	for _, point := range points {
		g.Expect(math.IsNaN(point.X) || math.IsInf(point.X, 0)).To(BeFalse())
		g.Expect(math.IsNaN(point.Y) || math.IsInf(point.Y, 0)).To(BeFalse())
	}

	g.Expect(center.DistanceTo(points[0])).
		To(BeNumerically("~", node.OuterRadius-halfWidth, 0.000001))
	g.Expect(center.DistanceTo(points[outerCount])).
		To(BeNumerically("~", node.InnerRadius+halfWidth, 0.000001))
	g.Expect(math.Abs(points[0].Y - center.Y)).To(BeNumerically("~", halfWidth, 0.000001))
	g.Expect(math.Abs(points[len(points)-2].Y - center.Y)).To(BeNumerically("~", halfWidth, 0.000001))
}

func TestInsetSectorPoints_KeepsNarrowSectorGeometryFinite(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	node := DonutNode{
		StartAngle:  0,
		SweepAngle:  math.Pi / 10000,
		InnerRadius: 40,
		OuterRadius: 80,
	}

	points := insetSectorPoints(node, geometry.Point{X: 120, Y: 160}, donutSectorBorderWidth)
	for _, point := range points {
		g.Expect(math.IsNaN(point.X) || math.IsInf(point.X, 0)).To(BeFalse())
		g.Expect(math.IsNaN(point.Y) || math.IsInf(point.Y, 0)).To(BeFalse())
	}

	g.Expect(points[0].DistanceTo(points[sectorSteps(node.SweepAngle)])).To(BeNumerically(">", 0))
}

func TestInsetSectorPoints_ScalesNarrowAdjacentBordersToRemainDisjoint(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	center := geometry.Point{X: 120, Y: 160}
	left := DonutNode{
		StartAngle:  0,
		SweepAngle:  math.Pi / 180,
		InnerRadius: 50,
		OuterRadius: 100,
	}
	right := left
	right.StartAngle = left.EndAngle()

	borderWidth := sectorBorderWidth(left)
	leftPoints := insetSectorPoints(left, center, borderWidth)
	rightPoints := insetSectorPoints(right, center, borderWidth)
	leftEnd := leftPoints[sectorSteps(left.SweepAngle)]
	rightStart := rightPoints[0]

	g.Expect(borderWidth).To(BeNumerically("<", donutSectorBorderWidth))
	g.Expect(leftEnd.DistanceTo(rightStart)).
		To(BeNumerically("~", borderWidth, 0.000001))

	leftStartInner := leftPoints[len(leftPoints)-2]
	leftEndInner := leftPoints[sectorSteps(left.SweepAngle)+1]
	g.Expect(leftStartInner.DistanceTo(leftEndInner)).To(BeNumerically(">=", borderWidth-0.000001))
}

func TestSectorBorderWidth_PreservesFullCircleBorders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(sectorBorderWidth(DonutNode{
		SweepAngle:  2 * math.Pi,
		InnerRadius: 50,
		OuterRadius: 100,
	})).To(Equal(donutSectorBorderWidth))
}

func TestRenderToCanvas_WritesRecognizablePNGAndSVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		width, height = 360, 240
		borderMetric  = metric.Name("file-freshness.sum")
	)

	root := donutRoot()
	root.SetQuantity(borderMetric, 1)
	root.Dirs[0].SetQuantity(borderMetric, 2)
	root.Dirs[0].Dirs[0].SetQuantity(borderMetric, 3)
	layout := Layout(root, width, filesystem.FileLines)
	is := BuildInks(
		root,
		stages.CollectRequestedMetrics(borderMetric),
		filesystem.FileLines,
		palette.Neutral,
		borderMetric,
		palette.GoodBad,
	)
	cv := RenderToCanvas(layout, root, width, height, is, LabelMetrics{Size: filesystem.FileLines})
	outputDir := donutOutputDir(t)

	pngPath := filepath.Join(outputDir, "donut.png")
	g.Expect(cv.Render(pngPath)).To(Succeed())

	png, err := os.Open(pngPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer png.Close()

	pngInfo, format, err := image.DecodeConfig(png)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
	g.Expect(pngInfo.Width).To(Equal(width))
	g.Expect(pngInfo.Height).To(Equal(height))

	pngStat, err := png.Stat()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pngStat.Size()).To(BeNumerically(">", 0))

	svgPath := filepath.Join(outputDir, "donut.svg")
	g.Expect(cv.Render(svgPath)).To(Succeed())
	data, err := os.ReadFile(svgPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).NotTo(BeEmpty())
	g.Expect(string(data)).To(ContainSubstring(`fill="rgba(0,0,0,0.000)"`))
	g.Expect(string(data)).To(ContainSubstring(`stroke-width="1.000"`))

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

	g.Expect(common.Canvas.DrawingMinY()).To(Equal(int(common.DrawingBounds.Min.Y)))
	g.Expect(common.Canvas.DrawingMaxY()).To(Equal(int(common.DrawingBounds.Max.Y)))

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

		return
	}

	g.Expect(legendBackground.Pos.Y).To(BeNumerically(">=", common.DrawingBounds.Min.Y))
	g.Expect(legendBackground.Bounds.Max.Y).
		To(BeNumerically("<=", common.DrawingBounds.Max.Y))
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
