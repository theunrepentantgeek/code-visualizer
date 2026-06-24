package main

// Validation-matrix tests: every visualization type × every metric field × every metric kind.
//
// Coverage that already exists in main_test.go and metric_validation_test.go is not
// duplicated here; this file fills the gaps (BubbletreeCmd, RadialCmd, ScatterCmd
// fill/border, and the full metric-kind sweep for SpiralCmd and TreemapCmd border).
//
// Registered providers (filesystem, git, golang) are set up in TestMain (main_test.go).

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// sizeCase is one row of the numeric-size validation matrix.
// Size / disc-size fields must be numeric (quantity or measure) and reject
// classification metrics and unknown names.
type sizeCase struct {
	metric  string
	wantOK  bool
	comment string
}

var sizeCases = []sizeCase{
	{metric: "file-size", wantOK: true, comment: "quantity metric"},
	{metric: "comment-ratio", wantOK: true, comment: "measure metric"},
	{metric: "file-type", wantOK: false, comment: "classification metric"},
	{metric: "no-such-metric", wantOK: false, comment: "unknown metric"},
}

// fillBorderCase is one row of the fill/border MetricSpec validation matrix.
// Fill and border accept any metric kind plus aggregation expressions; they
// reject unknown metrics and invalid palette names.
type fillBorderCase struct {
	metric  string
	palette string
	wantOK  bool
	comment string
}

var fillBorderCases = []fillBorderCase{
	{metric: "file-size", wantOK: true, comment: "quantity, no palette"},
	{metric: "comment-ratio", wantOK: true, comment: "measure, no palette"},
	{metric: "file-type", wantOK: true, comment: "classification, no palette"},
	{metric: "declarations.count", wantOK: true, comment: "aggregation expression"},
	{metric: "file-size", palette: "temperature", wantOK: true, comment: "quantity + valid palette"},
	{metric: "file-size", palette: "not-a-palette", wantOK: false, comment: "invalid palette"},
	{metric: "no-such-metric", wantOK: false, comment: "unknown metric"},
}

// axisCase is one row of the scatter-axis validation matrix.
// Scatter axes accept any metric kind (numeric and classification), since the
// viz engine handles classification axes; only unknown names are rejected.
type axisCase struct {
	metric  string
	wantOK  bool
	comment string
}

var axisCases = []axisCase{
	{metric: "file-size", wantOK: true, comment: "quantity metric"},
	{metric: "comment-ratio", wantOK: true, comment: "measure metric"},
	{metric: "file-type", wantOK: true, comment: "classification metric"},
	{metric: "no-such-metric", wantOK: false, comment: "unknown metric"},
}

// ─── BubbletreeCmd ──────────────────────────────────────────────────────────

func TestBubbletreeCmd_ValidateConfig_SizeMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range sizeCases {
		t.Run(tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Bubbletree.Size = new(tc.metric)

			err := (&BubbletreeCmd{}).validateConfig(cfg.Bubbletree)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestBubbletreeCmd_ValidateConfig_FillMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("fill/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Bubbletree.Size = new("file-size")
			cfg.Bubbletree.Fill = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&BubbletreeCmd{}).validateConfig(cfg.Bubbletree)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestBubbletreeCmd_ValidateConfig_BorderMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("border/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Bubbletree.Size = new("file-size")
			cfg.Bubbletree.Border = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&BubbletreeCmd{}).validateConfig(cfg.Bubbletree)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

// ─── RadialCmd (disc-size) ────────────────────────────────────────────────────

func TestRadialCmd_ValidateConfig_DiscSizeMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range sizeCases {
		t.Run(tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Radial.DiscSize = new(tc.metric)

			err := (&RadialCmd{}).validateConfig(cfg.Radial)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestRadialCmd_ValidateConfig_FillMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("fill/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Radial.DiscSize = new("file-size")
			cfg.Radial.Fill = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&RadialCmd{}).validateConfig(cfg.Radial)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestRadialCmd_ValidateConfig_BorderMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("border/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Radial.DiscSize = new("file-size")
			cfg.Radial.Border = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&RadialCmd{}).validateConfig(cfg.Radial)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

// ─── SpiralCmd border ─────────────────────────────────────────────────────────

func TestSpiralCmd_ValidateConfig_BorderMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("border/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Spiral.Size = new("file-size")
			cfg.Spiral.Border = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&SpiralCmd{}).validateConfig(cfg.Spiral)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

// ─── TreemapCmd border ────────────────────────────────────────────────────────

func TestTreemapCmd_ValidateConfig_BorderMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("border/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Treemap.Size = new("file-size")
			cfg.Treemap.Border = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&TreemapCmd{}).validateConfig(cfg.Treemap)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

// ─── ScatterCmd fill/border ───────────────────────────────────────────────────

func TestScatterCmd_ValidateConfig_FillMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("fill/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Scatter.XAxis = new("file-size")
			cfg.Scatter.YAxis = new("file-lines")
			cfg.Scatter.Size = new("file-size")
			cfg.Scatter.Fill = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&ScatterCmd{}).validateConfig(cfg.Scatter)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestScatterCmd_ValidateConfig_BorderMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range fillBorderCases {
		t.Run("border/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Scatter.XAxis = new("file-size")
			cfg.Scatter.YAxis = new("file-lines")
			cfg.Scatter.Size = new("file-size")
			cfg.Scatter.Border = &config.MetricSpec{
				Metric:  metric.Name(tc.metric),
				Palette: palette.PaletteName(tc.palette),
			}

			err := (&ScatterCmd{}).validateConfig(cfg.Scatter)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

// ─── ScatterCmd axes ──────────────────────────────────────────────────────────

func TestScatterCmd_ValidateConfig_XAxisMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range axisCases {
		t.Run("x-axis/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Scatter.XAxis = new(tc.metric)
			cfg.Scatter.YAxis = new("file-lines")
			cfg.Scatter.Size = new("file-size")

			err := (&ScatterCmd{}).validateConfig(cfg.Scatter)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}

func TestScatterCmd_ValidateConfig_YAxisMatrix(t *testing.T) {
	t.Parallel()

	for _, tc := range axisCases {
		t.Run("y-axis/"+tc.comment, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			cfg := config.New()
			cfg.Scatter.XAxis = new("file-size")
			cfg.Scatter.YAxis = new(tc.metric)
			cfg.Scatter.Size = new("file-size")

			err := (&ScatterCmd{}).validateConfig(cfg.Scatter)
			if tc.wantOK {
				g.Expect(err).NotTo(HaveOccurred())
			} else {
				g.Expect(err).To(HaveOccurred())
			}
		})
	}
}
