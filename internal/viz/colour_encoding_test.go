package viz

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// ---------------------------------------------------------------------------
// ColourEncoding.IsSet
// ---------------------------------------------------------------------------

func TestColourEncoding_IsSet_ReflectsMetricSelection(t *testing.T) {
	t.Parallel()

	// Arrange
	cases := map[string]struct {
		encoding ColourEncoding
		expected bool
	}{
		"empty encoding": {},
		"selected metric": {
			encoding: ColourEncoding{
				Metric:  metric.Name("file-type"),
				Palette: palette.Categorization,
			},
			expected: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			// Act
			actual := c.encoding.IsSet()

			// Assert
			g.Expect(actual).To(Equal(c.expected))
		})
	}
}
