package spiral_test

import (
	"bytes"
	"image/color"
	"log/slog"
	"math"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestBuildSurface_CreatesTrianglesWithCentroidsInSpiralAnnulus(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := spiral.Layout(make([]spiral.TimeBucket, 8), 240, 240, spiral.Hourly, spiral.LabelNone)
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}

	triangles := spiral.BuildSurface(layout, values, 42)

	g.Expect(triangles).NotTo(BeEmpty())

	halfSpacing := math.Pi * layout.B
	innerRadius := math.Max(0, layout.A-halfSpacing)
	outerRadius := layout.A + layout.B*layout.MaxTheta + halfSpacing

	for _, triangle := range triangles {
		centroidX := (triangle.Points[0].X + triangle.Points[1].X + triangle.Points[2].X) / 3
		centroidY := (triangle.Points[0].Y + triangle.Points[1].Y + triangle.Points[2].Y) / 3
		distance := math.Hypot(centroidX-layout.CX, centroidY-layout.CY)

		g.Expect(distance).To(BeNumerically(">=", innerRadius))
		g.Expect(distance).To(BeNumerically("<=", outerRadius))
	}
}

func TestBuildSurface_ExtendsHalfCoilSpacingBeyondSpiralTrack(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	buckets := make([]spiral.TimeBucket, 162)

	values := make([]float64, len(buckets))
	for index := range values {
		values[index] = float64(index)
	}

	layout := spiral.Layout(buckets, 1920, 1920, spiral.Daily, spiral.LabelNone)
	triangles := spiral.BuildSurface(layout, values, 42)
	halfSpacing := math.Pi * layout.B
	innerRadius := layout.A - halfSpacing
	outerRadius := layout.A + layout.B*layout.MaxTheta + halfSpacing

	var (
		minimumRadius = math.Inf(1)
		maximumRadius float64
	)

	for _, triangle := range triangles {
		for _, point := range triangle.Points {
			radius := math.Hypot(point.X-layout.CX, point.Y-layout.CY)
			minimumRadius = math.Min(minimumRadius, radius)
			maximumRadius = math.Max(maximumRadius, radius)
		}
	}

	g.Expect(minimumRadius).To(BeNumerically("~", innerRadius))
	g.Expect(maximumRadius).To(BeNumerically("~", outerRadius))
}

func TestBuildSurface_RendersShortSpiral(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	layout := spiral.Layout(make([]spiral.TimeBucket, 3), 240, 240, spiral.Hourly, spiral.LabelNone)

	g.Expect(spiral.BuildSurface(layout, []float64{1, 2, 3}, 42)).NotTo(BeEmpty())
}

func TestBuildSurface_RejectsInsufficientOrMismatchedInputs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(spiral.BuildSurface(spiral.SpiralLayout{
		Nodes: []spiral.SpiralNode{{}, {}},
	}, []float64{1, 2}, 42)).To(BeNil())
	g.Expect(spiral.BuildSurface(spiral.SpiralLayout{
		Nodes: []spiral.SpiralNode{{}, {}, {}},
	}, []float64{1, 2}, 42)).To(BeNil())
}

func TestRenderToCanvas_RendersSurfaceBeforeSpiralForeground(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets, triangles := surfaceRenderFixture()
	surfaceInk := numericInk()
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		160,
		120,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		triangles,
		surfaceInk,
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())
	g.Expect(backend.Calls).To(HaveLen(1 + len(triangles) + 1 + len(layout.Nodes)))
	g.Expect(backend.Calls[0].Method).To(Equal("DrawRectangle"))

	for index, triangle := range triangles {
		call := backend.Calls[index+1]
		expectedColour := surfaceInk.Dip(inks.MeasureValue(triangle.Value))

		g.Expect(call.Method).To(Equal("DrawPolygon"))
		g.Expect(call.Fill).To(Equal(expectedColour))
		g.Expect(call.Border).To(Equal(expectedColour))
		g.Expect(call.BorderWidth).To(Equal(0.5))
	}

	g.Expect(backend.Calls[len(triangles)+1].Method).To(Equal("DrawPath"))

	for _, call := range backend.Calls[len(triangles)+2:] {
		g.Expect(call.Method).To(Equal("DrawDisc"))
	}
}

func TestRenderToCanvas_OverlaysAnnulusBoundaryBeforeGuideTrack(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := largeSurfaceStageBuckets()
	layout := spiral.Layout(buckets, 320, 240, spiral.Daily, spiral.LabelNone)
	triangles := []surface.Triangle{{
		Points: [3]surface.Point{
			{X: 20, Y: 30},
			{X: 40, Y: 30},
			{X: 20, Y: 50},
		},
		Value: 2,
	}}
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		320,
		240,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		triangles,
		numericInk(),
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())
	g.Expect(backend.Calls[1].Method).To(Equal("DrawPolygon"))

	boundaryCalls := backend.Calls[2:4]
	for _, call := range boundaryCalls {
		g.Expect(call.Method).To(Equal("DrawPath"))
		g.Expect(call.Fill).To(Equal(color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}))
		g.Expect(call.StrokeWidth).To(Equal(2.0))
		g.Expect(len(call.Points)).To(BeNumerically(">", 3))
		g.Expect(call.Points[len(call.Points)-1]).To(Equal(call.Points[0]))
		g.Expect(call.Points[1 : len(call.Points)-1]).NotTo(ContainElement(call.Points[0]))
	}

	guideTrack := backend.Calls[4]
	g.Expect(guideTrack.Method).To(Equal("DrawPath"))
	g.Expect(guideTrack.Fill).To(Equal(color.RGBA{R: 0xDD, G: 0xDD, B: 0xDD, A: 0xFF}))
	g.Expect(guideTrack.StrokeWidth).To(Equal(1.0))
}

func TestRenderToCanvas_WithoutSurfaceRendersNoPolygons(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets, _ := surfaceRenderFixture()
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		160,
		120,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		nil,
		nil,
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())
	g.Expect(backend.Calls).To(HaveLen(1 + 1 + len(layout.Nodes)))

	for _, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawPolygon"))
	}

	var pathCalls int
	for _, call := range backend.Calls {
		if call.Method == "DrawPath" {
			pathCalls++
		}
	}

	g.Expect(pathCalls).To(Equal(1))
}

func TestRenderStage_UsesSurfaceValuesWhenEnabled(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := surfaceStageBuckets()
	surfaceInk := inks.NumericInk(
		"surface",
		[]float64{10, 20},
		palette.GetPalette(palette.Temperature),
	)
	common := &stages.CommonState{Width: 240, Height: 240}
	state := &spiral.State{
		Buckets:        buckets,
		Inks:           spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceEnabled: true,
		SurfaceInk:     surfaceInk,
		Layout:         spiral.Layout(buckets, 240, 240, spiral.Hourly, spiral.LabelNone),
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	expectedColour := surfaceInk.Dip(inks.MeasureValue(10))

	var polygonCount int

	for _, call := range backend.Calls {
		if call.Method != "DrawPolygon" {
			continue
		}

		polygonCount++

		g.Expect(call.Fill).To(Equal(expectedColour))
	}

	g.Expect(polygonCount).To(BeNumerically(">", 0))
}

func TestRenderStage_DisabledSurfaceRendersNoPolygons(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := surfaceStageBuckets()
	common := &stages.CommonState{Width: 240, Height: 240}
	state := &spiral.State{
		Buckets:      buckets,
		Inks:         spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceInk:   numericInk(),
		Layout:       spiral.Layout(buckets, 240, 240, spiral.Hourly, spiral.LabelNone),
		LegendConfig: nil,
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	for _, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawPolygon"))
	}
}

//nolint:paralleltest // mutates global slog default logger
func TestRenderStage_WarnsAndRendersSpiralWhenSurfaceCannotBeBuilt(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	defer slog.SetDefault(oldDefault)

	buckets := []spiral.TimeBucket{
		{FillValue: 100, BorderValue: 100, SurfaceValue: 10},
		{FillValue: 100, BorderValue: 100, SurfaceValue: 20},
	}
	common := &stages.CommonState{Width: 240, Height: 240}
	state := &spiral.State{
		Buckets:        buckets,
		Inks:           spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceEnabled: true,
		SurfaceInk:     numericInk(),
		Layout:         spiral.Layout(buckets, 240, 240, spiral.Hourly, spiral.LabelNone),
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())
	g.Expect(buf.String()).To(ContainSubstring("surface rendering unavailable"))

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	for _, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawPolygon"))
	}

	g.Expect(backend.Calls).To(HaveLen(1 + 1 + len(buckets)))
}

//nolint:paralleltest // mutates global slog default logger
func TestRenderStage_RendersSurfaceFor162PointSpiral(t *testing.T) {
	g := NewGomegaWithT(t)
	buckets := largeSurfaceStageBuckets()
	layout := spiral.Layout(
		buckets,
		320,
		240-int(canvas.FooterReservedHeight),
		spiral.Daily,
		spiral.LabelNone,
	)

	values := make([]float64, len(buckets))
	for index := range buckets {
		values[index] = buckets[index].SurfaceValue
	}

	g.Expect(spiral.BuildSurface(layout, values, surfaceSeedForTest(layout))).NotTo(BeEmpty())

	var buf bytes.Buffer

	oldDefault := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	defer slog.SetDefault(oldDefault)

	common := &stages.CommonState{Width: 320, Height: 240}
	state := &spiral.State{
		Buckets:        buckets,
		Inks:           spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceEnabled: true,
		SurfaceInk:     numericInk(),
		Layout:         layout,
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())
	g.Expect(buf.String()).NotTo(ContainSubstring("surface rendering unavailable"))

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	var polygons int

	for _, call := range backend.Calls {
		if call.Method == "DrawPolygon" {
			polygons++
		}
	}

	g.Expect(polygons).To(BeNumerically(">", 0))
}

func surfaceRenderFixture() (spiral.SpiralLayout, []spiral.TimeBucket, []surface.Triangle) {
	layout := spiral.SpiralLayout{
		Nodes: []spiral.SpiralNode{
			{X: 20, Y: 30, DiscRadius: 4},
			{X: 40, Y: 50, DiscRadius: 4},
			{X: 60, Y: 70, DiscRadius: 4},
		},
	}
	buckets := []spiral.TimeBucket{
		{FillValue: 1, BorderValue: 1},
		{FillValue: 2, BorderValue: 2},
		{FillValue: 3, BorderValue: 3},
	}
	triangles := []surface.Triangle{{
		Points: [3]surface.Point{
			{X: 20, Y: 30},
			{X: 40, Y: 30},
			{X: 20, Y: 50},
		},
		Value: 2,
	}}

	return layout, buckets, triangles
}

func numericInk() inks.Ink {
	return inks.NumericInk("surface", []float64{1, 2, 3}, palette.GetPalette(palette.Temperature))
}

func surfaceStageBuckets() []spiral.TimeBucket {
	buckets := make([]spiral.TimeBucket, 8)
	for index := range buckets {
		buckets[index] = spiral.TimeBucket{
			FillValue:    100,
			BorderValue:  100,
			SurfaceValue: 10,
		}
	}

	return buckets
}

func largeSurfaceStageBuckets() []spiral.TimeBucket {
	buckets := make([]spiral.TimeBucket, 162)
	for index := range buckets {
		value := float64(index + 1)
		buckets[index] = spiral.TimeBucket{
			FillValue:    value,
			BorderValue:  value,
			SurfaceValue: value,
		}
	}

	return buckets
}

func surfaceSeedForTest(layout spiral.SpiralLayout) uint64 {
	seed := uint64(len(layout.Nodes))
	for _, dimension := range [...]float64{
		layout.CX,
		layout.CY,
		layout.A,
		layout.B,
		layout.MaxTheta,
	} {
		seed ^= math.Float64bits(dimension)
		seed *= 1099511628211
	}

	return seed
}
