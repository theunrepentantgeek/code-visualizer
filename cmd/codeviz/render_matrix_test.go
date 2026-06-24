package main

// End-to-end metric-render matrix: confirm that *every* metric the registry
// knows actually computes and renders through the real CLI pipeline.
//
// Metrics and visualizations are orthogonal concerns: whether a metric resolves,
// computes a value and feeds a render is independent of which visualization
// consumes it. So rather than cross every metric with every viz, this suite
// enumerates the full set of valid metric expressions straight from the metric
// registry — each base metric, each base × aggregation, and each filter × base
// (× aggregation) — and drives each one end-to-end through a single
// representative visualization (treemap). Because the set is derived from
// provider.AllBase() and provider.ResolveForValidation, newly added metrics and
// aggregations are covered automatically without editing this test (see #442).
//
// This exercises the real pipeline — providers, declaration parsing,
// aggregations, layout and rendering — not just config validation, which is the
// wiring bug #440 slipped through: declarations.count validated fine as a
// fill/border metric but was rejected as a size metric because nothing drove the
// full render path per role. Each numeric metric here is placed in the size role
// (the role that regressed), so this matrix would have failed on the pre-#441
// code.
//
// Providers (filesystem, git, golang) are registered in TestMain (main_test.go).

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

const (
	matrixWidth  = 240
	matrixHeight = 180
)

// fixtureAlphaGo is a Go file with a deterministic mix of exported/unexported
// declarations (types, interfaces, structs, methods, functions, constants,
// variables), imports and comments, so the Go metrics (and their filtered and
// aggregated forms) all compute non-trivial values.
const fixtureAlphaGo = `// Package sample is a deterministic fixture for the metric-render matrix.
package sample

import (
	"fmt"
	"strings"
)

// ExportedConst is an exported constant.
const ExportedConst = 1

// unexportedConst is an unexported constant.
const unexportedConst = 2

// ExportedVar is an exported variable.
var ExportedVar = 10

// Shape is an exported interface.
type Shape interface {
	Area() float64
}

// ExportedType is an exported struct type.
type ExportedType struct {
	Field int
}

// Area implements Shape; its branches give it non-trivial complexity.
func (e ExportedType) Area() float64 {
	if e.Field > 0 {
		return float64(e.Field)
	}

	return 0
}

// Classify returns a label based on the sign of x.
func (e ExportedType) Classify(x int) string {
	switch {
	case x > 0:
		return strings.ToUpper("positive")
	case x < 0:
		return "negative"
	default:
		return "zero"
	}
}

// Exported formats and returns x.
func Exported(x int) string {
	return fmt.Sprintf("%d", x)
}

// unexported is an unexported helper.
func unexported() {}
`

// fixtureBetaGo is a second Go file so directory-level aggregation spans more
// than one file.
const fixtureBetaGo = `// Package sample (beta) adds a second file to the fixture.
package sample

// Helper is an exported helper type.
type Helper struct{}

// Run runs the helper.
func (Helper) Run() {}

// Total adds two numbers.
func Total(a, b int) int {
	return a + b
}
`

// writeRenderMatrixFixture builds a small deterministic source tree under a git
// repository and returns the path to scan. The mix of Go and non-Go files
// exercises file-type classification alongside the Go-specific metrics, and the
// git repository lets the git metrics compute.
//
// The scanned tree lives in a "src" subdirectory while the .git directory sits
// in its parent, so the filesystem scan of the target stays clean (no git
// internals leak into the visualization) while git-derived metrics can still
// resolve the repository.
func writeRenderMatrixFixture(t *testing.T) string {
	t.Helper()
	g := NewGomegaWithT(t)

	root := t.TempDir()
	target := filepath.Join(root, "src")
	g.Expect(os.MkdirAll(target, 0o750)).To(Succeed())

	files := map[string]string{
		"alpha.go":  fixtureAlphaGo,
		"beta.go":   fixtureBetaGo,
		"notes.txt": "Project notes.\nSecond line of notes.\n",
	}

	for name, content := range files {
		path := filepath.Join(target, name)
		g.Expect(os.WriteFile(path, []byte(content), 0o600)).
			To(Succeed(), "writing fixture file %s", name)
	}

	initFixtureGitRepo(t, root)

	return target
}

// initFixtureGitRepo turns root into a git repository with two commits at fixed,
// distinct timestamps. The git metrics need a commit history to compute, and a
// non-zero time span keeps file-age / commit-density meaningful. Pinning both
// the author and committer dates (via GIT_AUTHOR_DATE / GIT_COMMITTER_DATE)
// keeps the history deterministic across machines and CI — the determinism #442
// calls out for git fixtures.
func initFixtureGitRepo(t *testing.T, root string) {
	t.Helper()
	g := NewGomegaWithT(t)

	runAt := func(date string, args ...string) {
		t.Helper()

		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // test helper, fixed args
		cmd.Dir = root

		cmd.Env = append(
			os.Environ(),
			"GIT_AUTHOR_NAME=Fixture Author",
			"GIT_AUTHOR_EMAIL=fixture@example.com",
			"GIT_COMMITTER_NAME=Fixture Author",
			"GIT_COMMITTER_EMAIL=fixture@example.com",
			"GIT_AUTHOR_DATE="+date,
			"GIT_COMMITTER_DATE="+date,
		)

		out, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred(), "command %v failed: %s", args, out)
	}

	const (
		firstDate  = "2024-01-01T00:00:00+00:00"
		secondDate = "2024-06-01T00:00:00+00:00"
	)

	runAt(firstDate, "git", "init")
	runAt(firstDate, "git", "config", "user.name", "Fixture Author")
	runAt(firstDate, "git", "config", "user.email", "fixture@example.com")
	runAt(firstDate, "git", "add", ".")
	runAt(firstDate, "git", "commit", "-m", "fixture commit")

	// A second commit at a later date gives the git metrics a non-zero time
	// span. Touching only the non-Go notes file keeps the Go metrics stable.
	notes := filepath.Join(root, "src", "notes.txt")
	g.Expect(os.WriteFile(notes, []byte("Project notes.\nSecond line of notes.\nThird line.\n"), 0o600)).
		To(Succeed())

	runAt(secondDate, "git", "add", ".")
	runAt(secondDate, "git", "commit", "-m", "update notes")
}

// metricExpr is a valid metric expression together with the kind of value it
// resolves to, which determines the role it is rendered in.
type metricExpr struct {
	name string
	kind metric.Kind
}

// allValidMetricExpressions enumerates every metric expression the registry
// accepts. Candidate expressions come from candidateMetricExpressions; validity
// is decided by provider.ResolveForValidation — the same resolver config and CLI
// validation use — so the set automatically tracks the registry as metrics,
// aggregations and filters are added or removed.
func allValidMetricExpressions(t *testing.T) []metricExpr {
	t.Helper()

	seen := make(map[string]bool)
	candidates := candidateMetricExpressions()
	exprs := make([]metricExpr, 0, len(candidates))

	for _, name := range candidates {
		if seen[name] {
			continue
		}

		seen[name] = true

		resolved, err := provider.ResolveForValidation(metric.Name(name))
		if err != nil {
			continue
		}

		exprs = append(exprs, metricExpr{name: name, kind: resolved.ResultKind})
	}

	return exprs
}

// candidateMetricExpressions builds every expression worth probing for each
// registered base metric: the bare metric, each base × aggregation, and each
// filter × base (× aggregation). Invalid combinations are filtered out later by
// allValidMetricExpressions.
func candidateMetricExpressions() []string {
	names := make([]string, 0)

	for _, desc := range provider.AllBase() {
		base := string(desc.Name)

		names = append(names, base)

		for _, agg := range desc.Aggregations {
			names = append(names, base+"."+string(agg))
		}

		for _, filterName := range desc.Filters {
			filtered := string(filterName) + "." + base
			names = append(names, filtered)

			for _, agg := range desc.Aggregations {
				names = append(names, filtered+"."+string(agg))
			}
		}
	}

	return names
}

// renderMetricEndToEnd drives the full treemap pipeline with expr placed in a
// role appropriate to its kind (numeric → size, classification → fill) and
// asserts the pipeline succeeds and writes non-empty output. Numeric metrics go
// in the size role specifically because that is the role bug #440 regressed.
func renderMetricEndToEnd(t *testing.T, target string, expr metricExpr) {
	t.Helper()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "out.svg")

	cmd := &TreemapCmd{
		TargetPath: target,
		Output:     out,
		Size:       filesystem.FileLines, // baseline numeric size; overridden below for numeric metrics
		Width:      matrixWidth,
		Height:     matrixHeight,
	}

	if expr.kind == metric.Classification {
		cmd.Fill = config.MetricSpec{Metric: metric.Name(expr.name)}
	} else {
		cmd.Size = metric.Name(expr.name)
	}

	flags := &Flags{Config: config.New()}
	g.Expect(cmd.Run(flags)).To(Succeed(), "rendering metric %q", expr.name)

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred(), "output file for %q should exist", expr.name)
	g.Expect(data).NotTo(BeEmpty(), "output for %q should be non-empty", expr.name)
}

// TestAllMetrics_RenderEndToEnd renders every metric expression the registry
// accepts through the real treemap pipeline and asserts it produces output.
func TestAllMetrics_RenderEndToEnd(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	target := writeRenderMatrixFixture(t)

	exprs := allValidMetricExpressions(t)
	g.Expect(exprs).NotTo(BeEmpty(), "registry should yield at least one valid metric expression")

	for _, expr := range exprs {
		t.Run(expr.name, func(t *testing.T) {
			t.Parallel()
			renderMetricEndToEnd(t, target, expr)
		})
	}
}
