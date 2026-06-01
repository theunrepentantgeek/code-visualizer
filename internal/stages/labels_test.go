package stages

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

type fakeLabelState struct {
	CommonState
	labels []canvas.BlockLabel
}

func (s *fakeLabelState) Common() *CommonState { return &s.CommonState }

func (s *fakeLabelState) CanvasLabels() []canvas.BlockLabel { return s.labels }

func TestApplyCanvasBlockLabels_AddsLabelsToCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "labels.svg")
	state := &fakeLabelState{
		CommonState: CommonState{
			Output: out,
			Canvas: canvas.NewCanvas(120, 80),
		},
		labels: []canvas.BlockLabel{{
			X:     10,
			Y:     10,
			W:     100,
			H:     40,
			Lines: []string{"hello", "42"},
			Ink:   color.RGBA{A: 255},
		}},
	}

	err := ApplyCanvasBlockLabels(state)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(state.Canvas.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("hello"))
	g.Expect(string(data)).To(ContainSubstring("42"))
}
