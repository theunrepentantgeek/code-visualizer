package scatter_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/scatter"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveMetrics_FillDefaultsToSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis: new("file-type"),
		YAxis: new("file-lines"),
		Size:  new("file-size"),
	}

	err := scatter.ResolveMetrics(common, viz, cfg)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(viz.XAxis).To(Equal(scatter.AxisSpec{Metric: filesystem.FileType, Kind: metric.Classification}))
	g.Expect(viz.YAxis).To(Equal(scatter.AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity}))
	g.Expect(viz.Size).To(Equal(filesystem.FileSize))
	g.Expect(viz.FillMetric).To(Equal(filesystem.FileSize))
	g.Expect(common.Requested).To(Equal([]metric.Name{
		filesystem.FileType,
		filesystem.FileLines,
		filesystem.FileSize,
	}))
}

func TestResolveMetrics_ParsesLogScale(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis:  new("file-lines"),
		YAxis:  new("file-size"),
		Size:   new("file-size"),
		XScale: new("log"),
		YScale: new("linear"),
	}

	err := scatter.ResolveMetrics(common, viz, cfg)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(viz.XAxis.Scale).To(Equal(scatter.Log))
	g.Expect(viz.YAxis.Scale).To(Equal(scatter.Linear))
}

func TestResolveMetrics_FillAndBorderOverrideDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis: new("file-lines"),
		YAxis: new("file-size"),
		Size:  new("file-size"),
		Fill:  &config.MetricSpec{Metric: filesystem.FileType},
		Border: &config.MetricSpec{
			Metric: filesystem.FileLines,
		},
	}

	err := scatter.ResolveMetrics(common, viz, cfg)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(viz.FillMetric).To(Equal(filesystem.FileType))
	g.Expect(viz.BorderMetric).To(Equal(filesystem.FileLines))
	g.Expect(common.Requested).To(Equal([]metric.Name{
		filesystem.FileLines,
		filesystem.FileSize,
		filesystem.FileType,
	}))
}
