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

func strPtr(s string) *string { return &s }

func TestResolveMetrics_FillDefaultsToSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis: strPtr("file-type"),
		YAxis: strPtr("file-lines"),
		Size:  strPtr("file-size"),
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

func TestResolveMetrics_FillAndBorderOverrideDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis: strPtr("file-lines"),
		YAxis: strPtr("file-size"),
		Size:  strPtr("file-size"),
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
