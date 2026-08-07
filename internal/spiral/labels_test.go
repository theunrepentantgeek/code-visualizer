package spiral

import (
	"image/color"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestBuildDiscLabel_FormatsDateAndMetricValuesInRoleOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:        time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		SizeValue:    3.5,
		FillLabel:    "go",
		BorderValue:  8,
		SurfaceValue: 1.25,
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Size: "file-size", Fill: "file-type", Border: "file-lines", Surface: "git-age",
	})).To(Equal([]string{"day 7", "Aug", "3.5", "go", "8", "1.25"}))
}

func TestBuildDiscLabel_DeduplicatesMetricRolesAndOmitsMissingCategory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:     time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		SizeValue: 2,
		FillLabel: "",
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Size: "file-size", Fill: "file-size", Border: "file-type", Surface: "file-size",
	})).To(Equal([]string{"day 7", "Aug", "2"}))
}

func TestBuildDiscLabel_RetainsZeroNumericFillAndBorderValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bucket := TimeBucket{
		Start:       time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC),
		FillValue:   0,
		BorderValue: 0,
	}

	g.Expect(buildDiscLabel(bucket, LabelMetrics{
		Fill: "file-lines", Border: "file-size",
	})).To(Equal([]string{"day 7", "Aug", "0", "0", "0"}))
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
		To(Equal([]string{"day 7", "Aug", "3"}))
}

func TestBuildDiscLabels_UsesActiveNodesAndContrastingFillInk(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	buckets := []TimeBucket{
		{Start: time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC), Files: makeFiles(1)},
		{Start: time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC)},
	}
	nodes := []SpiralNode{
		{X: 30, Y: 40, DiscRadius: 12},
		{X: 60, Y: 80, DiscRadius: 0},
	}
	darkFill := inks.FixedInk(color.RGBA{A: 255})

	labels := buildDiscLabels(nodes, buckets, darkFill, LabelMetrics{Size: commitCountMetric})

	g.Expect(labels).To(HaveLen(1))
	g.Expect(labels[0]).To(Equal(canvas.BlockLabel{
		X: 20, Y: 30, W: 20, H: 20,
		Lines: []string{"day 7", "Aug", "1"},
		Ink:   canvas.TextColourFor(color.RGBA{A: 255}),
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
		Lines: []string{"day 7", "Aug", "2"},
	}))
}

func TestBuildLegendStage_SkipsSampleWithoutActiveBuckets(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	viz := &State{Inks: Inks{Fill: inks.FixedInk(color.RGBA{A: 255})}}

	g.Expect(BuildLegendStage(&stages.CommonState{RootConfig: config.New()}, viz)).To(Succeed())
	g.Expect(viz.LegendConfig.LabelSample).To(Equal(legend.LabelSample{}))
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
	g.Expect(viz.DiscLabels[0].X).To(BeNumerically("==", viz.Layout.Nodes[0].X-10))
	g.Expect(viz.DiscLabels[0].Y).To(BeNumerically("==", viz.Layout.Nodes[0].Y-10))
}
