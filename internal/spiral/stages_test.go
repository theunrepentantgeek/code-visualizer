package spiral_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &spiral.State{
		Config: &config.Spiral{Size: &sizeStr},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("file-size")))
	// Spiral does not fall back FillMetric to Size; without an explicit Fill
	// the spiral renders without a fill metric.
	g.Expect(s.FillMetric).To(Equal(metric.Name("")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_NilSizeExcludesSizeFromRequested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// When Size is nil the spiral defaults to commit-count.
	// Only fill and border contribute to Requested.
	s := &spiral.State{
		Config: &config.Spiral{
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-type")))
}

func TestResolveMetrics_FillMetricSetWhenFillConfigured(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &spiral.State{
		Config: &config.Spiral{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &spiral.State{
		Config: &config.Spiral{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestResolveMetrics_DefaultsResolutionToDaily(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &spiral.State{
		Config: &config.Spiral{Size: &sizeStr},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Resolution).To(Equal(spiral.Daily))
}

func TestResolveMetrics_HourlyResolution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	res := "hourly"
	s := &spiral.State{
		Config: &config.Spiral{Size: &sizeStr, Resolution: &res},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Resolution).To(Equal(spiral.Hourly))
}

func TestResolveMetrics_DefaultsLabelsToLaps(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &spiral.State{
		Config: &config.Spiral{Size: &sizeStr},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(spiral.LabelLaps))
}

func TestResolveMetrics_LabelsAllCanBeSet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	lblStr := "all"
	s := &spiral.State{
		Config: &config.Spiral{Size: &sizeStr, Labels: &lblStr},
	}

	g.Expect(spiral.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Labels).To(Equal(spiral.LabelAll))
}

func TestState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &spiral.State{}
	c := s.Common()
	c.Width = 42
	g.Expect(s.CommonState.Width).To(Equal(42))
}

func TestState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &spiral.State{IncludeBinaryFiles: true}
	off := &spiral.State{IncludeBinaryFiles: false}

	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	var _ stages.BinaryFilterToggler = on
}
