package config

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/palette"
)

// Config.OverrideWidth / OverrideHeight

func TestConfig_OverrideWidth_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	cfg.OverrideWidth(2560)
	g.Expect(*cfg.Width).To(Equal(2560))
}

func TestConfig_OverrideWidth_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	original := *cfg.Width
	cfg.OverrideWidth(0)
	g.Expect(*cfg.Width).To(Equal(original))
}

func TestConfig_OverrideHeight_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	cfg.OverrideHeight(1440)
	g.Expect(*cfg.Height).To(Equal(1440))
}

// Treemap overrides

func TestTreemap_OverrideSize_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tm := &Treemap{}
	tm.OverrideSize("large")
	g.Expect(*tm.Size).To(Equal("large"))
}

func TestTreemap_OverrideSize_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "medium"
	tm := &Treemap{Size: &existing}
	tm.OverrideSize("")
	g.Expect(*tm.Size).To(Equal("medium"))
}

func TestTreemap_OverrideFill_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tm := &Treemap{}
	spec := MetricSpec{Metric: metric.Name("file-lines"), Palette: palette.PaletteName("foliage")}
	tm.OverrideFill(spec)
	g.Expect(*tm.Fill).To(Equal(spec))
}

func TestTreemap_OverrideFill_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	tm := &Treemap{Fill: &existing}
	tm.OverrideFill(MetricSpec{})
	g.Expect(*tm.Fill).To(Equal(existing))
}

func TestTreemap_OverrideLegend_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tm := &Treemap{}
	tm.OverrideLegend("top-right")
	g.Expect(*tm.Legend).To(Equal("top-right"))
}

func TestTreemap_OverrideLegendOrientation_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tm := &Treemap{}
	tm.OverrideLegendOrientation("horizontal")
	g.Expect(*tm.LegendOrientation).To(Equal("horizontal"))
}

// Radial overrides

func TestRadial_OverrideDiscSize_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	r := &Radial{}
	r.OverrideDiscSize("commit-count")
	g.Expect(*r.DiscSize).To(Equal("commit-count"))
}

func TestRadial_OverrideLabels_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	r := &Radial{}
	r.OverrideLabels("all")
	g.Expect(*r.Labels).To(Equal("all"))
}

// Bubbletree overrides

func TestBubbletree_OverrideSize_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	b := &Bubbletree{}
	b.OverrideSize("file-lines")
	g.Expect(*b.Size).To(Equal("file-lines"))
}

func TestBubbletree_OverrideLabels_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	b := &Bubbletree{}
	b.OverrideLabels("folders")
	g.Expect(*b.Labels).To(Equal("folders"))
}

// Spiral overrides

func TestSpiral_OverrideResolution_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Spiral{}
	s.OverrideResolution("hourly")
	g.Expect(*s.Resolution).To(Equal("hourly"))
}

func TestSpiral_OverrideResolution_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "daily"
	s := &Spiral{Resolution: &existing}
	s.OverrideResolution("")
	g.Expect(*s.Resolution).To(Equal("daily"))
}

func TestSpiral_OverrideSize_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Spiral{}
	s.OverrideSize("file-age")
	g.Expect(*s.Size).To(Equal("file-age"))
}

func TestSpiral_OverrideLabels_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Spiral{}
	s.OverrideLabels("none")
	g.Expect(*s.Labels).To(Equal("none"))
}
