package radialtree_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveRadialMetrics_DiscSizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{FileDiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DiscSize).To(Equal(metric.Name("file-size")))
	// Without an explicit Fill, fill metric defaults to disc size.
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(common.Requested.BaseMetrics).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveRadialMetrics_FillOverridesDiscSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		FileDiscSize: &discSizeStr,
		FileFill:     &config.MetricSpec{Metric: "file-type"},
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(common.Requested.BaseMetrics).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveRadialMetrics_DefaultDirectoryFillAggregatesFileFill(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{FileDiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DirectoryFillMetric).To(Equal(metric.Name("file-size.sum")))
	g.Expect(common.Requested.Expressions).To(HaveLen(1))
	g.Expect(common.Requested.Expressions[0].ResultName).To(Equal(metric.Name("file-size.sum")))
}

func TestResolveRadialMetrics_DirectoryDiscSizeAggregatesDiscSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{FileDiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DirectoryDiscSize).To(Equal(metric.Name("file-size.sum")))
	g.Expect(common.Requested.Expressions).To(ContainElement(
		HaveField("ResultName", metric.Name("file-size.sum")),
	))
}

func TestResolveRadialMetrics_DirectoryDiscSizeOverridesFileDiscSize(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	fileDiscSize := "file-size"
	directoryDiscSize := "file-lines.sum"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		FileDiscSize:      &fileDiscSize,
		DirectoryDiscSize: &directoryDiscSize,
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DiscSize).To(Equal(metric.Name("file-size")))
	g.Expect(viz.DirectoryDiscSize).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(common.Requested.Expressions).To(ContainElement(
		HaveField("ResultName", metric.Name("file-lines.sum")),
	))
}

func TestResolveRadialMetrics_ExplicitDirectoryBorder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		FileDiscSize:    &discSizeStr,
		DirectoryBorder: &config.MetricSpec{Metric: "file-type.mode"},
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.DirectoryBorderMetric).To(Equal(metric.Name("file-type.mode")))
	g.Expect(common.Requested.Expressions).To(ContainElement(
		HaveField("ResultName", metric.Name("file-type.mode")),
	))
}

func TestResolveRadialMetrics_LabelsDefaultToFolders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{FileDiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(radialtree.LabelFoldersOnly))
}

func TestResolveRadialMetrics_LabelsNoneExplicit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	labelStr := string(radialtree.LabelNone)
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		FileDiscSize: &discSizeStr,
		Labels:       &labelStr,
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Labels).To(Equal(radialtree.LabelNone))
}

func TestResolveRadialMetrics_GrainDefaultsToFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{FileDiscSize: &discSizeStr}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Grain).To(Equal(radialtree.GrainFile))
}

func TestResolveRadialMetrics_GrainDirectoryExplicit(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	discSizeStr := "file-size"
	grainStr := string(radialtree.GrainDirectory)
	common := &stages.CommonState{}
	viz := &radialtree.State{}
	cfg := &config.Radial{
		FileDiscSize: &discSizeStr,
		Grain:        &grainStr,
	}

	g.Expect(radialtree.ResolveMetrics(common, viz, cfg)).To(Succeed())
	g.Expect(viz.Grain).To(Equal(radialtree.GrainDirectory))
}

func TestBuildLegendStage_GrainDirectoryDescribesDirectoryMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{RootConfig: config.New()}
	viz := &radialtree.State{
		Grain:               radialtree.GrainDirectory,
		FillMetric:          metric.Name("file-lines"),
		DirectoryFillMetric: metric.Name("file-lines.sum"),
	}

	g.Expect(radialtree.BuildLegendStage(common, viz)).To(Succeed())
	g.Expect(viz.LegendConfig).ToNot(BeNil())
	g.Expect(viz.LegendConfig.Entries).To(HaveLen(1))
	g.Expect(viz.LegendConfig.Entries[0].MetricName).To(Equal("file-lines.sum"))
}

func TestBuildLegendStage_GrainFileBuildsLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{RootConfig: config.New()}
	viz := &radialtree.State{
		Grain:      radialtree.GrainFile,
		FillMetric: metric.Name("file-lines"),
	}

	g.Expect(radialtree.BuildLegendStage(common, viz)).To(Succeed())
	g.Expect(viz.LegendConfig).ToNot(BeNil())
}
