package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"

	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestLayoutStage_ReservesLegendSpace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		position     canvasmodel.LegendPosition
		orientation  canvasmodel.LegendOrientation
		startOnX     bool
		startMessage string
	}{
		{
			name:         "top legend",
			position:     canvasmodel.LegendPositionTopCenter,
			orientation:  canvasmodel.LegendOrientationHorizontal,
			startMessage: "bubble layout should start below the reserved top legend area",
		},
		{
			name:         "left legend",
			position:     canvasmodel.LegendPositionCenterLeft,
			orientation:  canvasmodel.LegendOrientationVertical,
			startOnX:     true,
			startMessage: "bubble layout should start to the right of the reserved left legend area",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := testLegendConfig(tt.position, tt.orientation)
			common := &stages.CommonState{Root: testLayoutRoot(), Width: 1200, Height: 800}
			stages.InitDrawingBounds(common) //nolint:errcheck // always succeeds

			viz := &State{
				Size:         filesystem.FileSize,
				Labels:       LabelAll,
				LegendConfig: cfg,
			}

			g.Expect(LayoutStage(common, viz)).To(Succeed())

			wReduce, hReduce := cfg.ReserveSpace()
			layoutW, layoutH := legend.ReserveAndLayout(cfg, common.Width, common.Height)
			dx, dy := legend.LayoutOffset(cfg, wReduce, hReduce)
			box := contentBoundsForTest(viz.Nodes)

			if tt.startOnX {
				g.Expect(box.minX).To(BeNumerically(">=", dx-1.0), tt.startMessage)
				g.Expect(box.maxX).To(BeNumerically("<=", dx+float64(layoutW)+1.0))
				g.Expect(box.minY).To(BeNumerically(">=", dy-1.0))
				g.Expect(box.maxY).To(BeNumerically("<=", dy+float64(layoutH)+1.0))

				return
			}

			g.Expect(box.minY).To(BeNumerically(">=", dy-1.0), tt.startMessage)
			g.Expect(box.maxY).To(BeNumerically("<=", dy+float64(layoutH)+1.0))
			g.Expect(box.minX).To(BeNumerically(">=", dx-1.0))
			g.Expect(box.maxX).To(BeNumerically("<=", dx+float64(layoutW)+1.0))
		})
	}
}

func testLegendConfig(pos canvasmodel.LegendPosition, orient canvasmodel.LegendOrientation) *legend.Config {
	fill := inks.NumericInk("file-size", []float64{100, 200, 400}, palette.GetPalette(palette.Temperature))

	return &legend.Config{
		Position:    pos,
		Orientation: orient,
		Entries: []legend.Entry{{
			Role:       legend.RoleFill,
			MetricName: "file-size",
			Ink:        fill,
		}},
	}
}

func testLayoutRoot() *model.Directory {
	root := &model.Directory{
		Name: "root",
		Path: "root",
		Files: []*model.File{
			testLayoutFile("root/a.go", 100),
			testLayoutFile("root/b.go", 200),
			testLayoutFile("root/c.go", 300),
		},
	}

	return root
}

func testLayoutFile(path string, size int64) *model.File {
	f := &model.File{Name: path, Path: path}
	f.SetQuantity(filesystem.FileSize, size)

	return f
}
