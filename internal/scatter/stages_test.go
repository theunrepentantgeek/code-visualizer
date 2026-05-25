package scatter

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestResolveMetrics_FillDefaultsToSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &State{
		Config: &config.Scatter{
			XAxis: new("file-type"),
			YAxis: new("file-lines"),
			Size:  new("file-size"),
		},
	}

	err := ResolveMetrics(s)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s.XAxis).To(Equal(AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification}))
	g.Expect(s.YAxis).To(Equal(AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity}))
	g.Expect(s.Size).To(Equal(filesystem.FileSize))
	g.Expect(s.FillMetric).To(Equal(filesystem.FileSize))
	g.Expect(s.Common().Requested).To(Equal([]metric.Name{
		filesystem.FileType,
		filesystem.FileLines,
		filesystem.FileSize,
	}))
}

func TestResolveMetrics_FillAndBorderOverrideDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &State{
		Config: &config.Scatter{
			XAxis: new("file-lines"),
			YAxis: new("file-size"),
			Size:  new("file-size"),
			Fill:  &config.MetricSpec{Metric: filesystem.FileType},
			Border: &config.MetricSpec{
				Metric: filesystem.FileLines,
			},
		},
	}

	err := ResolveMetrics(s)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(s.FillMetric).To(Equal(filesystem.FileType))
	g.Expect(s.BorderMetric).To(Equal(filesystem.FileLines))
	g.Expect(s.Common().Requested).To(Equal([]metric.Name{
		filesystem.FileLines,
		filesystem.FileSize,
		filesystem.FileType,
	}))
}
