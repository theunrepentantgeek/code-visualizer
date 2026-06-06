package treemap_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestApplyCanvasBlockLabels_AddsLabelsToCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "labels.svg")
	common := &stages.CommonState{
		Output: out,
		Canvas: canvas.NewCanvas(120, 80),
	}
	viz := &treemap.State{
		BlockLabels: []canvas.BlockLabel{{
			X:     10,
			Y:     10,
			W:     100,
			H:     40,
			Lines: []string{"hello", "42"},
			Ink:   color.RGBA{A: 255},
		}},
	}

	g.Expect(treemap.ApplyCanvasBlockLabels(common, viz)).NotTo(HaveOccurred())
	g.Expect(common.Canvas.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("hello"))
	g.Expect(string(data)).To(ContainSubstring("42"))
}

func TestApplyCanvasBlockLabels_NilCanvas_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{Output: "out.png", Canvas: nil}
	viz := &treemap.State{}

	g.Expect(treemap.ApplyCanvasBlockLabels(common, viz)).To(Succeed())
}
