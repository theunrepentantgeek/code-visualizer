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
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/surface"
)

func TestBuildSurface_CreatesTrianglesWithCentroidsInSpiralAnnulus(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout := spiral.Layout(make([]spiral.TimeBucket, 8), 240, 240, spiral.Hourly)
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8}

	triangles := spiral.BuildSurface(layout, values, 42)

	g.Expect(triangles).NotTo(BeEmpty())

	halfSpacing := math.Pi * layout.B
	innerRadius := math.Max(0, layout.A-halfSpacing)
	outerRadius := layout.A + layout.B*layout.MaxTheta + halfSpacing

	for _, triangle := range triangles {
		centroidX := (triangle.Points[0].Position.X + triangle.Points[1].Position.X + triangle.Points[2].Position.X) / 3
		centroidY := (triangle.Points[0].Position.Y + triangle.Points[1].Position.Y + triangle.Points[2].Position.Y) / 3
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

	layout := spiral.Layout(buckets, 1920, 1920, spiral.Daily)
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
			radius := math.Hypot(point.Position.X-layout.CX, point.Position.Y-layout.CY)
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
	layout := spiral.Layout(make([]spiral.TimeBucket, 3), 240, 240, spiral.Hourly)

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

func TestRenderToCanvas_RendersNumericSurfaceBandsBeforeGuideTrack(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets, triangles := bandedSurfaceRenderFixture()
	surfaceInk := numericInk()
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		160,
		120,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		spiral.RenderOptions{
			Triangles:  triangles,
			SurfaceInk: surfaceInk,
			Format:     canvas.FormatPNG,
		},
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())
	g.Expect(backend.Calls[0].Method).To(Equal("DrawRectangle"))

	surfaceCalls := leadingSurfaceFilledPathCalls(backend.Calls)
	g.Expect(surfaceCalls).To(HaveLen(2))

	for index, expectedColour := range []color.RGBA{
		surfaceInk.Dip(inks.MeasureValue(1)),
		surfaceInk.Dip(inks.MeasureValue(2)),
	} {
		call := surfaceCalls[index]
		g.Expect(call.Method).To(Equal("DrawFilledPath"))
		g.Expect(call.Fill).To(Equal(expectedColour))
		g.Expect(call.Loops).To(HaveLen(1))
	}

	g.Expect(backend.Calls[len(surfaceCalls)+1].Method).To(Equal("DrawPath"))

	for _, call := range backend.Calls[len(surfaceCalls)+2:] {
		g.Expect(call.Method).To(Equal("DrawDisc"))
	}
}

func TestRenderToCanvas_MergesSameColourNumericSurfaceFragments(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets := surfaceRenderFixture()
	triangles := []surface.Triangle{
		{
			Points: [3]surface.Sample{
				{Position: geometry.NewPoint(20, 30), Value: 1},
				{Position: geometry.NewPoint(40, 30), Value: 1},
				{Position: geometry.NewPoint(20, 50), Value: 1},
			},
			Value: 1,
		},
		{
			Points: [3]surface.Sample{
				{Position: geometry.NewPoint(40, 30), Value: 1},
				{Position: geometry.NewPoint(40, 50), Value: 1},
				{Position: geometry.NewPoint(20, 50), Value: 1},
			},
			Value: 1,
		},
	}
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		160,
		120,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		spiral.RenderOptions{
			Triangles:  triangles,
			SurfaceInk: inks.NumericInk("surface", []float64{1}, palette.GetPalette(palette.Temperature)),
			Format:     canvas.FormatPNG,
		},
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var filledPaths []mock.Call

	for _, call := range backend.Calls {
		if call.Method == "DrawFilledPath" {
			filledPaths = append(filledPaths, call)
		}
	}

	g.Expect(filledPaths).To(HaveLen(1))

	if len(filledPaths) != 1 {
		return
	}

	g.Expect(filledPaths[0].Loops).To(HaveLen(2))
}

func TestRenderToCanvas_UsesFlatSurfaceFallbackForFixedInk(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets, triangles := bandedSurfaceRenderFixture()
	fixedColour := color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xFF}
	surfaceInk := inks.FixedInk(fixedColour)
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		320,
		240,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		spiral.RenderOptions{
			Triangles:  triangles,
			SurfaceInk: surfaceInk,
			Format:     canvas.FormatPNG,
		},
	)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	surfaceCalls := leadingSurfaceFilledPathCalls(backend.Calls)
	g.Expect(surfaceCalls).To(HaveLen(1))
	g.Expect(surfaceCalls[0].Fill).To(Equal(fixedColour))
	g.Expect(surfaceCalls[0].Loops).To(HaveLen(1))
}

func TestRenderToCanvas_WithoutSurfaceRendersNoPolygons(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	layout, buckets := surfaceRenderFixture()
	cv := spiral.RenderToCanvas(
		layout,
		buckets,
		160,
		120,
		spiral.Inks{Fill: numericInk(), Border: numericInk()},
		spiral.RenderOptions{
			Format: canvas.FormatPNG,
		},
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
	common := &stages.CommonState{Width: 240, Height: 240, Output: "spiral.png"}
	state := &spiral.State{
		Buckets:        buckets,
		Inks:           spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceEnabled: true,
		SurfaceInk:     surfaceInk,
		Layout:         spiral.Layout(buckets, 240, 240, spiral.Hourly),
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	expectedColour := surfaceInk.Dip(inks.MeasureValue(10))

	var filledPathCount int

	for _, call := range backend.Calls {
		if call.Method != "DrawFilledPath" {
			continue
		}

		filledPathCount++

		g.Expect(call.Fill).To(Equal(expectedColour))
	}

	g.Expect(filledPathCount).To(BeNumerically(">", 0))
}

func TestRenderStage_DisabledSurfaceRendersNoPolygons(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := surfaceStageBuckets()
	common := &stages.CommonState{Width: 240, Height: 240, Output: "spiral.png"}
	state := &spiral.State{
		Buckets:      buckets,
		Inks:         spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceInk:   numericInk(),
		Layout:       spiral.Layout(buckets, 240, 240, spiral.Hourly),
		LegendConfig: nil,
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	for _, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawFilledPath"))
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
	common := &stages.CommonState{Width: 240, Height: 240, Output: "spiral.png"}
	state := &spiral.State{
		Buckets:        buckets,
		Inks:           spiral.Inks{Fill: numericInk(), Border: numericInk()},
		SurfaceEnabled: true,
		SurfaceInk:     numericInk(),
		Layout:         spiral.Layout(buckets, 240, 240, spiral.Hourly),
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())
	g.Expect(buf.String()).To(ContainSubstring("surface rendering unavailable"))

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	for _, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawFilledPath"))
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

	common := &stages.CommonState{Width: 320, Height: 240, Output: "spiral.png"}
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

	var filledPaths int

	for _, call := range backend.Calls {
		if call.Method == "DrawFilledPath" {
			filledPaths++
		}
	}

	g.Expect(filledPaths).To(BeNumerically(">", 0))
}

func surfaceRenderFixture() (spiral.SpiralLayout, []spiral.TimeBucket) {
	layout := spiral.SpiralLayout{
		Nodes: []spiral.SpiralNode{
			{Geometry: geometry.NewCircle(geometry.NewPoint(20, 30), 4)},
			{Geometry: geometry.NewCircle(geometry.NewPoint(40, 50), 4)},
			{Geometry: geometry.NewCircle(geometry.NewPoint(60, 70), 4)},
		},
	}
	buckets := []spiral.TimeBucket{
		{FillValue: 1, BorderValue: 1},
		{FillValue: 2, BorderValue: 2},
		{FillValue: 3, BorderValue: 3},
	}

	return layout, buckets
}

func bandedSurfaceRenderFixture() (spiral.SpiralLayout, []spiral.TimeBucket, []surface.Triangle) {
	buckets := largeSurfaceStageBuckets()
	layout := spiral.Layout(buckets, 320, 240, spiral.Daily)
	triangles := []surface.Triangle{{
		Points: [3]surface.Sample{
			{Position: geometry.NewPoint(20, 30), Value: 1},
			{Position: geometry.NewPoint(40, 30), Value: 1},
			{Position: geometry.NewPoint(20, 50), Value: 3},
		},
		Value: 5.0 / 3.0,
	}}

	return layout, buckets, triangles
}

func numericInk() inks.Ink {
	return inks.NumericInk("surface", []float64{1, 2, 3}, palette.GetPalette(palette.Temperature))
}

func leadingSurfaceFilledPathCalls(calls []mock.Call) []mock.Call {
	filledPaths := make([]mock.Call, 0, len(calls))

	for _, call := range calls[1:] {
		if call.Method != "DrawFilledPath" {
			break
		}

		filledPaths = append(filledPaths, call)
	}

	return filledPaths
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
