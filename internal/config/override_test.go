package config

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// Config.OverrideWidth / OverrideHeight

func TestConfig_OverrideWidth_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	cfg.OverrideWidth(2560)
	g.Expect(cfg.ImageSize).NotTo(BeNil())
	g.Expect(*cfg.ImageSize.Width).To(Equal(2560))
}

func TestConfig_OverrideWidth_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	original := *cfg.ImageSize.Width
	cfg.OverrideWidth(0)
	g.Expect(*cfg.ImageSize.Width).To(Equal(original))
}

func TestConfig_OverrideHeight_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	cfg.OverrideHeight(1440)
	g.Expect(cfg.ImageSize).NotTo(BeNil())
	g.Expect(*cfg.ImageSize.Height).To(Equal(1440))
}

func TestConfig_OverrideHeight_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := New()
	original := *cfg.ImageSize.Height
	cfg.OverrideHeight(0)
	g.Expect(*cfg.ImageSize.Height).To(Equal(original))
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

func TestConfig_OverrideLegendPosition_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := &Config{}
	cfg.OverrideLegendPosition("top-right")
	g.Expect(*cfg.Legend.Position).To(Equal("top-right"))
}

func TestConfig_OverrideLegendPosition_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := &Config{}
	cfg.OverrideLegendPosition("")
	g.Expect(cfg.Legend).To(BeNil())
}

func TestConfig_OverrideLegendOrientation_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	cfg := &Config{}
	cfg.OverrideLegendOrientation("horizontal")
	g.Expect(*cfg.Legend.Orientation).To(Equal("horizontal"))
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

func TestBubbletree_OverrideFill_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	b := &Bubbletree{}
	spec := MetricSpec{Metric: metric.Name("file-lines"), Palette: palette.PaletteName("foliage")}
	b.OverrideFill(spec)
	g.Expect(*b.Fill).To(Equal(spec))
}

func TestBubbletree_OverrideFill_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	b := &Bubbletree{Fill: &existing}
	b.OverrideFill(MetricSpec{})
	g.Expect(*b.Fill).To(Equal(existing))
}

func TestBubbletree_OverrideBorder_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	b := &Bubbletree{}
	spec := MetricSpec{Metric: metric.Name("commit-count"), Palette: palette.PaletteName("fire")}
	b.OverrideBorder(spec)
	g.Expect(*b.Border).To(Equal(spec))
}

func TestBubbletree_OverrideBorder_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	b := &Bubbletree{Border: &existing}
	b.OverrideBorder(MetricSpec{})
	g.Expect(*b.Border).To(Equal(existing))
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

func TestSpiral_OverrideFill_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Spiral{}
	spec := MetricSpec{Metric: metric.Name("file-lines"), Palette: palette.PaletteName("foliage")}
	s.OverrideFill(spec)
	g.Expect(*s.Fill).To(Equal(spec))
}

func TestSpiral_OverrideFill_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	s := &Spiral{Fill: &existing}
	s.OverrideFill(MetricSpec{})
	g.Expect(*s.Fill).To(Equal(existing))
}

func TestSpiral_OverrideBorder_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Spiral{}
	spec := MetricSpec{Metric: metric.Name("commit-count"), Palette: palette.PaletteName("fire")}
	s.OverrideBorder(spec)
	g.Expect(*s.Border).To(Equal(spec))
}

func TestSpiral_OverrideBorder_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	s := &Spiral{Border: &existing}
	s.OverrideBorder(MetricSpec{})
	g.Expect(*s.Border).To(Equal(existing))
}

// Radial overrides (fill/border)

func TestRadial_OverrideFill_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	r := &Radial{}
	spec := MetricSpec{Metric: metric.Name("file-lines"), Palette: palette.PaletteName("foliage")}
	r.OverrideFill(spec)
	g.Expect(*r.Fill).To(Equal(spec))
}

func TestRadial_OverrideFill_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	r := &Radial{Fill: &existing}
	r.OverrideFill(MetricSpec{})
	g.Expect(*r.Fill).To(Equal(existing))
}

func TestRadial_OverrideBorder_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	r := &Radial{}
	spec := MetricSpec{Metric: metric.Name("commit-count"), Palette: palette.PaletteName("fire")}
	r.OverrideBorder(spec)
	g.Expect(*r.Border).To(Equal(spec))
}

func TestRadial_OverrideBorder_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	r := &Radial{Border: &existing}
	r.OverrideBorder(MetricSpec{})
	g.Expect(*r.Border).To(Equal(existing))
}

// Treemap override (border)

func TestTreemap_OverrideBorder_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	tm := &Treemap{}
	spec := MetricSpec{Metric: metric.Name("commit-count"), Palette: palette.PaletteName("fire")}
	tm.OverrideBorder(spec)
	g.Expect(*tm.Border).To(Equal(spec))
}

func TestTreemap_OverrideBorder_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	tm := &Treemap{Border: &existing}
	tm.OverrideBorder(MetricSpec{})
	g.Expect(*tm.Border).To(Equal(existing))
}

// Scatter overrides

func TestScatter_OverrideXAxis_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}
	s.OverrideXAxis("file-lines")
	g.Expect(*s.XAxis).To(Equal("file-lines"))
}

func TestScatter_OverrideXAxis_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "file-age"
	s := &Scatter{XAxis: &existing}
	s.OverrideXAxis("")
	g.Expect(*s.XAxis).To(Equal("file-age"))
}

func TestScatter_OverrideYAxis_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}
	s.OverrideYAxis("commit-count")
	g.Expect(*s.YAxis).To(Equal("commit-count"))
}

func TestScatter_OverrideYAxis_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "file-lines"
	s := &Scatter{YAxis: &existing}
	s.OverrideYAxis("")
	g.Expect(*s.YAxis).To(Equal("file-lines"))
}

func TestScatter_OverrideSize_SetsWhenNonEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}
	s.OverrideSize("file-lines")
	g.Expect(*s.Size).To(Equal("file-lines"))
}

func TestScatter_OverrideSize_SkipsWhenEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := "commit-count"
	s := &Scatter{Size: &existing}
	s.OverrideSize("")
	g.Expect(*s.Size).To(Equal("commit-count"))
}

func TestScatter_OverrideFill_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}
	spec := MetricSpec{Metric: metric.Name("file-lines"), Palette: palette.PaletteName("foliage")}
	s.OverrideFill(spec)
	g.Expect(*s.Fill).To(Equal(spec))
}

func TestScatter_OverrideFill_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	s := &Scatter{Fill: &existing}
	s.OverrideFill(MetricSpec{})
	g.Expect(*s.Fill).To(Equal(existing))
}

func TestScatter_OverrideBorder_SetsWhenNonZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	s := &Scatter{}
	spec := MetricSpec{Metric: metric.Name("commit-count"), Palette: palette.PaletteName("fire")}
	s.OverrideBorder(spec)
	g.Expect(*s.Border).To(Equal(spec))
}

func TestScatter_OverrideBorder_SkipsWhenZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	existing := MetricSpec{Metric: metric.Name("file-age")}
	s := &Scatter{Border: &existing}
	s.OverrideBorder(MetricSpec{})
	g.Expect(*s.Border).To(Equal(existing))
}
