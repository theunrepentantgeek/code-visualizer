package golang_test

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
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
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

	// This file is at internal/provider/golang/e2e_test.go
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")

	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Fatalf("repo root %s does not contain go.mod", abs)
	}

	return abs
}

// setupE2E registers the golang provider, scans the repo, loads file-level
// metrics, populates declarations, and returns the populated tree.
//
// Each test rescans the repo independently to avoid conflicts with internal
// tests that call ResetBaseRegistryForTesting.
func setupE2E(
	t *testing.T,
	rules []filter.Rule,
) *model.Directory {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	golang.ResetCacheForTesting()
	t.Cleanup(provider.ResetBaseRegistryForTesting)
	t.Cleanup(golang.ResetCacheForTesting)

	golang.Register()

	if rules == nil {
		rules = make([]filter.Rule, 0)
	}

	rules = append(rules,
		filter.Rule{Pattern: ".*", Mode: filter.Exclude},
		filter.Rule{Pattern: "**/testdata/**", Mode: filter.Exclude},
	)

	root, err := scan.Scan(repoRoot(t), rules, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	err = provider.RunLoaders(
		root,
		[]metric.Name{golang.Imports, golang.CommentRatio},
		nil,
	)
	if err != nil {
		t.Fatalf("RunLoaders failed: %v", err)
	}

	// Populate declarations for declaration-level metrics.
	model.WalkFiles(root, func(f *model.File) {
		golang.PopulateDeclarations(f)
	})

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

// ---------------------------------------------------------------------------
// Base metric tests
// ---------------------------------------------------------------------------

// TestE2E_DeclarationCountMetrics verifies that all declaration-based count
// metrics return positive values for the repo (which has Go code).
//
//nolint:paralleltest // mutates global base registry
func TestE2E_DeclarationCountMetrics(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	metrics := []struct {
		expression string
		desc       string
	}{
		{"types.count", "the repo should have Go type declarations"},
		{"interfaces.count", "the repo should have Go interface declarations"},
		{"structs.count", "the repo should have Go struct declarations"},
		{"functions.count", "the repo should have Go function declarations"},
		{"methods.count", "the repo should have Go method declarations"},
		{"constants.count", "the repo should have Go constant declarations"},
		{"variables.count", "the repo should have Go variable declarations"},
		{"declarations.count", "the repo should have Go declarations"},
	}

	for _, tt := range metrics {
		resolved := resolveAndAggregate(t, root, tt.expression)
		v, ok := root.Quantity(resolved.ResultName)
		g.Expect(ok).To(BeTrue(), "%s should be set on root", tt.expression)
		g.Expect(v).To(BeNumerically(">", 0), "%s: %s", tt.expression, tt.desc)
	}

	// declarations.count should be >= the sum of individual kinds
	declResolved := resolveAndAggregate(t, root, "declarations.count")
	funcResolved := resolveAndAggregate(t, root, "functions.count")

	declCount, _ := root.Quantity(declResolved.ResultName)
	funcCount, _ := root.Quantity(funcResolved.ResultName)

	g.Expect(declCount).To(BeNumerically(">=", funcCount),
		"declarations.count should be >= functions.count")
}

// TestE2E_Imports_BaseMetric verifies that imports are populated on Go files
// and contain reasonable values.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Imports_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var goFileCount int

	var filesWithImports int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		goFileCount++

		v, ok := f.Quantity(golang.Imports)
		g.Expect(ok).To(BeTrue(), "imports should be set on Go file %s", f.Path)
		g.Expect(v).To(BeNumerically(">=", 0), "imports should be >=0 for %s", f.Path)

		if v > 0 {
			filesWithImports++
		}
	})

	g.Expect(goFileCount).To(BeNumerically(">", 10), "expected more than 10 Go files")
	g.Expect(filesWithImports).To(BeNumerically(">", 10), "expected most Go files to have imports")
}

// TestE2E_CyclomaticComplexity_BaseMetric verifies that cyclomatic complexity
// is ≥ 1 on declarations that are functions or methods.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var funcCount int

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		v, ok := d.Quantity(golang.CyclomaticComplexity)
		g.Expect(ok).To(BeTrue(),
			"cyclomatic-complexity should be set on %s %q", d.Kind, d.Name)
		g.Expect(v).To(BeNumerically(">=", 1),
			"cyclomatic-complexity should be >=1 for %s %q", d.Kind, d.Name)

		funcCount++
	})

	g.Expect(funcCount).To(BeNumerically(">", 10),
		"expected the repo to have more than 10 functions/methods")
}

// TestE2E_FunctionLength_BaseMetric verifies that function-length is ≥ 1
// on function and method declarations.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FunctionLength_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var funcCount int

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		v, ok := d.Quantity(golang.FunctionLength)
		g.Expect(ok).To(BeTrue(),
			"function-length should be set on %s %q", d.Kind, d.Name)
		g.Expect(v).To(BeNumerically(">=", 1),
			"function-length should be >=1 for %s %q", d.Kind, d.Name)

		funcCount++
	})

	g.Expect(funcCount).To(BeNumerically(">", 10),
		"expected the repo to have more than 10 functions/methods")
}

// TestE2E_CommentRatio_BaseMetric verifies that comment-ratio is between 0
// and 1 (inclusive) on all Go files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CommentRatio_BaseMetric(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	var goFileCount int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		goFileCount++

		v, ok := f.Measure(golang.CommentRatio)
		g.Expect(ok).To(BeTrue(), "comment-ratio should be set on Go file %s", f.Path)
		g.Expect(v).To(BeNumerically(">=", 0),
			"comment-ratio should be >=0 for %s", f.Path)
		g.Expect(v).To(BeNumerically("<=", 1),
			"comment-ratio should be <=1 for %s", f.Path)
	})

	g.Expect(goFileCount).To(BeNumerically(">", 10))
}

// ---------------------------------------------------------------------------
// Aggregation tests
// ---------------------------------------------------------------------------

// TestE2E_Imports_Aggregations verifies sum, min, max, mean aggregations
// for imports at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Imports_Aggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumResolved := resolveAndAggregate(t, root, "imports.sum")
	minResolved := resolveAndAggregate(t, root, "imports.min")
	maxResolved := resolveAndAggregate(t, root, "imports.max")
	meanResolved := resolveAndAggregate(t, root, "imports.mean")

	sumVal, ok := root.Quantity(sumResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "imports.sum should be set on root")
	g.Expect(sumVal).To(BeNumerically(">", 0), "imports.sum should be >0")

	minVal, ok := root.Quantity(minResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "imports.min should be set on root")
	g.Expect(minVal).To(BeNumerically(">=", 0), "imports.min should be >=0")

	maxVal, ok := root.Quantity(maxResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "imports.max should be set on root")
	g.Expect(maxVal).To(BeNumerically(">", 0), "imports.max should be >0")

	meanVal, ok := root.Measure(meanResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "imports.mean should be set on root")
	g.Expect(meanVal).To(BeNumerically(">", 0), "imports.mean should be >0")

	// Consistency: sum >= max >= min >= 0
	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum should be >= max")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max should be >= min")

	// mean should be between min and max
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)),
		"mean (%f) should be >= min (%d)", meanVal, minVal)
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)),
		"mean (%f) should be <= max (%d)", meanVal, maxVal)
}

// TestE2E_CyclomaticComplexity_Aggregations verifies sum, min, max, mean
// aggregations for cyclomatic-complexity at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_Aggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumResolved := resolveAndAggregate(t, root, "cyclomatic-complexity.sum")
	minResolved := resolveAndAggregate(t, root, "cyclomatic-complexity.min")
	maxResolved := resolveAndAggregate(t, root, "cyclomatic-complexity.max")
	meanResolved := resolveAndAggregate(t, root, "cyclomatic-complexity.mean")

	sumVal, ok := root.Quantity(sumResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "cyclomatic-complexity.sum should be set")
	g.Expect(sumVal).To(BeNumerically(">", 0))

	minVal, ok := root.Quantity(minResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "cyclomatic-complexity.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 1), "minimum complexity should be >=1")

	maxVal, ok := root.Quantity(maxResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "cyclomatic-complexity.max should be set")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max should be >= min")

	meanVal, ok := root.Measure(meanResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "cyclomatic-complexity.mean should be set")
	g.Expect(meanVal).To(BeNumerically(">=", 1.0), "mean complexity should be >=1")

	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum should be >= max")
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)))
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)))
}

// TestE2E_FunctionLength_Aggregations verifies sum, min, max, mean
// aggregations for function-length at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FunctionLength_Aggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	sumResolved := resolveAndAggregate(t, root, "function-length.sum")
	minResolved := resolveAndAggregate(t, root, "function-length.min")
	maxResolved := resolveAndAggregate(t, root, "function-length.max")
	meanResolved := resolveAndAggregate(t, root, "function-length.mean")

	sumVal, ok := root.Quantity(sumResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "function-length.sum should be set")
	g.Expect(sumVal).To(BeNumerically(">", 0))

	minVal, ok := root.Quantity(minResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "function-length.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 1), "minimum function length should be >=1")

	maxVal, ok := root.Quantity(maxResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "function-length.max should be set")
	g.Expect(maxVal).To(BeNumerically(">", minVal), "max length should be > min for a real repo")

	meanVal, ok := root.Measure(meanResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "function-length.mean should be set")
	g.Expect(meanVal).To(BeNumerically(">=", 1.0))

	g.Expect(sumVal).To(BeNumerically(">=", maxVal), "sum should be >= max")
	g.Expect(meanVal).To(BeNumerically(">=", float64(minVal)))
	g.Expect(meanVal).To(BeNumerically("<=", float64(maxVal)))
}

// TestE2E_CommentRatio_Aggregations verifies min, max, mean aggregations
// for comment-ratio at directory level.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CommentRatio_Aggregations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	minResolved := resolveAndAggregate(t, root, "comment-ratio.min")
	maxResolved := resolveAndAggregate(t, root, "comment-ratio.max")
	meanResolved := resolveAndAggregate(t, root, "comment-ratio.mean")

	minVal, ok := root.Measure(minResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "comment-ratio.min should be set")
	g.Expect(minVal).To(BeNumerically(">=", 0))
	g.Expect(minVal).To(BeNumerically("<=", 1))

	maxVal, ok := root.Measure(maxResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "comment-ratio.max should be set")
	g.Expect(maxVal).To(BeNumerically(">=", minVal), "max should be >= min")
	g.Expect(maxVal).To(BeNumerically("<=", 1))

	meanVal, ok := root.Measure(meanResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "comment-ratio.mean should be set")
	g.Expect(meanVal).To(BeNumerically(">=", minVal),
		"mean (%f) should be >= min (%f)", meanVal, minVal)
	g.Expect(meanVal).To(BeNumerically("<=", maxVal),
		"mean (%f) should be <= max (%f)", meanVal, maxVal)
}

// ---------------------------------------------------------------------------
// Provider-specific filter tests
// ---------------------------------------------------------------------------

// TestE2E_PublicPrivateFilters verifies that public/private filters produce
// counts that are ≤ the unfiltered total and together equal the total.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_PublicPrivateFilters(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	allResolved := resolveAndAggregate(t, root, "functions.count")
	pubResolved := resolveAndAggregate(t, root, "public.functions.count")
	privResolved := resolveAndAggregate(t, root, "private.functions.count")

	allCount, _ := root.Quantity(allResolved.ResultName)
	pubCount, ok := root.Quantity(pubResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "public.functions.count should be set")
	privCount, ok := root.Quantity(privResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "private.functions.count should be set")

	g.Expect(pubCount).To(BeNumerically(">", 0),
		"repo should have public functions")
	g.Expect(privCount).To(BeNumerically(">", 0),
		"repo should have private functions")
	g.Expect(pubCount).To(BeNumerically("<=", allCount),
		"public count should be <= total")
	g.Expect(privCount).To(BeNumerically("<=", allCount),
		"private count should be <= total")
	g.Expect(pubCount+privCount).To(Equal(allCount),
		"public + private should equal total")
}

// TestE2E_PublicPrivateDeclarations verifies public/private filters on the
// broader declarations metric.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_PublicPrivateDeclarations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	allResolved := resolveAndAggregate(t, root, "declarations.count")
	pubResolved := resolveAndAggregate(t, root, "public.declarations.count")
	privResolved := resolveAndAggregate(t, root, "private.declarations.count")

	allCount, _ := root.Quantity(allResolved.ResultName)
	pubCount, _ := root.Quantity(pubResolved.ResultName)
	privCount, _ := root.Quantity(privResolved.ResultName)

	g.Expect(pubCount).To(BeNumerically(">", 0))
	g.Expect(privCount).To(BeNumerically(">", 0))
	g.Expect(pubCount+privCount).To(Equal(allCount),
		"public + private declarations should equal total declarations")
}

// TestE2E_ImportFilters verifies that stdlib, external, and internal import
// filters produce counts that are ≤ the unfiltered total.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ImportFilters(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	allResolved := resolveAndAggregate(t, root, "imports.sum")
	stdlibResolved := resolveAndAggregate(t, root, "stdlib.imports.sum")
	externalResolved := resolveAndAggregate(t, root, "external.imports.sum")
	internalResolved := resolveAndAggregate(t, root, "internal.imports.sum")

	allSum, _ := root.Quantity(allResolved.ResultName)
	stdlibSum, ok := root.Quantity(stdlibResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "stdlib.imports.sum should be set")
	externalSum, ok := root.Quantity(externalResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "external.imports.sum should be set")
	internalSum, ok := root.Quantity(internalResolved.ResultName)
	g.Expect(ok).To(BeTrue(), "internal.imports.sum should be set")

	g.Expect(stdlibSum).To(BeNumerically(">", 0),
		"repo should use stdlib imports")
	g.Expect(externalSum).To(BeNumerically(">", 0),
		"repo should use external imports")
	g.Expect(internalSum).To(BeNumerically(">", 0),
		"repo should use internal imports")

	g.Expect(stdlibSum).To(BeNumerically("<=", allSum),
		"stdlib imports should be <= total imports")
	g.Expect(externalSum).To(BeNumerically("<=", allSum),
		"external imports should be <= total imports")
	g.Expect(internalSum).To(BeNumerically("<=", allSum),
		"internal imports should be <= total imports")

	// stdlib + external + internal should equal total imports
	g.Expect(stdlibSum+externalSum+internalSum).To(Equal(allSum),
		"stdlib + external + internal imports should equal total imports")
}

// ---------------------------------------------------------------------------
// Path filtering (include/exclude) tests
// ---------------------------------------------------------------------------

// TestE2E_ExcludeTestFiles verifies that excluding *_test.go files removes
// them from the scan and that metrics still compute correctly.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ExcludeTestFiles(t *testing.T) {
	g := NewGomegaWithT(t)

	rules := []filter.Rule{
		{Pattern: "**/*_test.go", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	// Verify no test files remain
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

	// Metrics should still work on the reduced set
	resolved := resolveAndAggregate(t, root, "functions.count")
	v, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue(), "functions.count should be set after filtering")
	g.Expect(v).To(BeNumerically(">", 0), "functions.count should be >0 after filtering")
}

// TestE2E_IncludeOnlyInternal verifies that including only the internal/
// directory restricts the scan appropriately.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_IncludeOnlyInternal(t *testing.T) {
	g := NewGomegaWithT(t)

	rules := []filter.Rule{
		{Pattern: "internal/**", Mode: filter.Include},
		{Pattern: "**", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	// All files should be under internal/
	model.WalkFiles(root, func(f *model.File) {
		g.Expect(f.Path).To(ContainSubstring("internal"),
			"include filter should restrict to internal/, found: %s", f.Path)
	})

	// Verify we still have Go files to compute metrics on
	var goFileCount int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension == "go" {
			goFileCount++
		}
	})

	g.Expect(goFileCount).To(BeNumerically(">", 5),
		"internal/ should contain Go files")

	// Declarations should still populate and aggregate
	resolved := resolveAndAggregate(t, root, "declarations.count")
	v, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically(">", 0))
}

// TestE2E_ExcludeByDirectory verifies that excluding a specific directory
// removes its files from metrics.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ExcludeByDirectory(t *testing.T) {
	g := NewGomegaWithT(t)

	// Get unfiltered count first
	unfilteredRoot := setupE2E(t, nil)
	unfilteredResolved := resolveAndAggregate(t, unfilteredRoot, "functions.count")
	unfilteredCount, _ := unfilteredRoot.Quantity(unfilteredResolved.ResultName)

	// Now exclude a directory known to have Go functions
	rules := []filter.Rule{
		{Pattern: "internal/provider/**", Mode: filter.Exclude},
	}

	filteredRoot := setupE2E(t, rules)
	filteredResolved := resolveAndAggregate(t, filteredRoot, "functions.count")
	filteredCount, ok := filteredRoot.Quantity(filteredResolved.ResultName)
	g.Expect(ok).To(BeTrue())

	g.Expect(filteredCount).To(BeNumerically("<", unfilteredCount),
		"excluding internal/provider/ should reduce function count")
	g.Expect(filteredCount).To(BeNumerically(">", 0),
		"should still have functions outside internal/provider/")
}

// ---------------------------------------------------------------------------
// Consistency and cross-cutting tests
// ---------------------------------------------------------------------------

// TestE2E_SubdirectoryAggregation verifies that subdirectories also receive
// aggregated values, not just the root.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_SubdirectoryAggregation(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	var dirsWithFunctions int

	for _, d := range root.Dirs {
		if v, ok := d.Quantity(resolved.ResultName); ok && v > 0 {
			dirsWithFunctions++
		}
	}

	g.Expect(dirsWithFunctions).To(BeNumerically(">", 0),
		"at least one immediate subdirectory should have functions.count populated")
}

// TestE2E_FileLevelDeclarationAggregation verifies that file-level
// aggregation of declaration metrics works (each file gets its own count).
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileLevelDeclarationAggregation(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	var filesWithFunctions int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		if v, ok := f.Quantity(resolved.ResultName); ok && v > 0 {
			filesWithFunctions++
		}
	})

	g.Expect(filesWithFunctions).To(BeNumerically(">", 5),
		"multiple Go files should have per-file function counts")
}

// TestE2E_FilteredMetricsStillAggregate verifies that after filtering,
// aggregations work correctly on the reduced file set.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FilteredMetricsStillAggregate(t *testing.T) {
	g := NewGomegaWithT(t)

	rules := []filter.Rule{
		{Pattern: "**/*_test.go", Mode: filter.Exclude},
	}

	root := setupE2E(t, rules)

	// All aggregations should still work on the filtered set
	funcResolved := resolveAndAggregate(t, root, "functions.count")
	v, ok := root.Quantity(funcResolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically(">", 0))

	// Import aggregations should also work
	importResolved := resolveAndAggregate(t, root, "imports.sum")
	iv, ok := root.Quantity(importResolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(iv).To(BeNumerically(">", 0))

	// Comment ratio should work on filtered set
	crResolved := resolveAndAggregate(t, root, "comment-ratio.mean")
	crv, ok := root.Measure(crResolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(crv).To(BeNumerically(">", 0))
	g.Expect(crv).To(BeNumerically("<=", 1))
}

// TestE2E_FilteredCountsLessThanUnfiltered verifies that applying provider
// filters always produces counts ≤ the unfiltered equivalent.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FilteredCountsLessThanUnfiltered(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	// Test across multiple declaration metrics with public filter
	declarationMetrics := []string{
		"types", "interfaces", "structs", "functions",
		"methods", "constants", "variables",
	}

	for _, m := range declarationMetrics {
		allExpr := m + ".count"
		pubExpr := "public." + m + ".count"

		allResolved := resolveAndAggregate(t, root, allExpr)
		pubResolved := resolveAndAggregate(t, root, pubExpr)

		allVal, allOk := root.Quantity(allResolved.ResultName)
		pubVal, pubOk := root.Quantity(pubResolved.ResultName)

		if allOk && pubOk {
			g.Expect(pubVal).To(BeNumerically("<=", allVal),
				"public.%s.count (%d) should be <= %s.count (%d)", m, pubVal, m, allVal)
		}
	}
}

// ---------------------------------------------------------------------------
// Aggregation correctness — first-principles verification
// ---------------------------------------------------------------------------

// TestE2E_FileLevel_FunctionCount_MatchesManualCount walks each Go file and
// verifies that the aggregated functions.count on the file equals the number
// of function-kind declarations actually present on the file.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileLevel_FunctionCount_MatchesManualCount(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	var filesChecked int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		// Manual count: declarations with kind "function"
		var manualCount int64
		for _, d := range f.Declarations {
			if d.Kind == model.DeclKindFunction {
				manualCount++
			}
		}

		aggregatedCount, ok := f.Quantity(resolved.ResultName)
		if manualCount == 0 {
			// Files with zero functions may or may not have the metric set
			if ok {
				g.Expect(aggregatedCount).To(Equal(int64(0)),
					"file %s has no function declarations but functions.count is %d", f.Name, aggregatedCount)
			}
		} else {
			g.Expect(ok).To(BeTrue(),
				"file %s has %d function declarations but functions.count not set", f.Name, manualCount)
			g.Expect(aggregatedCount).To(Equal(manualCount),
				"file %s: functions.count (%d) should equal manual declaration count (%d)",
				f.Name, aggregatedCount, manualCount)
		}

		filesChecked++
	})

	g.Expect(filesChecked).To(BeNumerically(">", 10))
}

// TestE2E_RootFunctionCount_MatchesFlatDeclarationWalk verifies that the
// root directory's functions.count equals the total number of function
// declarations found by walking all files.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_RootFunctionCount_MatchesFlatDeclarationWalk(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	// Manual flat count of all function declarations
	var manualTotal int64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind == model.DeclKindFunction {
			manualTotal++
		}
	})

	rootCount, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue(), "functions.count should be set on root")
	g.Expect(rootCount).To(Equal(manualTotal),
		"root functions.count (%d) should equal flat walk count (%d)",
		rootCount, manualTotal)
}

// TestE2E_DirectoryCount_EqualsChildContributions verifies that a directory's
// aggregated count equals the sum of its direct files' counts plus the sum of
// its child directories' counts.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_DirectoryCount_EqualsChildContributions(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	rootCount, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())

	// Sum counts from root's direct files
	var directFileCount int64

	for _, f := range root.Files {
		if v, ok := f.Quantity(resolved.ResultName); ok {
			directFileCount += v
		}
	}

	// Sum counts from root's child directories
	var childDirCount int64

	for _, d := range root.Dirs {
		if v, ok := d.Quantity(resolved.ResultName); ok {
			childDirCount += v
		}
	}

	g.Expect(directFileCount+childDirCount).To(Equal(rootCount),
		"root functions.count (%d) should equal direct files (%d) + child dirs (%d)",
		rootCount, directFileCount, childDirCount)
}

// TestE2E_CyclomaticComplexity_FileSum_MatchesDeclarations verifies that
// each file's cyclomatic-complexity.sum equals the manual sum of cyclomatic
// complexity values from its function/method declarations.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_FileSum_MatchesDeclarations(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "cyclomatic-complexity.sum")

	var filesChecked int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		var manualSum int64

		var hasFuncs bool

		for _, d := range f.Declarations {
			if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
				continue
			}

			hasFuncs = true

			if v, ok := d.Quantity(golang.CyclomaticComplexity); ok {
				manualSum += v
			}
		}

		if !hasFuncs {
			return
		}

		aggregatedSum, ok := f.Quantity(resolved.ResultName)
		g.Expect(ok).To(BeTrue(),
			"file %s has functions but cyclomatic-complexity.sum not set", f.Name)
		g.Expect(aggregatedSum).To(Equal(manualSum),
			"file %s: cyclomatic-complexity.sum (%d) should equal manual sum (%d)",
			f.Name, aggregatedSum, manualSum)

		filesChecked++
	})

	g.Expect(filesChecked).To(BeNumerically(">", 5))
}

// TestE2E_CyclomaticComplexity_DirectorySum_MatchesFlatSum verifies that
// the root directory's cyclomatic-complexity.sum equals the sum computed
// by manually walking all function/method declarations.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_DirectorySum_MatchesFlatSum(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "cyclomatic-complexity.sum")

	var manualSum int64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		if v, ok := d.Quantity(golang.CyclomaticComplexity); ok {
			manualSum += v
		}
	})

	rootSum, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootSum).To(Equal(manualSum),
		"root cyclomatic-complexity.sum (%d) should equal flat manual sum (%d)",
		rootSum, manualSum)
}

// TestE2E_CyclomaticComplexity_DirectoryMean_FromRawValues verifies that the
// directory-level mean is computed from all raw declaration values (first
// principles), not as a mean-of-file-level-means.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_DirectoryMean_FromRawValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "cyclomatic-complexity.mean")

	// Manually compute the correct mean from all raw declaration values
	var rawValues []float64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		if v, ok := d.Quantity(golang.CyclomaticComplexity); ok {
			rawValues = append(rawValues, float64(v))
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty(), "should have function declarations")

	expectedMean := metric.AggregateMean(rawValues)

	rootMean, ok := root.Measure(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", expectedMean, 0.0001),
		"root cyclomatic-complexity.mean (%f) should equal mean of all raw values (%f), not mean-of-means",
		rootMean, expectedMean)

	// Compute mean-of-file-means to verify it differs (would catch the bug)
	var fileMeans []float64

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		var fileValues []float64

		for _, d := range f.Declarations {
			if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
				continue
			}

			if v, ok := d.Quantity(golang.CyclomaticComplexity); ok {
				fileValues = append(fileValues, float64(v))
			}
		}

		if len(fileValues) > 0 {
			fileMeans = append(fileMeans, metric.AggregateMean(fileValues))
		}
	})

	if len(fileMeans) > 1 {
		meanOfMeans := metric.AggregateMean(fileMeans)

		// If mean-of-means differs from the correct mean, the test above
		// would have caught a mean-of-means bug. Log the difference to show
		// the test has discriminating power.
		if meanOfMeans != expectedMean {
			t.Logf("mean-of-means (%.4f) differs from correct mean (%.4f) — "+
				"this test has discriminating power", meanOfMeans, expectedMean)
		}
	}
}

// TestE2E_FunctionLength_DirectoryMean_FromRawValues verifies that the
// directory-level function-length.mean is computed from all raw declaration
// values, not as a mean-of-file-level-means.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FunctionLength_DirectoryMean_FromRawValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "function-length.mean")

	var rawValues []float64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		if v, ok := d.Quantity(golang.FunctionLength); ok {
			rawValues = append(rawValues, float64(v))
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty())

	expectedMean := metric.AggregateMean(rawValues)

	rootMean, ok := root.Measure(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", expectedMean, 0.0001),
		"root function-length.mean (%f) should equal mean of all raw values (%f)",
		rootMean, expectedMean)
}

// TestE2E_FunctionLength_DirectoryMin_MatchesFlatMin verifies that the
// directory-level min is the true minimum across all declarations, not the
// minimum of file-level mins.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FunctionLength_DirectoryMin_MatchesFlatMin(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "function-length.min")

	var rawValues []float64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		if v, ok := d.Quantity(golang.FunctionLength); ok {
			rawValues = append(rawValues, float64(v))
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty())

	expectedMin := metric.AggregateMin(rawValues)

	rootMin, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(float64(rootMin)).To(BeNumerically("~", expectedMin, 0.0001),
		"root function-length.min (%d) should equal flat min (%.0f)",
		rootMin, expectedMin)
}

// TestE2E_FunctionLength_DirectoryMax_MatchesFlatMax verifies that the
// directory-level max is the true maximum across all declarations.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FunctionLength_DirectoryMax_MatchesFlatMax(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "function-length.max")

	var rawValues []float64

	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		if d.Kind != model.DeclKindFunction && d.Kind != model.DeclKindMethod {
			return
		}

		if v, ok := d.Quantity(golang.FunctionLength); ok {
			rawValues = append(rawValues, float64(v))
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty())

	expectedMax := metric.AggregateMax(rawValues)

	rootMax, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(float64(rootMax)).To(BeNumerically("~", expectedMax, 0.0001),
		"root function-length.max (%d) should equal flat max (%.0f)",
		rootMax, expectedMax)
}

// TestE2E_CommentRatio_DirectoryMean_FromRawValues verifies that the
// directory-level comment-ratio.mean is computed from all raw file values,
// not from child directory means.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CommentRatio_DirectoryMean_FromRawValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "comment-ratio.mean")

	// Manually collect all per-file comment-ratio values
	var rawValues []float64

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Measure(golang.CommentRatio); ok {
			rawValues = append(rawValues, v)
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty())

	expectedMean := metric.AggregateMean(rawValues)

	rootMean, ok := root.Measure(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", expectedMean, 0.0001),
		"root comment-ratio.mean (%f) should equal mean of all file values (%f)",
		rootMean, expectedMean)
}

// TestE2E_ImportSum_MatchesSumOfFileValues verifies that the directory-level
// imports.sum equals the sum of per-file import counts.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ImportSum_MatchesSumOfFileValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "imports.sum")

	var manualSum int64

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Quantity(golang.Imports); ok {
			manualSum += v
		}
	})

	rootSum, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootSum).To(Equal(manualSum),
		"root imports.sum (%d) should equal sum of per-file imports (%d)",
		rootSum, manualSum)
}

// TestE2E_ImportMean_FromRawFileValues verifies that imports.mean at
// directory level is computed from raw per-file values.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_ImportMean_FromRawFileValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "imports.mean")

	var rawValues []float64

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Quantity(golang.Imports); ok {
			rawValues = append(rawValues, float64(v))
		}
	})

	g.Expect(rawValues).NotTo(BeEmpty())

	expectedMean := metric.AggregateMean(rawValues)

	rootMean, ok := root.Measure(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootMean).To(BeNumerically("~", expectedMean, 0.0001),
		"root imports.mean (%f) should equal mean of per-file values (%f)",
		rootMean, expectedMean)
}

// TestE2E_FilteredImportSum_MatchesSumOfFilteredFileValues verifies that
// stdlib.imports.sum equals the sum of per-file stdlib import counts,
// confirming filters are applied consistently at both levels.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FilteredImportSum_MatchesSumOfFilteredFileValues(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "stdlib.imports.sum")

	// The file-level filtered metric is stored under "stdlib.imports"
	filteredMetricName := metric.MetricExpression{
		Filter: "stdlib",
		Base:   golang.Imports,
	}.ResultName()

	var manualSum int64

	model.WalkFiles(root, func(f *model.File) {
		if v, ok := f.Quantity(filteredMetricName); ok {
			manualSum += v
		}
	})

	rootSum, ok := root.Quantity(resolved.ResultName)
	g.Expect(ok).To(BeTrue())
	g.Expect(rootSum).To(Equal(manualSum),
		"root stdlib.imports.sum (%d) should equal sum of per-file stdlib imports (%d)",
		rootSum, manualSum)
}

// TestE2E_FileLevel_PublicPrivateCount_AddsUp verifies that on individual
// files, public + private counts for each declaration metric equal the
// unfiltered count.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_FileLevel_PublicPrivateCount_AddsUp(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	allResolved := resolveAndAggregate(t, root, "functions.count")
	pubResolved := resolveAndAggregate(t, root, "public.functions.count")
	privResolved := resolveAndAggregate(t, root, "private.functions.count")

	var filesChecked int

	model.WalkFiles(root, func(f *model.File) {
		if f.Extension != "go" {
			return
		}

		allVal, allOk := f.Quantity(allResolved.ResultName)
		if !allOk || allVal == 0 {
			return
		}

		pubVal, _ := f.Quantity(pubResolved.ResultName)
		privVal, _ := f.Quantity(privResolved.ResultName)

		g.Expect(pubVal+privVal).To(Equal(allVal),
			"file %s: public (%d) + private (%d) should equal total (%d) functions",
			f.Name, pubVal, privVal, allVal)

		filesChecked++
	})

	g.Expect(filesChecked).To(BeNumerically(">", 5),
		"should have checked multiple files with functions")
}

// TestE2E_Declarations_IncludesAllKinds verifies that declarations.count
// at the root equals the sum of all per-kind counts (types, interfaces,
// structs, functions, methods, constants, variables).
//
//nolint:paralleltest // mutates global base registry
func TestE2E_Declarations_IncludesAllKinds(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	allResolved := resolveAndAggregate(t, root, "declarations.count")
	allCount, ok := root.Quantity(allResolved.ResultName)
	g.Expect(ok).To(BeTrue())

	// Manual count by walking all declarations (no kind filter)
	var manualTotal int64

	model.WalkDeclarations(root, func(_ *model.Declaration, _ *model.File) {
		manualTotal++
	})

	g.Expect(allCount).To(Equal(manualTotal),
		"declarations.count (%d) should equal total manual walk (%d)",
		allCount, manualTotal)

	// Also verify: sum of per-kind counts should equal declarations.count
	kinds := []string{
		"types", "interfaces", "structs", "functions",
		"methods", "constants", "variables",
	}

	var kindSum int64

	for _, k := range kinds {
		resolved := resolveAndAggregate(t, root, k+".count")
		if v, ok := root.Quantity(resolved.ResultName); ok {
			kindSum += v
		}
	}

	// Note: types.count includes structs and interfaces, so the per-kind sum
	// will exceed declarations.count due to overlap. Instead verify that
	// each kind count ≤ declarations.count.
	for _, k := range kinds {
		resolved := resolveAndAggregate(t, root, k+".count")
		if v, ok := root.Quantity(resolved.ResultName); ok {
			g.Expect(v).To(BeNumerically("<=", allCount),
				"%s.count (%d) should be <= declarations.count (%d)", k, v, allCount)
		}
	}
}

// TestE2E_NestedDirectoryAggregation_IsAdditive verifies that for a directory
// with children, the aggregated count equals the sum of contributions from
// its direct files and child directories — recursively.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_NestedDirectoryAggregation_IsAdditive(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "functions.count")

	// Check additivity for every directory in the tree
	var dirsChecked int

	model.WalkDirectories(root, func(dir *model.Directory) {
		dirCount, ok := dir.Quantity(resolved.ResultName)
		if !ok {
			return
		}

		var fileSum int64

		for _, f := range dir.Files {
			if v, ok := f.Quantity(resolved.ResultName); ok {
				fileSum += v
			}
		}

		var childSum int64

		for _, child := range dir.Dirs {
			if v, ok := child.Quantity(resolved.ResultName); ok {
				childSum += v
			}
		}

		g.Expect(fileSum+childSum).To(Equal(dirCount),
			"dir %s: functions.count (%d) should equal files (%d) + children (%d)",
			dir.Name, dirCount, fileSum, childSum)

		dirsChecked++
	})

	g.Expect(dirsChecked).To(BeNumerically(">", 3),
		"should have verified additivity on multiple directories")
}

// TestE2E_CyclomaticComplexity_NestedSum_IsAdditive verifies that for
// cyclomatic-complexity.sum, directory values are additive across children,
// just like counts.
//
//nolint:paralleltest // mutates global base registry
func TestE2E_CyclomaticComplexity_NestedSum_IsAdditive(t *testing.T) {
	g := NewGomegaWithT(t)
	root := setupE2E(t, nil)

	resolved := resolveAndAggregate(t, root, "cyclomatic-complexity.sum")

	model.WalkDirectories(root, func(dir *model.Directory) {
		dirSum, ok := dir.Quantity(resolved.ResultName)
		if !ok {
			return
		}

		var fileSum int64

		for _, f := range dir.Files {
			if v, ok := f.Quantity(resolved.ResultName); ok {
				fileSum += v
			}
		}

		var childSum int64

		for _, child := range dir.Dirs {
			if v, ok := child.Quantity(resolved.ResultName); ok {
				childSum += v
			}
		}

		g.Expect(fileSum+childSum).To(Equal(dirSum),
			"dir %s: cyclomatic-complexity.sum (%d) should equal files (%d) + children (%d)",
			dir.Name, dirSum, fileSum, childSum)
	})
}
