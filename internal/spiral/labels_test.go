package spiral

import (
	"image/color"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestBuildDiscLabel_FormatsDateAndMetricValuesInRoleOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:                 time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		SizeValue:             3.5,
		SizeValueAvailable:    true,
		FillLabel:             "go",
		BorderValue:           8,
		BorderValueAvailable:  true,
		SurfaceValue:          1.25,
		SurfaceValueAvailable: true,
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Size: "file-size", Fill: "file-type", Border: "file-lines", Surface: "git-age",
	})).To(Equal([]string{"7", "Aug", "3.5", "go", "8", "1.25"}))
}

func TestBuildDiscLabel_DeduplicatesMetricRolesAndOmitsMissingCategory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:              time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		SizeValue:          2,
		SizeValueAvailable: true,
		FillLabel:          "",
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Size: "file-size", Fill: "file-size", Border: "file-type", Surface: "file-size",
	})).To(Equal([]string{"7", "Aug", "2"}))
}

func TestBuildDiscLabel_RetainsZeroNumericFillAndBorderValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:                 time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		FillValue:             0,
		FillValueAvailable:    true,
		BorderValue:           0,
		BorderValueAvailable:  true,
		SurfaceValueAvailable: true,
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Fill: "file-lines", Border: "file-size",
	})).To(Equal([]string{"7", "Aug", "0", "0", "0"}))
}

func TestBuildDiscLabel_OmitsUnavailableNumericMetric(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := []TimeBucket{{
		Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		Files: makeFiles(1),
	}}
	requested := stages.ClassifyRequestedMetrics(
		[]metric.Name{commitCountMetric, "file-lines"},
		metric.LevelDirectory,
	)
	AggregateBucketMetrics(buckets, requested, commitCountMetric, "file-lines", "", "")

	g.Expect(buildDiscLabel(buckets[0], LabelMetrics{
		Size: commitCountMetric, Fill: "file-lines", Requested: requested,
	})).To(Equal([]string{"7", "Aug", "1"}))
}

func TestBuildDiscLabel_DefaultSizeUsesCommitCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		Files: makeFiles(3),
	}

	g.Expect(effectiveSizeMetric("")).To(Equal(commitCountMetric))
	g.Expect(buildDiscLabel(bucket, LabelMetrics{})).
		To(Equal([]string{"7", "Aug", "3"}))
}

func TestBuildDiscLabels_UsesActiveNodesAndContrastingFillInk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := []TimeBucket{
		{Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC), Files: makeFiles(1)},
		{Start: time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)},
	}
	nodes := []SpiralNode{
		{Position: geometry.Point{X: 30, Y: 40}, DiscRadius: 12},
		{Position: geometry.Point{X: 60, Y: 80}, DiscRadius: 0},
	}
	darkFill := inks.FixedInk(color.RGBA{A: 255})

	labels := buildDiscLabels(nodes, buckets, darkFill, LabelMetrics{Size: commitCountMetric})

	g.Expect(labels).To(HaveLen(1))
	g.Expect(labels[0]).To(Equal(canvas.BlockLabel{
		X: 20, Y: 30, W: 20, H: 20,
		Lines:        []string{"7", "Aug", "1"},
		Ink:          canvas.TextColourFor(color.RGBA{A: 255}),
		PreserveText: true,
	}))
}

func TestBuildDiscLabels_PreservesPrePointOffsetGrouping(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	nodes := []SpiralNode{{
		Position:   geometry.Point{X: -2.3, Y: -0.1},
		DiscRadius: 2.1,
	}}
	buckets := []TimeBucket{{
		Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		Files: makeFiles(1),
	}}

	labels := buildDiscLabels(
		nodes,
		buckets,
		inks.FixedInk(color.RGBA{A: 255}),
		LabelMetrics{Size: commitCountMetric},
	)

	g.Expect(labels).To(HaveLen(1))
	g.Expect(labels[0].X).To(Equal(-2.4000000000000004))
	g.Expect(labels[0].Y).To(Equal(-0.20000000000000018))
}

func TestBuildDiscLabels_UsesOnlyPairedNodesAndBuckets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	nodes := []SpiralNode{
		{Position: geometry.Point{X: 30, Y: 40}, DiscRadius: 12},
		{Position: geometry.Point{X: 60, Y: 80}, DiscRadius: 12},
	}
	buckets := []TimeBucket{{
		Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		Files: makeFiles(1),
	}}

	labels := buildDiscLabels(nodes, buckets, inks.FixedInk(color.RGBA{A: 255}), LabelMetrics{
		Size: commitCountMetric,
	})

	g.Expect(labels).To(HaveLen(1))
	g.Expect(labels[0].Lines).To(Equal([]string{"7", "Aug", "1"}))
}

func TestRenderToCanvas_PreservesZeroMetricTextInRasterDiscLabels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := []TimeBucket{{
		Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
	}}
	nodes := []SpiralNode{{Position: geometry.Point{X: 25, Y: 25}, DiscRadius: 12}}
	ink := inks.FixedInk(color.RGBA{A: 255})
	labels := buildDiscLabels(nodes, buckets, ink, LabelMetrics{
		Size: commitCountMetric, Fill: "file-lines", Border: "file-size", Surface: "git-age",
		Requested: stages.ClassifyRequestedMetrics(
			[]metric.Name{"file-lines", "file-size"}, metric.LevelDirectory,
		),
	})
	cv := RenderToCanvas(
		SpiralLayout{Nodes: nodes}, buckets, 50, 50, Inks{Fill: ink, Border: ink}, RenderOptions{
			DiscLabels: labels,
			Format:     canvas.FormatPNG,
		},
	)

	mb := mock.NewBackend()
	g.Expect(cv.RenderTo(mb)).To(Succeed())

	hasZeroText := false
	hasLine := false

	for _, call := range mb.Calls {
		hasZeroText = hasZeroText || (call.Method == "DrawText" && call.Text == "0")
		hasLine = hasLine || call.Method == "DrawLine"
	}

	g.Expect(hasZeroText).To(BeTrue())
	g.Expect(hasLine).To(BeFalse())
}

func TestBuildLegendLabelSample_UsesDefaultSizeMetric(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(buildLegendLabelSample(LabelMetrics{})).To(Equal([]string{
		"Day", "Month", "commit-count",
	}))
}

func TestBuildLegendLabelSample_DeduplicatesMetricRoles(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(buildLegendLabelSample(LabelMetrics{
		Size: "file-size", Fill: "file-size", Border: "file-lines", Surface: "file-size",
	})).To(Equal([]string{
		"Day", "Month", "file-size", "file-lines",
	}))
}

func TestBuildLegendLabelSample_IncludesDistinctSurfaceMetric(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(buildLegendLabelSample(LabelMetrics{
		Size: "file-size", Fill: "file-lines", Border: "file-size", Surface: "git-age",
	})).To(Equal([]string{
		"Day", "Month", "file-size", "file-lines", "git-age",
	}))
}

func TestBuildLegendStage_UsesCircleSampleAndDefaultCommitCountSize(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	viz := &State{
		Buckets: []TimeBucket{{
			Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
			Files: makeFiles(2),
		}},
		Inks: Inks{Fill: inks.FixedInk(color.RGBA{A: 255})},
	}

	g.Expect(BuildLegendStage(&stages.CommonState{RootConfig: config.New()}, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.Entries).To(ContainElement(legend.Entry{
		Role: legend.RoleSize, MetricName: string(commitCountMetric),
		Ink: inks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
	}))
	g.Expect(viz.LegendConfig.LabelSample).To(Equal(legend.LabelSample{
		Shape: legend.LabelSampleCircle,
		Lines: []string{"Day", "Month", "commit-count"},
	}))
}

func TestBuildLegendStage_SetsSampleWithoutActiveBuckets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	viz := &State{Inks: Inks{Fill: inks.FixedInk(color.RGBA{A: 255})}}

	g.Expect(BuildLegendStage(&stages.CommonState{RootConfig: config.New()}, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.LabelSample).To(Equal(legend.LabelSample{
		Shape: legend.LabelSampleCircle,
		Lines: []string{"Day", "Month", "commit-count"},
	}))
}

func TestLayoutStage_StoresDiscLabelsAfterVerticalOffset(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	viz := &State{
		Buckets: []TimeBucket{{
			Start:     time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
			Files:     makeFiles(1),
			SizeValue: 1,
		}},
		Inks: Inks{Fill: inks.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255})},
	}
	common := &stages.CommonState{
		Width: 200,
		DrawingBounds: stages.DrawingBounds{
			MinY: 25,
			MaxY: 200,
		},
	}

	g.Expect(LayoutStage(common, viz)).To(Succeed())
	g.Expect(viz.DiscLabels).To(HaveLen(1))
	g.Expect(viz.DiscLabels[0].X).To(
		BeNumerically("==", viz.Layout.Nodes[0].Position.X-viz.Layout.Nodes[0].DiscRadius+discLabelPadding),
	)
	g.Expect(viz.DiscLabels[0].Y).To(
		BeNumerically("==", viz.Layout.Nodes[0].Position.Y-viz.Layout.Nodes[0].DiscRadius+discLabelPadding),
	)
}
