package filesystem_test

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
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
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

	// This file is at internal/provider/filesystem/e2e_test.go
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

// setupE2E registers the filesystem provider, scans the repo, loads metrics,
// and returns the populated tree.
//
// Each test currently rescans the repo independently. A shared fixture
// (TestMain + sync.Once) would eliminate the redundancy, but conflicts with
// internal tests in this package that call ResetBaseRegistryForTesting().
// Solving this requires either moving E2E tests to a separate package or
// making RegisterBase() idempotent.
func setupE2E(
	t *testing.T,
	rules []filter.Rule,
) *model.Directory {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)

	filesystem.Register()

	// Mirror default config behavior: exclude dotfiles (e.g. .git) to keep the E2E scan bounded.
	if rules == nil {
		rules = make([]filter.Rule, 0)
	}

	rules = append(rules, filter.Rule{Pattern: ".*", Mode: filter.Exclude})

	root, err := scan.Scan(repoRoot(t), rules, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	err = provider.RunLoaders(
		root,
		[]metric.Name{filesystem.FileSize, filesystem.FileLines, filesystem.FileType},
		nil,
	)
	if err != nil {
		t.Fatalf("RunLoaders failed: %v", err)
	}

	return root
}

// resolveAndAggregate resolves a metric expression string and computes
// aggregations over the given tree.
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

	err = stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	if err != nil {
		t.Fatalf("ComputeAggregations failed for %q: %v", expression, err)
	}

	return resolved
}

// TestE2E_FileSize_BaseMetric verifies that file-size is populated on all
// scanned files and contains reasonable values.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileSize_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var count int

	var nonEmpty int

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Quantity(filesystem.FileSize)
		g.Expect(ok).To(BeTrue(), "file-size missing for %s", f.Path)
		g.Expect(v).To(BeNumerically(">=", 0), "file-size should be >=0 for %s", f.Path)

		if v > 0 {
			nonEmpty++
		}

		count++
	})

	// The repo should have a reasonable number of files
	g.Expect(count).To(BeNumerically(">", 10), "expected more than 10 files in the repo")
	g.Expect(nonEmpty).To(BeNumerically(">", 10), "expected most files to have size >0")
}

// TestE2E_FileLines_BaseMetric verifies that file-lines is populated on
// all non-binary text files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileLines_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var textFileCount int

	model.WalkFiles(root, func(f *model.File) {
		if f.IsBinary {
			return
		}

		v, ok := f.Quantity(filesystem.FileLines)
		g.Expect(ok).To(BeTrue(), "file-lines missing for text file %s", f.Path)
		g.Expect(v).To(BeNumerically(">=", 0), "file-lines should be >=0 for %s", f.Path)

		textFileCount++
	})

	g.Expect(textFileCount).To(BeNumerically(">", 10), "expected more than 10 text files")
}

// TestE2E_FileType_BaseMetric verifies that file-type is populated with
// non-empty classifications.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileType_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var count int

	model.WalkFiles(root, func(f *model.File) {
		v, ok := f.Classification(filesystem.FileType)
		g.Expect(ok).To(BeTrue(), "file-type missing for %s", f.Path)
		g.Expect(v).NotTo(BeEmpty(), "file-type should be non-empty for %s", f.Path)

		count++
	})

	g.Expect(count).To(BeNumerically(">", 10))
}

// TestE2E_FileSizeAggregations verifies all supported aggregations for file-size
// at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileSizeAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	tests := []struct {
		expression string
		checkKind  metric.Kind
		minVal     int // minimum expected value (0 for min which can be 0 for empty files)
	}{
		{"file-size.sum", metric.Quantity, 1},
		{"file-size.min", metric.Quantity, 0},
		{"file-size.max", metric.Quantity, 1},
		{"file-size.mean", metric.Measure, 1},
	}

	resolved := make(map[string]provider.ResolvedMetric, len(tests))

	for _, tt := range tests {
		resolved[tt.expression] = resolveAndAggregate(t, root, tt.expression)

		switch tt.checkKind {
		case metric.Quantity:
			v, ok := root.Quantity(resolved[tt.expression].ResultName)
			g.Expect(ok).To(BeTrue(), "expected %s to be set on root", tt.expression)
			g.Expect(v).To(BeNumerically(">=", tt.minVal), "%s should be >=%d", tt.expression, tt.minVal)
		case metric.Measure:
			v, ok := root.Measure(resolved[tt.expression].ResultName)
			g.Expect(ok).To(BeTrue(), "expected %s to be set on root", tt.expression)
			g.Expect(v).To(BeNumerically(">=", float64(tt.minVal)), "%s should be >=%d", tt.expression, tt.minVal)
		default:
			t.Fatalf("unexpected checkKind %d for %s", tt.checkKind, tt.expression)
		}
	}

	// Verify logical consistency: sum >= max >= min >= 0
	sumVal, _ := root.Quantity(resolved["file-size.sum"].ResultName)
	maxVal, _ := root.Quantity(resolved["file-size.max"].ResultName)
	minVal, _ := root.Quantity(resolved["file-size.min"].ResultName)

	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum should be >= max")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max should be >= min")
	g.Expect(minVal).To(BeNumerically(">=", 0), "min should be >=0")
}

// TestE2E_FileLinesAggregations verifies all supported aggregations for file-lines
// at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileLinesAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	tests := []struct {
		expression string
		checkKind  metric.Kind
	}{
		{"file-lines.sum", metric.Quantity},
		{"file-lines.min", metric.Quantity},
		{"file-lines.max", metric.Quantity},
		{"file-lines.mean", metric.Measure},
	}

	for _, tt := range tests {
		resolved := resolveAndAggregate(t, root, tt.expression)

		switch tt.checkKind {
		case metric.Quantity:
			v, ok := root.Quantity(resolved.ResultName)
			g.Expect(ok).To(BeTrue(), "expected %s to be set on root", tt.expression)
			g.Expect(v).To(BeNumerically(">=", 0), "%s should be >=0", tt.expression)
		case metric.Measure:
			v, ok := root.Measure(resolved.ResultName)
			g.Expect(ok).To(BeTrue(), "expected %s to be set on root", tt.expression)
			g.Expect(v).To(BeNumerically(">", 0), "%s should be >0", tt.expression)
		default:
			t.Fatalf("unexpected checkKind %d for %s", tt.checkKind, tt.expression)
		}
	}

	// Verify logical consistency
	sumResolved := resolveAndAggregate(t, root, "file-lines.sum")
	maxResolved := resolveAndAggregate(t, root, "file-lines.max")

	sumVal, _ := root.Quantity(sumResolved.ResultName)
	maxVal, _ := root.Quantity(maxResolved.ResultName)

	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum should be >= max")
	g.Expect(sumVal).To(BeNumerically(">", 0), "sum should be >0 for the whole repo")
}

// TestE2E_FileTypeAggregations verifies mode and distinct aggregations for
// file-type at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileTypeAggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	// mode: should return a non-empty classification
	modeResolved := resolveAndAggregate(t, root, "file-type.mode")
	modeVal, ok := root.Classification(modeResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "file-type.mode should be set on root")
	g.Expect(modeVal).NotTo(BeEmpty(), "file-type.mode should produce a non-empty value")

	// distinct: should return count > 1 (repo has .go, .mod, .sum, .yml, etc.)
	distinctResolved := resolveAndAggregate(t, root, "file-type.distinct")
	distinctVal, ok := root.Quantity(distinctResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "file-type.distinct should be set on root")
	g.Expect(distinctVal).To(BeNumerically(">", 1), "file-type.distinct should be >1")
}

// TestE2E_AggregationsOnSubdirectories verifies that subdirectories also
// receive aggregated values, not just the root.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_AggregationsOnSubdirectories(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "file-size.sum")

	// Check that at least one subdirectory has a value
	var dirsWithValue int

	for _, d := range root.Dirs {
		if v, ok := d.Quantity(resolved.ResultName); ok && v > 0 {
			dirsWithValue++
		}
	}

	g.Expect(dirsWithValue).To(BeNumerically(">", 0),
		"at least one immediate subdirectory should have file-size.sum populated")
}

// TestE2E_ExcludeFilterByExtension verifies that exclude path filtering
// removes all files matching a given extension pattern.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ExcludeFilterByExtension(t *testing.T) {
	g := NewGomegaWithT(t)

	// Exclude all .go files — verifies extension-based filtering at any depth.
	rules := []filter.Rule{
		{Pattern: "**/*.go", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	// Verify no .go files remain
	model.WalkFiles(root, func(f *model.File) {
		g.Expect(f.Extension).NotTo(Equal("go"),
			"exclude filter should remove .go files, found: %s", f.Name)
	})

	// Verify we still have non-Go files
	g.Expect(model.CountFiles(root)).To(BeNumerically(">", 0),
		"should still have non-Go files in the repo")
}

// TestE2E_ExcludeFilter verifies that exclude path filtering removes matching
// files from the scan.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ExcludeFilter(t *testing.T) {
	g := NewGomegaWithT(t)

	rules := []filter.Rule{
		{Pattern: "**/*_test.go", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	model.WalkFiles(root, func(f *model.File) {
		g.Expect(f.Name).NotTo(HaveSuffix("_test.go"),
			"exclude filter should remove test files, found: %s", f.Name)
	})

	// Verify non-test Go files still exist
	var goFiles int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension == "go" {
			goFiles++
		}
	})

	g.Expect(goFiles).To(BeNumerically(">", 5),
		"should still have non-test .go files")
}

// TestE2E_FilteredMetricsStillAggregate verifies that after filtering,
// aggregations work correctly on the reduced file set.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FilteredMetricsStillAggregate(t *testing.T) {
	g := NewGomegaWithT(t)

	// Exclude test files, leaving only production Go and other files
	rules := []filter.Rule{
		{Pattern: "**/*_test.go", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	// All aggregations should still work on the filtered set
	resolved := resolveAndAggregate(t, root, "file-lines.sum")
	v, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue(), "file-lines.sum should be set after filtering")
	g.Expect(v).To(BeNumerically(">", 0), "file-lines.sum should be >0 for filtered files")

	// file-type.mode on the remaining set should be non-empty
	modeResolved := resolveAndAggregate(t, root, "file-type.mode")
	modeVal, ok := root.Classification(modeResolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(modeVal).NotTo(BeEmpty(), "mode of file-type should be non-empty after filtering")
}

// TestE2E_MeanAggregation_Consistency verifies that mean is between min and max.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_MeanAggregation_Consistency(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	minResolved := resolveAndAggregate(t, root, "file-size.min")
	maxResolved := resolveAndAggregate(t, root, "file-size.max")
	meanResolved := resolveAndAggregate(t, root, "file-size.mean")

	minVal, _ := root.Quantity(minResolved.ResultName)
	maxVal, _ := root.Quantity(maxResolved.ResultName)
	meanVal, ok := root.Measure(meanResolved.ResultName)

	g.Expect(ok).To(BeTrue(), "mean should be set")
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)),
		"mean (%f) should be >= min (%d)", meanVal, minVal)
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)),
		"mean (%f) should be <= max (%d)", meanVal, maxVal)
}
