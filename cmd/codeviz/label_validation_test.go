package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

func TestLabelMetricValidation_AcceptsNumericAndClassificationMetrics(t *testing.T) {
	t.Parallel()

	cases := map[string]func() error{
		"treemap": func() error {
			return (&TreemapCmd{}).validateConfig(&config.Treemap{
				Size: new("file-size"), Labels: []metric.Name{"file-lines", "file-type"},
			})
		},
		"spiral": func() error {
			return (&SpiralCmd{}).validateConfig(&config.Spiral{
				Size: new("file-size"), Labels: []metric.Name{"file-lines", "file-type"},
			})
		},
		"donut tree": func() error {
			return (&DonutTreeCmd{}).validateConfig(&config.DonutTree{
				Size: new("file-size"), Labels: []metric.Name{"file-lines", "file-type"},
			})
		},
		"alluvial": func() error {
			return (&AlluvialCmd{}).validateConfig(&config.Alluvial{
				References: []string{"v1", "v2"},
				Metric:     new("file-size"),
				Labels:     []metric.Name{"file-lines", "file-type"},
			})
		},
	}

	for name, validate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			NewGomegaWithT(t).Expect(validate()).To(Succeed())
		})
	}
}

func TestLabelMetricValidation_RejectsUnknownAndRepeatedMetrics(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		labels  []metric.Name
		wantErr string
	}{
		{
			name:    "unknown",
			labels:  []metric.Name{"not-a-real-metric"},
			wantErr: `unknown label metric "not-a-real-metric"; available metrics:`,
		},
		{
			name:    "repeated",
			labels:  []metric.Name{"file-type", "file-type"},
			wantErr: `label metric "file-type" is repeated`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			cfg := &config.Treemap{Size: new("file-size"), Labels: tc.labels}

			err := (&TreemapCmd{}).validateConfig(cfg)

			g.Expect(err).To(MatchError(ContainSubstring(tc.wantErr)))
		})
	}
}

func TestLabelMetricValidation_RejectsSelectedRoleCollisions(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		validate func() error
		label    metric.Name
		role     string
	}{
		"treemap size": {
			validate: func() error {
				return (&TreemapCmd{}).validateConfig(&config.Treemap{
					Size: new("file-lines"), Labels: []metric.Name{"file-lines"},
				})
			},
			label: "file-lines", role: "size",
		},
		"treemap fill": {
			validate: func() error {
				return (&TreemapCmd{}).validateConfig(&config.Treemap{
					Size: new("file-size"), Fill: &config.MetricSpec{Metric: "file-type"},
					Labels: []metric.Name{"file-type"},
				})
			},
			label: "file-type", role: "fill",
		},
		"treemap border": {
			validate: func() error {
				return (&TreemapCmd{}).validateConfig(&config.Treemap{
					Size: new("file-size"), Border: &config.MetricSpec{Metric: "file-age"},
					Labels: []metric.Name{"file-age"},
				})
			},
			label: "file-age", role: "border",
		},
		"spiral default size": {
			validate: func() error {
				return (&SpiralCmd{}).validateConfig(&config.Spiral{
					Labels: []metric.Name{"commit-count"},
				})
			},
			label: "commit-count", role: "size",
		},
		"spiral surface": {
			validate: func() error {
				return (&SpiralCmd{}).validateConfig(&config.Spiral{
					Size:          new("file-size"),
					SurfaceMetric: &config.MetricSpec{Metric: "file-lines"},
					Labels:        []metric.Name{"file-lines"},
				})
			},
			label: "file-lines", role: "surface",
		},
		"donut effective size": {
			validate: func() error {
				return (&DonutTreeCmd{}).validateConfig(&config.DonutTree{
					Size: new("file-lines"), Labels: []metric.Name{"file-lines.sum"},
				})
			},
			label: "file-lines.sum", role: "size",
		},
		"alluvial width": {
			validate: func() error {
				return (&AlluvialCmd{}).validateConfig(&config.Alluvial{
					References: []string{"v1", "v2"}, Metric: new("file-lines"),
					Labels: []metric.Name{"file-lines.sum"},
				})
			},
			label: "file-lines.sum", role: "width",
		},
		"alluvial fill": {
			validate: func() error {
				return (&AlluvialCmd{}).validateConfig(&config.Alluvial{
					References: []string{"v1", "v2"}, Metric: new("file-size"),
					Fill:   &config.MetricSpec{Metric: "file-lines"},
					Labels: []metric.Name{"file-lines.sum"},
				})
			},
			label: "file-lines.sum", role: "fill",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			err := tc.validate()

			g.Expect(err).To(MatchError(And(
				ContainSubstring(`label metric "`+string(tc.label)+`"`),
				ContainSubstring(`selected `+tc.role+` role`),
			)))
		})
	}
}

func TestLabelMetricValidation_RejectsTemporalDirectoryLabels(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		validate func(label metric.Name) error
		label    metric.Name
	}{
		"donut tree delta": {
			validate: func(label metric.Name) error {
				return (&DonutTreeCmd{}).validateConfig(&config.DonutTree{
					Size: new("file-size"), Labels: []metric.Name{label},
				})
			},
			label: "file-lines.delta",
		},
		"donut tree stepdelta": {
			validate: func(label metric.Name) error {
				return (&DonutTreeCmd{}).validateConfig(&config.DonutTree{
					Size: new("file-size"), Labels: []metric.Name{label},
				})
			},
			label: "file-lines.stepdelta",
		},
		"alluvial delta": {
			validate: func(label metric.Name) error {
				return (&AlluvialCmd{}).validateConfig(&config.Alluvial{
					References: []string{"v1", "v2"}, Metric: new("file-size"),
					Labels: []metric.Name{label},
				})
			},
			label: "file-lines.delta",
		},
		"alluvial stepdelta": {
			validate: func(label metric.Name) error {
				return (&AlluvialCmd{}).validateConfig(&config.Alluvial{
					References: []string{"v1", "v2"}, Metric: new("file-size"),
					Labels: []metric.Name{label},
				})
			},
			label: "file-lines.stepdelta",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			err := tc.validate(tc.label)

			g.Expect(err).To(MatchError(And(
				ContainSubstring(`invalid label metric "`+string(tc.label)+`"`),
				ContainSubstring("temporal modifier"),
				ContainSubstring("not supported for label metrics"),
			)))
		})
	}
}

func TestLabelMetricValidation_TemporalAlluvialFillStillCollidesWithLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := (&AlluvialCmd{}).validateConfig(&config.Alluvial{
		References: []string{"v1", "v2"}, Metric: new("file-size"),
		Fill:   &config.MetricSpec{Metric: "file-lines.delta"},
		Labels: []metric.Name{"file-lines.sum"},
	})

	g.Expect(err).To(MatchError(And(
		ContainSubstring(`label metric "file-lines.sum"`),
		ContainSubstring("selected fill role"),
	)))
}
