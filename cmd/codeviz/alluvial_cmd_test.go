package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	gogit "github.com/go-git/go-git/v5"

	"github.com/alecthomas/kong"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestAlluvialCmd_Run_EntersDataPipeline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := (&AlluvialCmd{
		Output:     "out.png",
		References: []string{"tag:v1.0", "tag:v2.0"},
		Metric:     "file-size",
	}).Run(&Flags{Config: config.New()})

	g.Expect(err).To(MatchError(ContainSubstring("alluvial pipeline failed")))
}

func TestBuildAlluvialPhasesOrdersOneLiveStagePerReference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cfg := &config.Alluvial{References: []string{"v1", "v2", ""}}

	phases := buildAlluvialPhases(&stages.CommonState{}, &alluvial.State{}, cfg)

	g.Expect(phases).To(HaveLen(6))
	g.Expect([]string{
		phases[0].Name,
		phases[1].Name,
		phases[2].Name,
		phases[3].Name,
		phases[4].Name,
		phases[5].Name,
	}).To(Equal([]string{
		"Preparing", "Loading v1", "Loading v2", "Loading HEAD", "Rendering", "Writing output",
	}))
	g.Expect(phases[0].Kind).To(Equal(progress.StageSummary))
	g.Expect(phases[1].Kind).To(Equal(progress.StageLive))
	g.Expect(phases[2].Kind).To(Equal(progress.StageLive))
	g.Expect(phases[3].Kind).To(Equal(progress.StageLive))
	g.Expect(phases[4].Kind).To(Equal(progress.StageSummary))
	g.Expect(phases[5].Kind).To(Equal(progress.StageSummary))
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
		"--fill", "file-lines.delta,temperature",
		"--include", "**/*.go",
		"--exclude", "**/*_test.go",
	})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cli.Alluvial.References).To(Equal([]string{"tag:v1.0", "sha:abc1234", "2026-01-01"}))
	g.Expect(cli.Alluvial.Expand).To(Equal([]string{"cmd", "internal/config"}))
	g.Expect(cli.Alluvial.Fill).To(Equal(config.MetricSpec{Metric: "file-lines.delta", Palette: "temperature"}))
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
		Fill:       config.MetricSpec{Metric: "file-lines.delta", Palette: "temperature"},
		Expand:     []string{"cmd", "internal/config"},
	}

	cmd.applyOverrides(cfg)

	g.Expect(cfg.Alluvial.References).To(Equal([]string{"tag:v2.0", "date:2026-01-01"}))
	g.Expect(*cfg.Alluvial.Fill).To(Equal(config.MetricSpec{Metric: "file-lines.delta", Palette: "temperature"}))
	g.Expect(cfg.Alluvial.Expand).To(Equal([]string{"cmd", "internal/config"}))
}

func TestAlluvialCmd_ValidateConfig(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cfg     *config.Alluvial
		wantErr string
	}{
		"requires two references": {
			cfg:     &config.Alluvial{References: []string{"tag:v1.0"}, Metric: new("file-size")},
			wantErr: "at least two references",
		},
		"requires a metric": {
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v2.0"}},
			wantErr: "metric is required",
		},
		"rejects duplicate references": {
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v1.0"}, Metric: new("file-size")},
			wantErr: "references must be unique",
		},
		"rejects unknown metric": {
			cfg:     &config.Alluvial{References: []string{"tag:v1.0", "tag:v2.0"}, Metric: new("not-a-metric")},
			wantErr: "unknown metric",
		},
		"accepts delta fill metric": {
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "tag:v2.0"},
				Metric:     new("file-size"),
				Fill:       &config.MetricSpec{Metric: "file-lines.delta", Palette: "temperature"},
			},
		},
		"accepts palette with default fill metric": {
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "tag:v2.0"},
				Metric:     new("file-size"),
				Fill:       &config.MetricSpec{Palette: "temperature"},
			},
		},
		"rejects unknown delta fill metric": {
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "tag:v2.0"},
				Metric:     new("file-size"),
				Fill:       &config.MetricSpec{Metric: "not-a-metric.delta"},
			},
			wantErr: "unknown fill metric",
		},
		"rejects parent expansion": {
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "tag:v2.0"},
				Metric:     new("file-size"),
				Expand:     []string{"../outside"},
			},
			wantErr: "invalid expansion path",
		},
		"accepts tags and supported references": {
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "sha:abc1234", "date:2026-01-01"},
				Metric:     new("file-size"),
				Expand:     []string{".", "internal/config"},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			err := (&AlluvialCmd{}).validateConfig(c.cfg)

			if c.wantErr == "" {
				g.Expect(err).NotTo(HaveOccurred())

				return
			}

			g.Expect(err).To(MatchError(ContainSubstring(c.wantErr)))
		})
	}
}

func TestAlluvialCmd_Run_ResolvesTaggedSnapshots(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repository := createAlluvialTagFixture(t)
	output := filepath.Join(t.TempDir(), "alluvial.svg")
	cmd := &AlluvialCmd{
		TargetPath: repository,
		Output:     output,
		References: []string{"tag:v1.0", "tag:v2.0"},
		Metric:     "file-lines",
		Width:      320,
		Height:     240,
		Footer:     "fixture footer",
	}

	g.Expect(cmd.Run(&Flags{Config: config.New()})).To(Succeed())

	image, err := os.ReadFile(output)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(image)).To(ContainSubstring("tag:v1.0"))
	g.Expect(string(image)).To(ContainSubstring("tag:v2.0"))
}

func TestAlluvialCmd_Run_ComputesGitFillMetricForTaggedSnapshots(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repository := createAlluvialTagFixture(t)
	output := filepath.Join(t.TempDir(), "alluvial.svg")
	cmd := &AlluvialCmd{
		TargetPath: repository,
		Output:     output,
		References: []string{"tag:v1.0", "tag:v2.0"},
		Metric:     "file-lines",
		Fill:       config.MetricSpec{Metric: "lines-changed.sum", Palette: "temperature"},
		Width:      640,
		Height:     480,
	}

	g.Expect(cmd.Run(&Flags{Config: config.New()})).To(Succeed())

	image, err := os.ReadFile(output)
	g.Expect(err).NotTo(HaveOccurred())

	labels := svgTextLabels(string(image))
	apiIndex := slices.Index(labels, "api")
	g.Expect(apiIndex).To(BeNumerically(">=", 0))
	g.Expect(len(labels)).To(BeNumerically(">", apiIndex+2))
	g.Expect(labels[apiIndex+2]).NotTo(Equal("0"))
}

func TestAlluvialCmd_Run_UsesHEADForEmptyReference(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repository := createAlluvialTagFixture(t)
	output := filepath.Join(t.TempDir(), "alluvial.svg")
	cmd := &AlluvialCmd{
		TargetPath: repository,
		Output:     output,
		References: []string{"tag:v1.0", ""},
		Metric:     "file-lines",
		Width:      320,
		Height:     240,
	}

	g.Expect(cmd.Run(&Flags{Config: config.New()})).To(Succeed())

	image, err := os.ReadFile(output)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(image)).To(ContainSubstring(">HEAD<"))
	g.Expect(string(image)).NotTo(ContainSubstring("uncommitted"))
}

func TestAlluvialCmd_Run_ExportsFirstSnapshotAndIgnoresChangedOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	repository := createAlluvialTagFixture(t)
	output := filepath.Join(t.TempDir(), "alluvial.png")
	exportPath := filepath.Join(t.TempDir(), "metrics.json")
	changedOnly := true
	cmd := &AlluvialCmd{
		TargetPath: repository,
		Output:     output,
		References: []string{"tag:v1.0", "tag:v2.0"},
		Metric:     "file-lines",
		Width:      320,
		Height:     240,
	}
	cfg := config.New()
	cfg.ChangedOnly = &changedOnly

	g.Expect(cmd.Run(&Flags{Config: cfg, ExportData: exportPath})).To(Succeed())
	g.Expect(output).To(BeARegularFile())
	g.Expect(exportPath).To(BeARegularFile())
}

func TestAlluvialCmd_Run_RendersFilteredExpandedSnapshotsInEachFormat(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		extension string
		check     func(Gomega, []byte)
	}{
		"SVG": {
			extension: ".svg",
			check: func(g Gomega, image []byte) {
				g.Expect(string(image)).To(ContainSubstring("tag:v1.0"))
				g.Expect(string(image)).To(ContainSubstring("tag:v2.0"))
				g.Expect(string(image)).To(ContainSubstring("tag:v3.0"))
				g.Expect(string(image)).To(ContainSubstring("internal/config"))
				g.Expect(string(image)).NotTo(ContainSubstring(">docs<"))
			},
		},
		"PNG": {
			extension: ".png",
			check: func(g Gomega, image []byte) {
				g.Expect(string(image)).To(HavePrefix("\x89PNG\r\n\x1a\n"))
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			repository := createAlluvialTagFixture(t)
			output := filepath.Join(t.TempDir(), "alluvial"+c.extension)
			cmd := &AlluvialCmd{
				TargetPath: repository,
				Output:     output,
				References: []string{"tag:v1.0", "tag:v2.0", "tag:v3.0"},
				Metric:     "file-lines",
				Expand:     []string{"internal"},
				Include:    []filter.Rule{{Pattern: "**/*.go", Mode: filter.Include}},
				Exclude:    []filter.Rule{{Pattern: "docs/**", Mode: filter.Exclude}},
				Width:      320,
				Height:     240,
			}

			g.Expect(cmd.Run(&Flags{Config: config.New()})).To(Succeed())

			image, err := os.ReadFile(output)
			g.Expect(err).NotTo(HaveOccurred())
			c.check(g, image)
		})
	}
}

func createAlluvialTagFixture(t *testing.T) string {
	t.Helper()
	g := NewGomegaWithT(t)
	root := t.TempDir()

	repository, err := gogit.PlainInit(root, false)
	g.Expect(err).NotTo(HaveOccurred())

	writeFixtureFile := func(name, content string) {
		t.Helper()

		filename := filepath.Join(root, name)
		g.Expect(os.MkdirAll(filepath.Dir(filename), 0o750)).To(Succeed())
		g.Expect(os.WriteFile(filename, []byte(content), 0o600)).To(Succeed())
	}
	commit := func(message string, when time.Time) plumbing.Hash {
		t.Helper()

		worktree, worktreeErr := repository.Worktree()
		g.Expect(worktreeErr).NotTo(HaveOccurred())
		_, worktreeErr = worktree.Add(".")
		g.Expect(worktreeErr).NotTo(HaveOccurred())

		hash, commitErr := worktree.Commit(message, &gogit.CommitOptions{
			Author: &object.Signature{Name: "Fixture", Email: "fixture@example.com", When: when},
		})
		g.Expect(commitErr).NotTo(HaveOccurred())

		return hash
	}

	writeFixtureFile("api/main.go", "package api\n\nfunc First() {}\n")

	first := commit("first release", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err = repository.CreateTag("v1.0", first, nil)
	g.Expect(err).NotTo(HaveOccurred())

	writeFixtureFile("docs/guide.md", "# Guide\n")
	writeFixtureFile("api/main.go", "package api\n\nfunc First() {}\nfunc Second() {}\n")

	second := commit("second release", time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	_, err = repository.CreateTag("v2.0", second, nil)
	g.Expect(err).NotTo(HaveOccurred())

	writeFixtureFile("internal/config/settings.go", "package config\n\nconst Name = \"fixture\"\n")

	third := commit("third release", time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC))
	_, err = repository.CreateTag("v3.0", third, nil)
	g.Expect(err).NotTo(HaveOccurred())

	writeFixtureFile("uncommitted/ignored.go", "package ignored\n")

	return root
}

func svgTextLabels(image string) []string {
	matches := regexp.MustCompile(`>([^<>]*)</text>`).FindAllStringSubmatch(image, -1)

	labels := make([]string, 0, len(matches))
	for _, match := range matches {
		labels = append(labels, match[1])
	}

	return labels
}
