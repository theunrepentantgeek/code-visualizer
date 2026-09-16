package viz_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

func TestColourEncoding_IsSet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(viz.ColourEncoding{}.IsSet()).To(BeFalse())
	g.Expect(viz.ColourEncoding{
		Metric:  metric.Name("file-type"),
		Palette: palette.Categorization,
	}.IsSet()).To(BeTrue())
}
