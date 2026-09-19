package alluvial_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestRenderToCanvas_AddsFilledPathForEachValidFlow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := alluvial.RenderToCanvas(alluvial.Layout{
		Flows: []alluvial.Flow{
			{Path: "continuing", FromX: 20, ToX: 180, FromTop: 10, FromBottom: 40, ToTop: 20, ToBottom: 70},
			{Path: "introduced", FromX: 20, ToX: 180, FromTop: 60, FromBottom: 60, ToTop: 50, ToBottom: 80},
			{Path: "empty", FromX: 20, ToX: 180, FromTop: 90, FromBottom: 90, ToTop: 90, ToBottom: 90},
		},
	}, 200, 100)
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	filledPaths := make([]mock.Call, 0)

	for _, call := range backend.Calls {
		if call.Method == "DrawFilledPath" {
			filledPaths = append(filledPaths, call)
		}
	}

	g.Expect(filledPaths).To(HaveLen(2))
	g.Expect(filledPaths[0].Loops).To(Equal([][]geometry.Point{{
		{X: 20, Y: 10}, {X: 180, Y: 20}, {X: 180, Y: 70}, {X: 20, Y: 40},
	}}))
	g.Expect(filledPaths[1].Loops[0][0]).To(Equal(filledPaths[1].Loops[0][3]))
}
