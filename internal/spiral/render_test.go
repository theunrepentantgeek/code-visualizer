package spiral_test

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

//nolint:dupl // Intentionally parallel structure testing different output formats
func TestRenderToCanvas_PNG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 800, 600, spiral.Hourly)
	shapeInks := spiral.BuildInks(buckets, stages.RequestedMetrics{}, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 800, 600, shapeInks, spiral.RenderOptions{
		Format: canvas.FormatPNG,
	})

	out := filepath.Join(t.TempDir(), "spiral.png")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("png"))
}

func TestRenderToCanvas_SVG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly)
	shapeInks := spiral.BuildInks(buckets, stages.RequestedMetrics{}, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, shapeInks, spiral.RenderOptions{
		Format: canvas.FormatSVG,
	})

	out := filepath.Join(t.TempDir(), "spiral.svg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	decoder := xml.NewDecoder(bytes.NewReader(data))

	var rootElement string

	for {
		tok, xmlErr := decoder.Token()
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

//nolint:dupl // Intentionally parallel structure testing different output formats
func TestRenderToCanvas_JPG(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	buckets := sampleTimeBuckets()
	layout := spiral.Layout(buckets, 400, 300, spiral.Hourly)
	shapeInks := spiral.BuildInks(buckets, stages.RequestedMetrics{}, "", "", "", "")
	cv := spiral.RenderToCanvas(layout, buckets, 400, 300, shapeInks, spiral.RenderOptions{
		Format: canvas.FormatJPG,
	})

	out := filepath.Join(t.TempDir(), "spiral.jpg")
	err := cv.Render(out)
	g.Expect(err).NotTo(HaveOccurred())

	f, err := os.Open(out)
	g.Expect(err).NotTo(HaveOccurred())

	defer f.Close()

	_, format, err := image.DecodeConfig(f)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(format).To(Equal("jpeg"))
}

func TestRenderStage_RendersOnlyActiveDiscLabelsBeforeLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	black := color.RGBA{A: 255}

	common := &stages.CommonState{
		Width:  200,
		Height: 200,
		Output: "spiral.svg",
	}
	state := &spiral.State{
		Buckets: []spiral.TimeBucket{
			{FillValue: 1, BorderValue: 1},
			{FillValue: 2, BorderValue: 2},
		},
		Layout: spiral.SpiralLayout{Nodes: []spiral.SpiralNode{
			{Geometry: geometry.Circle{Center: geometry.Point{X: 50, Y: 50}, Radius: 20}},
			{Geometry: geometry.Circle{Center: geometry.Point{X: 100, Y: 100}}},
		}},
		Inks: spiral.Inks{
			Fill:   inks.FixedInk(black),
			Border: inks.FixedInk(black),
		},
		DiscLabels: []canvas.BlockLabel{{
			Bounds: geometry.Rect{Min: geometry.Point{X: 10, Y: 10}, Max: geometry.Point{X: 90, Y: 90}},
			Lines:  []string{"7", "Aug", "3"},
		}},
		LegendConfig: &legend.Config{
			Position:    model.LegendPositionBottomRight,
			Orientation: model.LegendOrientationVertical,
			Entries: []legend.Entry{{
				Role: legend.RoleFill, MetricName: "file-lines", Ink: inks.FixedInk(black),
			}},
		},
	}

	g.Expect(spiral.RenderStage(common, state)).To(Succeed())

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	labelIndexes := make(map[string]int)
	fillIndex := -1

	for index, call := range backend.Calls {
		g.Expect(call.Method).NotTo(Equal("DrawArcText"))

		if call.Method != "DrawText" {
			continue
		}

		g.Expect(call.Rotation).To(BeZero())

		if call.Text == "Fill" {
			fillIndex = index
		}

		if call.Text == "7" || call.Text == "Aug" || call.Text == "3" {
			labelIndexes[call.Text] = index
			g.Expect(call.Anchor).To(Equal(canvas.AnchorMiddle))
		}

		g.Expect(call.Text).NotTo(Equal("8"))
	}

	g.Expect(labelIndexes).To(HaveLen(3))
	g.Expect(fillIndex).To(BeNumerically(">=", 0))

	for _, labelIndex := range labelIndexes {
		g.Expect(labelIndex).To(BeNumerically("<", fillIndex))
	}
}

func TestRenderStage_RejectsUnknownDiscLabelFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := spiral.RenderStage(
		&stages.CommonState{Width: 200, Height: 200, Output: "spiral.gif"},
		&spiral.State{},
	)

	g.Expect(err).To(MatchError(ContainSubstring("resolve spiral label format")))
}
