package bubbletree

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestLayoutStage_ReservesTopLegendSpace(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := testLegendConfig(canvasmodel.LegendPositionTopCenter, canvasmodel.LegendOrientationHorizontal)
	state := &State{
		CommonState:  stages.CommonState{Root: testLayoutRoot(), Width: 1200, Height: 800},
		Size:         filesystem.FileSize,
		Labels:       LabelAll,
		LegendConfig: cfg,
	}

	g.Expect(LayoutStage(state)).To(Succeed())

	wReduce, hReduce := cfg.ReserveSpace()
	layoutW, layoutH := legend.ReserveAndLayout(cfg, state.Width, state.Height)
	dx, dy := legend.LayoutOffset(cfg, wReduce, hReduce)
	box := childrenBounds(&state.Nodes)

	g.Expect(box.minY).To(BeNumerically(">=", dy-1.0),
		"bubble layout should start below the reserved top legend area")
	g.Expect(box.maxY).To(BeNumerically("<=", dy+float64(layoutH)+1.0))
	g.Expect(box.minX).To(BeNumerically(">=", dx-1.0))
	g.Expect(box.maxX).To(BeNumerically("<=", dx+float64(layoutW)+1.0))
}

func TestLayoutStage_ReservesLeftLegendSpace(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := testLegendConfig(canvasmodel.LegendPositionCenterLeft, canvasmodel.LegendOrientationVertical)
	state := &State{
		CommonState:  stages.CommonState{Root: testLayoutRoot(), Width: 1200, Height: 800},
		Size:         filesystem.FileSize,
		Labels:       LabelAll,
		LegendConfig: cfg,
	}

	g.Expect(LayoutStage(state)).To(Succeed())

	wReduce, hReduce := cfg.ReserveSpace()
	layoutW, layoutH := legend.ReserveAndLayout(cfg, state.Width, state.Height)
	dx, dy := legend.LayoutOffset(cfg, wReduce, hReduce)
	box := childrenBounds(&state.Nodes)

	g.Expect(box.minX).To(BeNumerically(">=", dx-1.0),
		"bubble layout should start to the right of the reserved left legend area")
	g.Expect(box.maxX).To(BeNumerically("<=", dx+float64(layoutW)+1.0))
	g.Expect(box.minY).To(BeNumerically(">=", dy-1.0))
	g.Expect(box.maxY).To(BeNumerically("<=", dy+float64(layoutH)+1.0))
}

func testLegendConfig(pos canvasmodel.LegendPosition, orient canvasmodel.LegendOrientation) *canvas.LegendConfig {
	fill := canvas.NumericInk("file-size", []float64{100, 200, 400}, palette.GetPalette(palette.Temperature))

	return &canvas.LegendConfig{
		Position:    pos,
		Orientation: orient,
		Entries: []canvas.LegendEntry{{
			Role:       canvas.LegendRoleFill,
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
