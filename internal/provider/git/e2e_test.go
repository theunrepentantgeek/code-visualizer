package git_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// repoRoot returns the absolute path of the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	// This file is at internal/provider/git/e2e_test.go
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")

	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	// Sanity: go.mod should exist
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Fatalf("repo root %s does not contain go.mod", abs)
	}

	return abs
}

// setupE2E registers the git provider, scans the repo, loads all file-level
// git metrics, and returns the populated tree.
//
// Each test rescans the repo independently. A shared fixture would be more
// efficient, but conflicts with internal tests in this package that also
// manipulate the global base registry via ResetBaseRegistryForTesting.
func setupE2E(t *testing.T, rules []filter.Rule) *model.Directory {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	git.Register()

	root, err := scan.Scan(repoRoot(t), rules, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	err = provider.RunLoaders(
		root,
		[]metric.Name{
			git.FileAge,
			git.FileFreshness,
			git.AuthorCount,
			git.CommitCount,
			git.TotalLinesAdded,
			git.TotalLinesRemoved,
			git.CommitDensity,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("RunLoaders failed: %v", err)
	}

	return root
}

// resolveAndAggregate resolves a metric expression and computes aggregations
// over the tree, returning the resolved metric descriptor.
func resolveAndAggregate(t *testing.T, root *model.Directory, expression string) provider.ResolvedMetric {
	t.Helper()

	expr, err := metric.ParseExpression(expression)
	if err != nil {
		t.Fatalf("failed to parse expression %q: %v", expression, err)
	}

	resolved, err := provider.ResolveExpression(expr, metric.LevelDirectory)
	if err != nil {
		t.Fatalf("failed to resolve expression %q: %v", expression, err)
	}

	if err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved}); err != nil {
		t.Fatalf("ComputeAggregations failed for %q: %v", expression, err)
	}

	return resolved
}

// trackedFileCount returns the number of files in the tree that have the
// given quantity metric set (i.e., were found in git history).
func trackedFileCount(root *model.Directory, name metric.Name) int {
	var n int

	model.WalkFiles(root, func(f *model.File) {
		if _, ok := f.Quantity(name); ok {
			n++
		}
	})

	return n
}

// TestE2E_Git_FileAge_BaseMetric verifies that file-age is populated with
// positive values for files tracked in git.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_FileAge_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.FileAge)
		if !ok {
			return // untracked file
		}

		g.Expect(v).To(BeNumerically(">=", 0), "file-age should be >=0 for %s", f.Path)
	})

	// At least some files must have the metric
	g.Expect(trackedFileCount(root, git.FileAge)).To(BeNumerically(">", 10),
		"expected more than 10 tracked files to have file-age")
}

// TestE2E_Git_FileFreshness_BaseMetric verifies that file-freshness is
// populated with non-negative values for git-tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_FileFreshness_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.FileFreshness)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">=", 0), "file-freshness should be >=0 for %s", f.Path)
	})

	g.Expect(trackedFileCount(root, git.FileFreshness)).To(BeNumerically(">", 10),
		"expected more than 10 tracked files to have file-freshness")
}

// TestE2E_Git_AuthorCount_BaseMetric verifies that author-count is ≥ 1 for
// all tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_AuthorCount_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.AuthorCount)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">=", 1), "author-count should be >=1 for %s", f.Path)
	})

	g.Expect(trackedFileCount(root, git.AuthorCount)).To(BeNumerically(">", 10))
}

// TestE2E_Git_CommitCount_BaseMetric verifies that commit-count is ≥ 1 for
// all tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_CommitCount_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.CommitCount)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">=", 1), "commit-count should be >=1 for %s", f.Path)
	})

	g.Expect(trackedFileCount(root, git.CommitCount)).To(BeNumerically(">", 10))
}

// TestE2E_Git_TotalLinesAdded_BaseMetric verifies that total-lines-added is
// non-negative for all tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_TotalLinesAdded_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.TotalLinesAdded)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">=", 0), "total-lines-added should be >=0 for %s", f.Path)
	})

	g.Expect(trackedFileCount(root, git.TotalLinesAdded)).To(BeNumerically(">", 10))
}

// TestE2E_Git_TotalLinesRemoved_BaseMetric verifies that total-lines-removed
// is non-negative for all tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_TotalLinesRemoved_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(git.TotalLinesRemoved)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">=", 0), "total-lines-removed should be >=0 for %s", f.Path)
	})

	g.Expect(trackedFileCount(root, git.TotalLinesRemoved)).To(BeNumerically(">", 10))
}

// TestE2E_Git_CommitDensity_BaseMetric verifies that commit-density is
// positive for all tracked files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_CommitDensity_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var tracked int

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Measure(git.CommitDensity)
		if !ok {
			return
		}

		g.Expect(v).To(BeNumerically(">", 0), "commit-density should be >0 for %s", f.Path)

		tracked++
	})

	g.Expect(tracked).To(BeNumerically(">", 10),
		"expected more than 10 tracked files to have commit-density")
}

// TestE2E_Git_FileAgeAggregations verifies all supported aggregations for
// file-age (sum, min, max, mean) at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_FileAgeAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumR := resolveAndAggregate(t, root, "file-age.sum")
	minR := resolveAndAggregate(t, root, "file-age.min")
	maxR := resolveAndAggregate(t, root, "file-age.max")
	meanR := resolveAndAggregate(t, root, "file-age.mean")

	sumVal, ok := root.Quantity(sumR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-age.sum should be set")
	g.Expect(sumVal).To(BeNumerically(">", 0))

	// min can be 0 for files committed less than 24 hours ago (age truncates
	// to whole days), so only assert the metric was set.
	minVal, ok := root.Quantity(minR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-age.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 0))

	maxVal, ok := root.Quantity(maxR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-age.max should be set")
	g.Expect(maxVal).To(BeNumerically(">", 0))

	meanVal, ok := root.Measure(meanR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-age.mean should be set")
	g.Expect(meanVal).To(BeNumerically(">", 0))

	// Logical consistency
	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum >= max")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max >= min")
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)), "mean >= min")
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)), "mean <= max")

	// Aggregation must propagate to every directory in the tree, not just the
	// root. For each non-root directory that has files with file-age, verify
	// all four aggregates are populated and respect the same invariants.
	var subdirsChecked int

	model.WalkDirectories(root, func(d *model.Directory) {
		if d == root {
			return
		}

		dSum, hasSum := d.Quantity(sumR.ResultName)
		if !hasSum {
			return // directory has no tracked files
		}

		dMin, hasMin := d.Quantity(minR.ResultName)
		dMax, hasMax := d.Quantity(maxR.ResultName)
		dMean, hasMean := d.Measure(meanR.ResultName)

		g.Expect(hasMin).To(BeTrue(), "file-age.min should be set on %s", d.Path)
		g.Expect(hasMax).To(BeTrue(), "file-age.max should be set on %s", d.Path)
		g.Expect(hasMean).To(BeTrue(), "file-age.mean should be set on %s", d.Path)

		g.Expect(dSum).To(BeNumerically(">=", dMax), "sum >= max for %s", d.Path)
		g.Expect(dMax).To(BeNumerically(">=", dMin), "max >= min for %s", d.Path)
		g.Expect(dMean).To(BeNumerically(">=", float64(dMin)), "mean >= min for %s", d.Path)
		g.Expect(dMean).To(BeNumerically("<=", float64(dMax)), "mean <= max for %s", d.Path)

		subdirsChecked++
	})

	g.Expect(subdirsChecked).To(BeNumerically(">", 5),
		"expected file-age aggregates to propagate to multiple subdirectories")
}

// TestE2E_Git_FileFreshnessAggregations verifies aggregations for
// file-freshness (sum, min, max, mean) at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_FileFreshnessAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumR := resolveAndAggregate(t, root, "file-freshness.sum")
	minR := resolveAndAggregate(t, root, "file-freshness.min")
	maxR := resolveAndAggregate(t, root, "file-freshness.max")
	meanR := resolveAndAggregate(t, root, "file-freshness.mean")

	sumVal, ok := root.Quantity(sumR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-freshness.sum should be set")

	minVal, ok := root.Quantity(minR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-freshness.min should be set")

	maxVal, ok := root.Quantity(maxR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-freshness.max should be set")

	meanVal, ok := root.Measure(meanR.ResultName)
	g.Expect(ok).To(BeTrue(), "file-freshness.mean should be set")

	// Logical consistency (values can be 0 for very recently changed files)
	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum >= max")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max >= min")
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)), "mean >= min")
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)), "mean <= max")
}

// TestE2E_Git_AuthorCountAggregations verifies aggregations for author-count
// (sum, min, max, mean) at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_AuthorCountAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumR := resolveAndAggregate(t, root, "author-count.sum")
	minR := resolveAndAggregate(t, root, "author-count.min")
	maxR := resolveAndAggregate(t, root, "author-count.max")
	meanR := resolveAndAggregate(t, root, "author-count.mean")

	sumVal, ok := root.Quantity(sumR.ResultName)
	g.Expect(ok).To(BeTrue(), "author-count.sum should be set")
	g.Expect(sumVal).To(BeNumerically(">=", 1))

	minVal, ok := root.Quantity(minR.ResultName)
	g.Expect(ok).To(BeTrue(), "author-count.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 1))

	maxVal, ok := root.Quantity(maxR.ResultName)
	g.Expect(ok).To(BeTrue(), "author-count.max should be set")
	g.Expect(maxVal).To(BeNumerically(">=", 1))

	meanVal, ok := root.Measure(meanR.ResultName)
	g.Expect(ok).To(BeTrue(), "author-count.mean should be set")
	g.Expect(meanVal).To(BeNumerically(">=", 1.0))

	// Logical consistency
	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum >= max")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max >= min")
}

// TestE2E_Git_CommitCountAggregations verifies aggregations for commit-count
// (sum, min, max, mean) at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_CommitCountAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumR := resolveAndAggregate(t, root, "commit-count.sum")
	minR := resolveAndAggregate(t, root, "commit-count.min")
	maxR := resolveAndAggregate(t, root, "commit-count.max")
	meanR := resolveAndAggregate(t, root, "commit-count.mean")

	sumVal, ok := root.Quantity(sumR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-count.sum should be set")
	g.Expect(sumVal).To(BeNumerically(">=", 1))

	minVal, ok := root.Quantity(minR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-count.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 1))

	_, ok = root.Quantity(maxR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-count.max should be set")

	_, ok = root.Measure(meanR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-count.mean should be set")

	// Logical consistency
	g.Expect(sumVal).To(BeNumerically(">=", minVal), "sum >= min")
}

// TestE2E_Git_TotalLinesAddedAggregations verifies aggregations for
// total-lines-added (sum, min, max, mean) at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_TotalLinesAddedAggregations(t *testing.T) {
	cases := []string{
		"total-lines-added",
		"total-lines-removed",
	}

	for _, base := range cases {
		t.Run(base, func(t *testing.T) {
			g := NewGomegaWithT(t)
			root := setupE2E(t, nil)

			sumR := resolveAndAggregate(t, root, base+".sum")
			minR := resolveAndAggregate(t, root, base+".min")
			maxR := resolveAndAggregate(t, root, base+".max")
			meanR := resolveAndAggregate(t, root, base+".mean")

			sumVal, ok := root.Quantity(sumR.ResultName)
			g.Expect(ok).To(BeTrue(), base+".sum should be set")
			g.Expect(sumVal).To(BeNumerically(">", 0))

			_, ok = root.Quantity(minR.ResultName)
			g.Expect(ok).To(BeTrue(), base+".min should be set")

			maxVal, ok := root.Quantity(maxR.ResultName)
			g.Expect(ok).To(BeTrue(), base+".max should be set")
			g.Expect(maxVal).To(BeNumerically(">", 0))

			_, ok = root.Measure(meanR.ResultName)
			g.Expect(ok).To(BeTrue(), base+".mean should be set")

			g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum >= max")
		})
	}
}

// TestE2E_Git_CommitDensityAggregations verifies the supported aggregations
// for commit-density (min, max only — it is a Measure kind).
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_CommitDensityAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	minR := resolveAndAggregate(t, root, "commit-density.min")
	maxR := resolveAndAggregate(t, root, "commit-density.max")

	minVal, ok := root.Measure(minR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-density.min should be set")
	g.Expect(minVal).To(BeNumerically(">", 0))

	maxVal, ok := root.Measure(maxR.ResultName)
	g.Expect(ok).To(BeTrue(), "commit-density.max should be set")
	g.Expect(maxVal).To(BeNumerically(">", 0))

	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max >= min")
}

// TestE2E_Git_TemporalConsistency verifies that file-age (days since first
// commit) ≥ file-freshness (days since last commit) for every tracked file,
// since the last commit cannot predate the first.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_TemporalConsistency(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var checked int

	model.WalkFiles(root, func(f *model.File) {
		age, hasAge := f.Quantity(git.FileAge)
		freshness, hasFreshness := f.Quantity(git.FileFreshness)

		if !hasAge || !hasFreshness {
			return
		}

		g.Expect(age).To(BeNumerically(">=", freshness),
			"file-age (%d) should be >= file-freshness (%d) for %s", age, freshness, f.Path)

		checked++
	})

	g.Expect(checked).To(BeNumerically(">", 10),
		"expected at least 10 files to have both file-age and file-freshness")
}

// TestE2E_Git_AggregationsOnSubdirectories verifies that subdirectories
// receive aggregated commit-count values, not just the root.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_AggregationsOnSubdirectories(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "commit-count.sum")

	var dirsWithValue int

	for _, d := range root.Dirs {
		if v, ok := d.Quantity(resolved.ResultName); ok && v > 0 {
			dirsWithValue++
		}
	}

	g.Expect(dirsWithValue).To(BeNumerically(">", 0),
		"at least one immediate subdirectory should have commit-count.sum populated")
}

// TestE2E_Git_ExcludeFilterByExtension verifies that exclude path filtering
// removes all files matching a given extension pattern.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_ExcludeFilterByExtension(t *testing.T) {
	g := NewGomegaWithT(t)

	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
	}
	root := setupE2E(t, rules)

	model.WalkFiles(root, func(f *model.File) {
		g.Expect(f.Extension).NotTo(Equal("go"),
			"exclude filter should remove .go files, found: %s", f.Name)
	})

	// Non-Go files (go.mod, go.sum, YAML, etc.) should still be present
	g.Expect(model.CountFiles(root)).To(BeNumerically(">", 0),
		"should still have non-Go files in the repo")
}

// TestE2E_Git_ExcludeFilterReducesMetrics verifies that after filtering out
// test files, aggregations work correctly on the reduced file set.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_ExcludeFilterReducesMetrics(t *testing.T) {
	g := NewGomegaWithT(t)

	// Scan all Go files to get a baseline
	baseRoot := setupE2E(t, nil)
	baseResolved := resolveAndAggregate(t, baseRoot, "commit-count.sum")
	baseSum, ok := baseRoot.Quantity(baseResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "baseline commit-count.sum should be set")

	// Scan again excluding test files
	filteredRoot := setupE2E(t, []filter.Rule{
		{Pattern: "**/*_test.go", Mode: filter.Exclude},
	})
	filteredResolved := resolveAndAggregate(t, filteredRoot, "commit-count.sum")
	filteredSum, ok := filteredRoot.Quantity(filteredResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "filtered commit-count.sum should be set")

	// Removing files must not increase the aggregate sum
	g.Expect(filteredSum).To(BeNumerically("<=", baseSum),
		"excluding test files should not increase commit-count.sum")
}

// TestE2E_Git_CommitMetrics_Registered verifies that the commit-level
// metrics (lines-added, lines-removed, lines-changed) are correctly
// registered in the base registry. These metrics require a separate
// commit-level loading path and are not populated by the file-level
// loader, but they must be resolvable with a supported aggregation.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Git_CommitMetrics_Registered(t *testing.T) {
	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	git.Register()

	g := NewGomegaWithT(t)

	for _, expr := range []string{
		"lines-added.sum",
		"lines-removed.sum",
		"lines-changed.sum",
		"lines-added.min",
		"lines-removed.min",
		"lines-changed.min",
		"lines-added.max",
		"lines-removed.max",
		"lines-changed.max",
	} {
		parsed, err := metric.ParseExpression(expr)
		g.Expect(err).NotTo(HaveOccurred(), "failed to parse %q", expr)

		// Commit-level metrics aggregate to directory level via sum/min/max/mean.
		resolved, err := provider.ResolveExpression(parsed, metric.LevelDirectory)
		g.Expect(err).NotTo(HaveOccurred(), "failed to resolve %q at directory level", expr)
		g.Expect(string(resolved.ResultName)).NotTo(BeEmpty(), "result name should not be empty for %q", expr)
	}
}
