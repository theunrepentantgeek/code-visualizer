package scatter_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	pkginks "github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
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
	g.Expect(common.Requested.BaseMetrics).To(Equal([]metric.Name{
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
	g.Expect(common.Requested.BaseMetrics).To(Equal([]metric.Name{
		filesystem.FileLines,
		filesystem.FileSize,
		filesystem.FileType,
	}))
}

func TestBuildInksStage_UsesRequestedDescriptorForExpressionFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const expressionMetric = metric.Name("file-size.count")

	first := &model.File{Name: "main.go", Path: "main.go"}
	first.SetQuantity(filesystem.FileLines, 120)
	first.SetQuantity(filesystem.FileSize, 100)
	first.SetQuantity(expressionMetric, 2)

	second := &model.File{Name: "readme.md", Path: "readme.md"}
	second.SetQuantity(filesystem.FileLines, 40)
	second.SetQuantity(filesystem.FileSize, 60)
	second.SetQuantity(expressionMetric, 1)

	common := &stages.CommonState{
		Root: &model.Directory{Files: []*model.File{first, second}},
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{{
				ResultName: expressionMetric,
				ResultKind: metric.Quantity,
			}},
		},
	}
	viz := &scatter.State{
		XAxis:       scatter.AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: scatter.Linear},
		YAxis:       scatter.AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: scatter.Linear},
		Size:        filesystem.FileSize,
		FillMetric:  expressionMetric,
		FillPalette: palette.Temperature,
	}

	err := scatter.BuildInksStage(common, viz)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(viz.Inks.Fill.Info().Kind).To(Equal(pkginks.KindNumeric))
	g.Expect(viz.Inks.Fill.Info().MetricName).To(Equal(expressionMetric))
}
