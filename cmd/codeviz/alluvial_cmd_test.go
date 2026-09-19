package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	gogit "github.com/go-git/go-git/v5"

	"github.com/alecthomas/kong"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
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
			wantErr: "unknown metric",
		},
		{
			name: "rejects parent expansion",
			cfg: &config.Alluvial{
				References: []string{"tag:v1.0", "tag:v2.0"},
				Metric:     new("file-size"),
				Expand:     []string{"../outside"},
			},
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

	second := commit("second release", time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC))
	_, err = repository.CreateTag("v2.0", second, nil)
	g.Expect(err).NotTo(HaveOccurred())

	writeFixtureFile("uncommitted/ignored.go", "package ignored\n")

	return root
}
