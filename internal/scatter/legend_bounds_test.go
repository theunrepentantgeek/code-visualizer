package scatter

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestRenderStage_ReflowsLegendWithinReservedDrawingBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	categories := []string{"go", "md", "txt"}
	ink := inks.CategoricalInk("type", categories, palette.GetPalette(palette.Categorization))
	cfg := &legend.Config{
		Position:    canvasmodel.LegendPositionBottomRight,
		Orientation: canvasmodel.LegendOrientationVertical,
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "type", Ink: ink},
			{Role: legend.RoleBorder, MetricName: "type", Ink: ink},
		},
	}
	common := &stages.CommonState{
		Width: 320, Height: 320,
		DrawingBounds: geometry.Rect{
			Min: geometry.NewPoint(0, 38),
			Max: geometry.NewPoint(320, 293),
		},
	}
	reservation := legend.ReserveLayout(cfg, common.Width, int(common.DrawingBounds.Height()))

	g.Expect(RenderStage(common, &State{LegendConfig: cfg})).To(Succeed())
	g.Expect(common.Canvas.DrawingMinY()).To(Equal(38))
	g.Expect(common.Canvas.DrawingMaxY()).To(Equal(293))

	backend := mock.NewBackend()
	g.Expect(common.Canvas.RenderTo(backend)).To(Succeed())

	var background *mock.Call

	for i := range backend.Calls {
		call := &backend.Calls[i]
		if call.Method == "DrawRectangle" && call.Fill == (color.RGBA{R: 255, G: 255, B: 255, A: 230}) {
			background = call

			break
		}
	}

	if background == nil {
		t.Fatal("legend background not rendered")
	}

	g.Expect(background.Bounds.Min.Y).To(BeNumerically(">=", 38))
	g.Expect(background.Bounds.Max.Y).To(BeNumerically("<=", 293))
	g.Expect(background.Bounds.Width() + 2*canvasmodel.LegendMargin).
		To(BeNumerically("~", float64(common.Width-reservation.Width), 1))
}
