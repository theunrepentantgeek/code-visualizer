package main

// End-to-end render-matrix tests: drive each visualization command's full Run()
// pipeline against a deterministic on-disk fixture, asserting that every metric
// role accepts every metric kind it should and produces non-empty output.
//
// Unlike validation_matrix_test.go (which only exercises validateConfig in
// isolation), these tests run the real pipeline — providers, declaration
// parsing, aggregations, layout and rendering — for every matrix row. That is
// the wiring bug #440 slipped through: declarations.count validated fine as a
// fill/border metric but was rejected as a size/axis metric, because nothing
// drove the full render path per role.
//
// Coverage: {viz command} × {metric role} × {metric kind}, where:
//   - viz commands: treemap, bubbletree, radialtree, spiral, scatter
//   - roles: size/disc-size, x-axis, y-axis, fill, border
//   - kinds: base quantity/measure/classification, declaration-count
//     aggregation, filtered aggregation, numeric aggregation.
//
// Git-derived metrics are deliberately excluded: they need a fixed commit
// history to be deterministic (see #442), which is out of scope here. Every
// metric kind below is computed purely from the filesystem and Go providers,
// which are registered in TestMain (main_test.go).

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// fixtureAlphaGo is a Go file with a known mix of exported/unexported
// declarations (types, constants, variables, functions, methods) and comments,
// so declaration counts, comment ratios and cyclomatic complexity are all
// non-trivial and stable.
const fixtureAlphaGo = `// Package sample is a deterministic fixture for render-matrix tests.
package sample

// ExportedConst is an exported constant.
const ExportedConst = 1

// unexportedConst is an unexported constant.
const unexportedConst = 2

// ExportedVar is an exported variable.
var ExportedVar = 10

// ExportedType is an exported struct type.
type ExportedType struct {
	Field int
}

// Classify returns a label based on the sign of x; the branches give it
// non-trivial cyclomatic complexity.
func (e ExportedType) Classify(x int) string {
	if x > 0 {
		return "positive"
	}

	if x < 0 {
		return "negative"
	}

	return "zero"
}

// Exported is an exported function.
func Exported(x int) int {
	return x + 1
}

// unexported is an unexported helper.
func unexported() {}
`

// fixtureBetaGo is a second Go file to give the tree more than one declaration
// site and exercise directory-level aggregation across files.
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
// exercises file-type classification as well as the Go-specific metrics.
//
// The scanned tree lives in a "src" subdirectory while the .git directory sits
// in its parent, so the filesystem scan of the target stays clean (no git
// internals leak into the visualization) while git-history-dependent
// visualizations (spiral) can still resolve the repository.
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

// initFixtureGitRepo turns root into a git repository with two commits at
// fixed, distinct timestamps and returns. The spiral visualization always loads
// git history (it lays files out over time) and needs a non-zero commit time
// range to build its time buckets, so a single commit is not enough — even when
// no git-derived metric is selected. Pinning both the author and committer
// dates (via GIT_AUTHOR_DATE / GIT_COMMITTER_DATE) keeps the history
// deterministic across machines and CI — the determinism #442 calls out for git
// fixtures.
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

		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("command %v failed: %s\n%s", args, err, out)
		}
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

	// A second commit at a later date gives the spiral a non-zero commit time
	// range. Touching only the non-Go notes file keeps the Go metrics stable.
	notes := filepath.Join(root, "src", "notes.txt")
	g.Expect(os.WriteFile(notes, []byte("Project notes.\nSecond line of notes.\nThird line.\n"), 0o600)).
		To(Succeed())

	runAt(secondDate, "git", "add", ".")
	runAt(secondDate, "git", "commit", "-m", "update notes")
}

// Deterministic metric kinds, one representative per kind the issue calls out.
// All are computed from the filesystem and Go providers (no git history).
const (
	kindQuantity       = "file-size"                  // base quantity
	kindMeasure        = "comment-ratio"              // base measure
	kindClassification = "file-type"                  // base classification
	kindCountAgg       = "declarations.count"         // declaration-count aggregation (the #440 case)
	kindFilteredAgg    = "public.declarations.count"  // filtered aggregation
	kindNumericAgg     = "cyclomatic-complexity.mean" // numeric aggregation
)

// numericKinds resolve to a numeric value and are valid for numeric roles
// (size, disc-size). classification is intentionally absent.
var numericKinds = []string{
	kindQuantity,
	kindMeasure,
	kindCountAgg,
	kindFilteredAgg,
	kindNumericAgg,
}

// allKinds add classification, valid for axis and colour roles.
var allKinds = append(append([]string{}, numericKinds...), kindClassification)

// allVizCommands is every visualization command that participates in the matrix.
var allVizCommands = []string{"treemap", "bubbletree", "radialtree", "spiral", "scatter"}

const (
	matrixWidth  = 240
	matrixHeight = 180
)

// roles describes the metric assigned to each rendering role. Empty fields are
// left unset on the command (the role is not exercised for that row).
type roles struct {
	size   string // size / disc-size
	xAxis  string
	yAxis  string
	fill   string
	border string
}

// baselineRoles returns a set of valid role assignments sufficient for every
// viz command to render. Individual tests override exactly one role with the
// kind under test.
func baselineRoles() roles {
	return roles{
		size:  kindQuantity,
		xAxis: kindQuantity,
		yAxis: kindMeasure,
	}
}

// spec builds a fill/border MetricSpec, or the zero (unset) value when name is
// empty so the role passes through unexercised.
func spec(name string) config.MetricSpec {
	if name == "" {
		return config.MetricSpec{}
	}

	return config.MetricSpec{Metric: metric.Name(name)}
}

// newVizCmd constructs the named visualization command wired to render the
// given fixture to out, with the supplied role assignments. The returned value
// satisfies presetRunner (Run(flags) error).
func newVizCmd(viz, target, out string, r roles) presetRunner {
	switch viz {
	case "treemap":
		return &TreemapCmd{
			TargetPath: target, Output: out,
			Size:   metric.Name(r.size),
			Fill:   spec(r.fill),
			Border: spec(r.border),
			Width:  matrixWidth, Height: matrixHeight,
		}
	case "bubbletree":
		return &BubbletreeCmd{
			TargetPath: target, Output: out,
			Size:   metric.Name(r.size),
			Fill:   spec(r.fill),
			Border: spec(r.border),
			Width:  matrixWidth, Height: matrixHeight,
		}
	case "radialtree":
		return &RadialCmd{
			TargetPath: target, Output: out,
			DiscSize: metric.Name(r.size),
			Fill:     spec(r.fill),
			Border:   spec(r.border),
			Width:    matrixWidth, Height: matrixHeight,
		}
	case "spiral":
		return &SpiralCmd{
			TargetPath: target, Output: out,
			Size:   metric.Name(r.size),
			Fill:   spec(r.fill),
			Border: spec(r.border),
			Width:  matrixWidth, Height: matrixHeight,
		}
	case "scatter":
		return &ScatterCmd{
			TargetPath: target, Output: out,
			XAxis:  metric.Name(r.xAxis),
			YAxis:  metric.Name(r.yAxis),
			Size:   metric.Name(r.size),
			Fill:   spec(r.fill),
			Border: spec(r.border),
			Width:  matrixWidth, Height: matrixHeight,
		}
	default:
		return nil
	}
}

// runRenderMatrixCase drives a command's full pipeline and asserts it produced
// non-empty output.
func runRenderMatrixCase(t *testing.T, viz string, r roles) {
	t.Helper()
	g := NewGomegaWithT(t)

	dir := writeRenderMatrixFixture(t)
	out := filepath.Join(dir, "out.svg")

	cmd := newVizCmd(viz, dir, out, r)
	g.Expect(cmd).NotTo(BeNil(), "unknown viz command %q", viz)

	flags := &Flags{Config: config.New()}
	g.Expect(cmd.Run(flags)).To(Succeed())

	info, err := os.Stat(out)
	g.Expect(err).NotTo(HaveOccurred(), "output file should exist")
	g.Expect(info.Size()).To(BeNumerically(">", 0), "output file should be non-empty")
}

// TestRenderMatrix_SizeRole renders every viz command with each numeric metric
// kind in its size/disc-size role. This is the regression net for #440: a
// declaration-count aggregation (declarations.count) must render end-to-end as
// a size metric, not just validate.
func TestRenderMatrix_SizeRole(t *testing.T) {
	t.Parallel()

	for _, viz := range allVizCommands {
		for _, kind := range numericKinds {
			t.Run(viz+"/size/"+kind, func(t *testing.T) {
				t.Parallel()

				r := baselineRoles()
				r.size = kind
				runRenderMatrixCase(t, viz, r)
			})
		}
	}
}

// TestRenderMatrix_ScatterAxes renders the scatter command with each metric
// kind (including classification) in each axis role.
func TestRenderMatrix_ScatterAxes(t *testing.T) {
	t.Parallel()

	for _, axis := range []string{"x-axis", "y-axis"} {
		for _, kind := range allKinds {
			t.Run(axis+"/"+kind, func(t *testing.T) {
				t.Parallel()

				r := baselineRoles()
				if axis == "x-axis" {
					r.xAxis = kind
				} else {
					r.yAxis = kind
				}

				runRenderMatrixCase(t, "scatter", r)
			})
		}
	}
}

// TestRenderMatrix_ColourRoles renders every viz command with each metric kind
// (including classification) in its fill and border roles.
func TestRenderMatrix_ColourRoles(t *testing.T) {
	t.Parallel()

	for _, viz := range allVizCommands {
		for _, role := range []string{"fill", "border"} {
			for _, kind := range allKinds {
				t.Run(viz+"/"+role+"/"+kind, func(t *testing.T) {
					t.Parallel()

					r := baselineRoles()
					if role == "fill" {
						r.fill = kind
					} else {
						r.border = kind
					}

					runRenderMatrixCase(t, viz, r)
				})
			}
		}
	}
}
