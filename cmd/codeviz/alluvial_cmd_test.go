package main

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/alecthomas/kong"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

func TestAlluvialCmd_Run_ReportsUnavailablePipeline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := (&AlluvialCmd{
		Output:     "out.png",
		References: []string{"tag:v1.0", "tag:v2.0"},
		Metric:     "file-size",
	}).Run(&Flags{Config: config.New()})

	g.Expect(err).To(MatchError(ContainSubstring("alluvial pipeline is not implemented")))
}

func TestCLI_ParsesAlluvialOrderedInputs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cli := CLI{}
	parser, err := kong.New(&cli, kong.Name("codeviz"), filterMapperOption(), kong.Exit(func(int) {}))
	g.Expect(err).NotTo(HaveOccurred())

	_, err = parser.Parse([]string{
		"alluvial", ".", "-o", "out.svg",
		"--reference", "tag:v1.0",
		"--reference", "sha:abc1234",
		"--reference", "2026-01-01",
		"--expand", "cmd",
		"--expand", "internal/config",
		"--include", "**/*.go",
		"--exclude", "**/*_test.go",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Alluvial.References).To(Equal([]string{"tag:v1.0", "sha:abc1234", "2026-01-01"}))
	g.Expect(cli.Alluvial.Expand).To(Equal([]string{"cmd", "internal/config"}))
	rules := cli.Alluvial.Filters()
	g.Expect(rules).To(HaveLen(2))
	g.Expect(rules[0].Pattern).To(Equal("**/*.go"))
	g.Expect(rules[0].Mode).To(Equal(filter.Include))
	g.Expect(rules[1].Pattern).To(Equal("**/*_test.go"))
	g.Expect(rules[1].Mode).To(Equal(filter.Exclude))
}

func TestAlluvialCmd_MergeConfig_ReplacesConfiguredReferences(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.Alluvial.References = []string{"tag:v1.0", "tag:v1.1"}
	cfg.Alluvial.Expand = []string{"configured"}
	cmd := &AlluvialCmd{
		Output:     "out.png",
		References: []string{"tag:v2.0", "date:2026-01-01"},
		Expand:     []string{"cmd", "internal/config"},
	}

	cmd.applyOverrides(cfg)

	g.Expect(cfg.Alluvial.References).To(Equal([]string{"tag:v2.0", "date:2026-01-01"}))
	g.Expect(cfg.Alluvial.Expand).To(Equal([]string{"cmd", "internal/config"}))
}

func TestAlluvialCmd_ValidateConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     *config.Alluvial
		wantErr string
	}{
		{
			name:    "requires two references",
			cfg:     &config.Alluvial{References: []string{"tag:v1.0"}, Metric: new("file-size")},
			wantErr: "at least two references",
		},
		{
			name:    "requires a metric",
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v2.0"}},
			wantErr: "metric is required",
		},
		{
			name:    "rejects unknown metric",
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v2.0"}, Metric: new("not-a-metric")},
			wantErr: "unknown metric metric",
		},
		{
			name:    "rejects parent expansion",
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v2.0"}, Metric: new("file-size"), Expand: []string{"../outside"}},
			wantErr: "invalid expansion path",
		},
		{
			name: "accepts tags and supported references",
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "sha:abc1234", "date:2026-01-01"},
				Metric:     new("file-size"),
				Expand:     []string{".", "internal/config"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			err := (&AlluvialCmd{}).validateConfig(tc.cfg)

			if tc.wantErr == "" {
				g.Expect(err).NotTo(HaveOccurred())
				return
			}

			g.Expect(err).To(MatchError(ContainSubstring(tc.wantErr)))
		})
	}
}
