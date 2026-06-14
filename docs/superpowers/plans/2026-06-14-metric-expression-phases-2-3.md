# Metric Expression Phases 2 & 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable declaration-level and commit-level metric expressions (e.g., `public.methods.count`, `mean.cyclomatic-complexity`) to compute values end-to-end by adding model types and extending the aggregation pipeline.

**Architecture:** Add `Declaration` and `Commit` model types to `File`, extend the Go provider to populate per-declaration data, extend the git provider to populate per-commit data, add `WalkDeclarations`/`WalkCommits` helpers, add `FilterFunc` for filter evaluation, and update `ComputeAggregations` to handle `LevelDeclaration` and `LevelCommit` sources. The aggregation stage already works for `LevelFile` → `LevelDirectory`; this extends it to handle `LevelDeclaration` → `LevelFile`/`LevelDirectory` and `LevelCommit` → `LevelFile`/`LevelDirectory`.

**Tech Stack:** Go 1.26+, dave/dst (Go AST), go-git, eris, Gomega, Goldie v2

---

## File Structure

### New files
| File | Responsibility |
|------|---------------|
| `internal/model/declaration.go` | `Declaration` struct with `MetricContainer` |
| `internal/model/declaration_test.go` | Tests for declaration |
| `internal/model/commit.go` | `Commit` struct with `MetricContainer` |
| `internal/model/commit_test.go` | Tests for commit |
| `internal/provider/filter.go` | `FilterFunc` type + registration on `BaseMetricDescriptor` |
| `internal/provider/filter_test.go` | Filter predicate tests |
| `internal/provider/golang/declarations.go` | Per-declaration extraction logic (populates `File.Declarations`) |
| `internal/provider/golang/declarations_test.go` | Tests for per-declaration data |
| `internal/provider/git/commits.go` | Per-commit extraction logic (populates `File.Commits`) |
| `internal/provider/git/commits_test.go` | Tests for per-commit data |

### Modified files
| File | Change |
|------|--------|
| `internal/model/file.go` | Add `Declarations []*Declaration` and `Commits []*Commit` fields |
| `internal/model/walk.go` | Add `WalkDeclarations` and `WalkCommits` |
| `internal/model/walk_test.go` | Tests for new walk functions |
| `internal/provider/base_descriptor.go` | Add `FilterFunc` field to `BaseMetricDescriptor` |
| `internal/stages/aggregation.go` | Handle `LevelDeclaration` and `LevelCommit` source levels |
| `internal/stages/aggregation_test.go` | New tests for declaration/commit aggregation |
| `internal/provider/golang/base_metrics.go` | Register `FilterFunc` on declaration-level descriptors |
| `internal/provider/golang/go_provider.go` | Call declaration extraction during file analysis |
| `internal/provider/git/base_metrics.go` | Add commit-level base metrics + register `FilterFunc` |
| `internal/provider/git/git_provider.go` | Call commit extraction during file analysis |
| `internal/stages/providers.go` | Run declaration/commit providers (populate sub-file data) |

---

## Task 1: Declaration Model Type

**Files:**
- Create: `internal/model/declaration.go`
- Create: `internal/model/declaration_test.go`

- [ ] **Step 1: Write the test**

```go
// internal/model/declaration_test.go
package model_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestDeclaration_SetAndGetMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "HandleRequest",
		Kind:       "function",
		Visibility: "public",
	}

	d.SetQuantity("cyclomatic-complexity", 5)
	v, ok := d.Quantity("cyclomatic-complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(5)))
}

func TestDeclaration_VisibilityField(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "handleRequest",
		Kind:       "function",
		Visibility: "private",
	}

	g.Expect(d.Visibility).To(Equal("private"))
	g.Expect(d.Kind).To(Equal("function"))
}

func TestDeclaration_MatchesFilter_Public(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	d := &model.Declaration{
		Name:       "HandleRequest",
		Kind:       "function",
		Visibility: "public",
	}

	g.Expect(d.MatchesFilter(metric.FilterName("public"))).To(BeTrue())
	g.Expect(d.MatchesFilter(metric.FilterName("private"))).To(BeFalse())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `task test`
Expected: compilation error — `model.Declaration` undefined

- [ ] **Step 3: Write the implementation**

```go
// internal/model/declaration.go
package model

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

// Declaration represents a single named declaration within a Go file.
type Declaration struct {
	MetricContainer
	Name       string // e.g., "HandleRequest", "UserService"
	Kind       string // e.g., "function", "method", "interface", "struct"
	Visibility string // "public" or "private"
}

// MatchesFilter reports whether this declaration passes the named filter.
func (d *Declaration) MatchesFilter(filter metric.FilterName) bool {
	switch filter {
	case "public":
		return d.Visibility == "public"
	case "private":
		return d.Visibility == "private"
	default:
		return false
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/declaration.go internal/model/declaration_test.go
git commit -m "feat: add Declaration model type with MetricContainer and filter matching"
```

---

## Task 2: Commit Model Type

**Files:**
- Create: `internal/model/commit.go`
- Create: `internal/model/commit_test.go`

- [ ] **Step 1: Write the test**

```go
// internal/model/commit_test.go
package model_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestCommit_SetAndGetMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &model.Commit{
		Hash:   "abc123",
		Author: "dev@example.com",
		Date:   time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	c.SetQuantity("lines-added", 42)
	v, ok := c.Quantity("lines-added")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(42)))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `task test`
Expected: compilation error — `model.Commit` undefined

- [ ] **Step 3: Write the implementation**

```go
// internal/model/commit.go
package model

import "time"

// Commit represents a single git commit that touched a file.
type Commit struct {
	MetricContainer
	Hash   string
	Author string
	Date   time.Time
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/model/commit.go internal/model/commit_test.go
git commit -m "feat: add Commit model type with MetricContainer"
```

---

## Task 3: Extend File + Walk Helpers

**Files:**
- Modify: `internal/model/file.go`
- Modify: `internal/model/walk.go`
- Modify: `internal/model/walk_test.go`

- [ ] **Step 1: Write tests for WalkDeclarations and WalkCommits**

Add to `internal/model/walk_test.go`:

```go
func TestWalkDeclarations_CollectsAllDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	decl1 := &model.Declaration{Name: "Foo", Kind: "function", Visibility: "public"}
	decl2 := &model.Declaration{Name: "bar", Kind: "method", Visibility: "private"}
	decl3 := &model.Declaration{Name: "Baz", Kind: "struct", Visibility: "public"}

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{decl1}},
		},
		Dirs: []*model.Directory{
			{
				Name: "sub",
				Files: []*model.File{
					{Name: "b.go", Declarations: []*model.Declaration{decl2, decl3}},
				},
			},
		},
	}

	var names []string
	model.WalkDeclarations(root, func(d *model.Declaration, _ *model.File) {
		names = append(names, d.Name)
	})

	g.Expect(names).To(ConsistOf("Foo", "bar", "Baz"))
}

func TestWalkCommits_CollectsAllDescendants(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &model.Commit{Hash: "aaa"}
	c2 := &model.Commit{Hash: "bbb"}

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Commits: []*model.Commit{c1}},
		},
		Dirs: []*model.Directory{
			{
				Name: "sub",
				Files: []*model.File{
					{Name: "b.go", Commits: []*model.Commit{c2}},
				},
			},
		},
	}

	var hashes []string
	model.WalkCommits(root, func(c *model.Commit, _ *model.File) {
		hashes = append(hashes, c.Hash)
	})

	g.Expect(hashes).To(ConsistOf("aaa", "bbb"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `task test`
Expected: compilation error — `File.Declarations` and `WalkDeclarations` undefined

- [ ] **Step 3: Extend File struct**

In `internal/model/file.go`, add the fields:

```go
type File struct {
	MetricContainer
	Path         string
	Name         string
	Extension    string
	IsBinary     bool
	Declarations []*Declaration
	Commits      []*Commit
}
```

- [ ] **Step 4: Add WalkDeclarations and WalkCommits to walk.go**

```go
// WalkDeclarations calls fn for every declaration in every file in the tree.
func WalkDeclarations(dir *Directory, fn func(*Declaration, *File)) {
	WalkFiles(dir, func(f *File) {
		for _, d := range f.Declarations {
			fn(d, f)
		}
	})
}

// WalkCommits calls fn for every commit record in every file in the tree.
func WalkCommits(dir *Directory, fn func(*Commit, *File)) {
	WalkFiles(dir, func(f *File) {
		for _, c := range f.Commits {
			fn(c, f)
		}
	})
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/model/file.go internal/model/walk.go internal/model/walk_test.go
git commit -m "feat: add Declarations/Commits to File and WalkDeclarations/WalkCommits"
```

---

## Task 4: FilterFunc on BaseMetricDescriptor

**Files:**
- Modify: `internal/provider/base_descriptor.go`
- Create: `internal/provider/filter_test.go`

- [ ] **Step 1: Write the test**

```go
// internal/provider/filter_test.go
package provider_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestBaseMetricDescriptor_FilterFunc_Nil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	desc := provider.BaseMetricDescriptor{
		Name: "file-size",
		Kind: metric.Quantity,
	}

	// When no FilterFunc is set, PassesFilter returns true (no filtering)
	g.Expect(desc.PassesFilter("anything", &model.Declaration{})).To(BeTrue())
}

func TestBaseMetricDescriptor_FilterFunc_Applied(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	desc := provider.BaseMetricDescriptor{
		Name:    "methods",
		Kind:    metric.Quantity,
		Filters: []metric.FilterName{"public", "private"},
		FilterFunc: func(filter metric.FilterName, node any) bool {
			d, ok := node.(*model.Declaration)
			if !ok {
				return false
			}
			return d.MatchesFilter(filter)
		},
	}

	pub := &model.Declaration{Name: "Foo", Visibility: "public"}
	priv := &model.Declaration{Name: "bar", Visibility: "private"}

	g.Expect(desc.PassesFilter("public", pub)).To(BeTrue())
	g.Expect(desc.PassesFilter("public", priv)).To(BeFalse())
	g.Expect(desc.PassesFilter("private", priv)).To(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `task test`
Expected: compilation error — `BaseMetricDescriptor.FilterFunc` and `PassesFilter` undefined

- [ ] **Step 3: Add FilterFunc field and PassesFilter method**

In `internal/provider/base_descriptor.go`, add to `BaseMetricDescriptor`:

```go
type BaseMetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Level          metric.MetricLevel
	Description    string
	Filters        []metric.FilterName
	Aggregations   []metric.AggregationName
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
	FilterFunc     func(filter metric.FilterName, node any) bool
}

// PassesFilter evaluates the FilterFunc for the given filter and node.
// Returns true if no FilterFunc is registered (no filtering applied).
func (d BaseMetricDescriptor) PassesFilter(filter metric.FilterName, node any) bool {
	if d.FilterFunc == nil {
		return true
	}

	return d.FilterFunc(filter, node)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/base_descriptor.go internal/provider/filter_test.go
git commit -m "feat: add FilterFunc to BaseMetricDescriptor for predicate evaluation"
```

---

## Task 5: Go Provider — Populate Declarations on File

**Files:**
- Create: `internal/provider/golang/declarations.go`
- Create: `internal/provider/golang/declarations_test.go`
- Modify: `internal/provider/golang/go_provider.go`

- [ ] **Step 1: Write the test**

```go
// internal/provider/golang/declarations_test.go
package golang_test

import (
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

func TestPopulateDeclarations_FunctionAndMethod(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use testdata fixture
	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata")
	filePath := filepath.Join(testdataDir, "sample.go")

	f := &model.File{
		Path:      filePath,
		Name:      "sample.go",
		Extension: "go",
	}

	golang.PopulateDeclarations(f)

	g.Expect(f.Declarations).NotTo(BeEmpty())

	// Check that we have at least one function and one method
	var hasFunc, hasMethod bool
	for _, d := range f.Declarations {
		if d.Kind == "function" {
			hasFunc = true
		}
		if d.Kind == "method" {
			hasMethod = true
		}
	}

	g.Expect(hasFunc).To(BeTrue(), "expected at least one function declaration")
	g.Expect(hasMethod).To(BeTrue(), "expected at least one method declaration")
}

func TestPopulateDeclarations_Visibility(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata")
	filePath := filepath.Join(testdataDir, "sample.go")

	f := &model.File{
		Path:      filePath,
		Name:      "sample.go",
		Extension: "go",
	}

	golang.PopulateDeclarations(f)

	var publicNames, privateNames []string
	for _, d := range f.Declarations {
		if d.Visibility == "public" {
			publicNames = append(publicNames, d.Name)
		} else {
			privateNames = append(privateNames, d.Name)
		}
	}

	g.Expect(publicNames).To(ContainElement("PublicFunc"))
	g.Expect(privateNames).To(ContainElement("privateFunc"))
}

func TestPopulateDeclarations_CyclomaticComplexity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata")
	filePath := filepath.Join(testdataDir, "sample.go")

	f := &model.File{
		Path:      filePath,
		Name:      "sample.go",
		Extension: "go",
	}

	golang.PopulateDeclarations(f)

	// Find a function and check it has cyclomatic-complexity set
	for _, d := range f.Declarations {
		if d.Kind == "function" || d.Kind == "method" {
			_, ok := d.Quantity(metric.Name("cyclomatic-complexity"))
			g.Expect(ok).To(BeTrue(), "expected cyclomatic-complexity on %s", d.Name)
		}
	}
}

func TestPopulateDeclarations_TypesAndStructs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, thisFile, _, _ := runtime.Caller(0)
	testdataDir := filepath.Join(filepath.Dir(thisFile), "testdata")
	filePath := filepath.Join(testdataDir, "sample.go")

	f := &model.File{
		Path:      filePath,
		Name:      "sample.go",
		Extension: "go",
	}

	golang.PopulateDeclarations(f)

	var kinds []string
	for _, d := range f.Declarations {
		kinds = append(kinds, d.Kind)
	}

	g.Expect(kinds).To(ContainElement("struct"))
	g.Expect(kinds).To(ContainElement("interface"))
}
```

- [ ] **Step 2: Create testdata/sample.go fixture**

Create `internal/provider/golang/testdata/sample.go`:

```go
package sample

import "fmt"

// SampleInterface is a public interface.
type SampleInterface interface {
	DoSomething()
}

// SampleStruct is a public struct.
type SampleStruct struct {
	Name string
}

// PublicFunc is an exported function with branching.
func PublicFunc(x int) string {
	if x > 0 {
		return "positive"
	}
	return "non-positive"
}

// privateFunc is an unexported function.
func privateFunc() {
	fmt.Println("hello")
}

// PublicMethod is an exported method.
func (s *SampleStruct) PublicMethod() string {
	return s.Name
}

// privateMethod is an unexported method.
func (s *SampleStruct) privateMethod() int {
	if s.Name == "" {
		return 0
	}
	return len(s.Name)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/provider/golang/ -run TestPopulateDeclarations -v`
Expected: compilation error — `golang.PopulateDeclarations` undefined

- [ ] **Step 4: Implement PopulateDeclarations**

Create `internal/provider/golang/declarations.go`:

```go
package golang

import (
	"go/token"
	"log/slog"

	"github.com/dave/dst"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// PopulateDeclarations parses a Go file and populates f.Declarations with
// per-declaration model nodes including metrics and visibility.
func PopulateDeclarations(f *model.File) {
	if f.Extension != "go" {
		return
	}

	stats, err := getOrAnalyze(f.Path)
	if err != nil {
		slog.Debug("could not analyze Go file for declarations", "path", f.Path, "error", err)
		return
	}

	_ = stats // stats already cached; we need the DST for per-decl info

	decls, err := extractDeclarations(f.Path)
	if err != nil {
		slog.Debug("could not extract declarations", "path", f.Path, "error", err)
		return
	}

	f.Declarations = decls
}

// extractDeclarations parses the Go file and builds Declaration model nodes.
func extractDeclarations(path string) ([]*model.Declaration, error) {
	dstFile, dec, fset, err := parseGoFile(path)
	if err != nil {
		return nil, err
	}

	var decls []*model.Declaration

	for _, decl := range dstFile.Decls {
		switch d := decl.(type) {
		case *dst.GenDecl:
			decls = append(decls, extractGenDecl(d)...)
		case *dst.FuncDecl:
			decls = append(decls, extractFuncDecl(d, dec, fset))
		}
	}

	return decls, nil
}

func extractGenDecl(d *dst.GenDecl) []*model.Declaration {
	var decls []*model.Declaration

	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *dst.TypeSpec:
			kind := typeSpecKind(s)
			decls = append(decls, &model.Declaration{
				Name:       s.Name.Name,
				Kind:       kind,
				Visibility: visibility(s.Name.Name),
			})
		case *dst.ValueSpec:
			valKind := "variable"
			if d.Tok == token.CONST {
				valKind = "constant"
			}
			for _, name := range s.Names {
				decls = append(decls, &model.Declaration{
					Name:       name.Name,
					Kind:       valKind,
					Visibility: visibility(name.Name),
				})
			}
		}
	}

	return decls
}

func extractFuncDecl(d *dst.FuncDecl, dec interface{ Ast interface{ Nodes map[dst.Node]interface{} } }, fset interface{ Position(token.Pos) token.Position }) *model.Declaration {
	kind := "function"
	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
	}

	md := &model.Declaration{
		Name:       d.Name.Name,
		Kind:       kind,
		Visibility: visibility(d.Name.Name),
	}

	// Set cyclomatic complexity
	cc := cyclomaticComplexity(d.Body)
	md.SetQuantity(metric.Name("cyclomatic-complexity"), cc)

	// Set function length if we can resolve positions
	// (use the decorator's AST mapping)
	type astMapper interface {
		Ast() interface{ Nodes() map[dst.Node]interface{} }
	}
	// For simplicity, compute from the DST body statement count as approximation
	// Actually, use the proper fset approach via the decorator
	// This will be refined in step — for now, set based on body statements
	if d.Body != nil {
		md.SetQuantity(metric.Name("function-length"), int64(len(d.Body.List)))
	}

	return md
}

func typeSpecKind(s *dst.TypeSpec) string {
	switch s.Type.(type) {
	case *dst.InterfaceType:
		return "interface"
	case *dst.StructType:
		return "struct"
	default:
		return "type"
	}
}

func visibility(name string) string {
	if token.IsExported(name) {
		return "public"
	}
	return "private"
}
```

**Note:** The `extractFuncDecl` signature above is a sketch. The actual implementation must use the decorator pattern already established in `file_stats.go`. See Step 5 for the refined version.

- [ ] **Step 5: Refine implementation to use existing parse infrastructure**

The Go provider already has `getOrAnalyze` and parsing code. We should not parse the file a second time. Instead, extend the existing `analyzeFile` to also return declaration nodes, or create a parallel cached function that builds declarations from the same DST.

The cleanest approach: add a `declarationCache` alongside `statsCache` that stores `[]*model.Declaration` per file path, populated on first call. `PopulateDeclarations` uses this cache.

Refined `internal/provider/golang/declarations.go`:

```go
package golang

import (
	"go/token"
	"log/slog"
	"os"
	"sync"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"golang.org/x/sync/singleflight"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

var globalDeclCache = &declCache{
	decls: make(map[string][]*model.Declaration),
	errs:  make(map[string]error),
}

type declCache struct {
	mu    sync.Mutex
	group singleflight.Group
	decls map[string][]*model.Declaration
	errs  map[string]error
}

// PopulateDeclarations parses a Go file and populates f.Declarations with
// per-declaration model nodes including metrics and visibility.
func PopulateDeclarations(f *model.File) {
	if f.Extension != "go" {
		return
	}

	decls, err := getOrExtractDeclarations(f.Path)
	if err != nil {
		slog.Debug("could not extract declarations", "path", f.Path, "error", err)
		return
	}

	f.Declarations = decls
}

func getOrExtractDeclarations(path string) ([]*model.Declaration, error) {
	globalDeclCache.mu.Lock()
	if d, ok := globalDeclCache.decls[path]; ok {
		globalDeclCache.mu.Unlock()
		return d, nil
	}
	if err, ok := globalDeclCache.errs[path]; ok {
		globalDeclCache.mu.Unlock()
		return nil, err
	}
	globalDeclCache.mu.Unlock()

	result, err, _ := globalDeclCache.group.Do(path, func() (any, error) {
		decls, err := extractDeclarations(path)
		if err != nil {
			globalDeclCache.mu.Lock()
			globalDeclCache.errs[path] = err
			globalDeclCache.mu.Unlock()
			return nil, err
		}

		globalDeclCache.mu.Lock()
		globalDeclCache.decls[path] = decls
		globalDeclCache.mu.Unlock()

		return decls, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]*model.Declaration), nil
}

func extractDeclarations(path string) ([]*model.Declaration, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	dstFile, err := dec.ParseFile(path, src, 0)
	if err != nil {
		return nil, err
	}

	var decls []*model.Declaration

	for _, decl := range dstFile.Decls {
		switch d := decl.(type) {
		case *dst.GenDecl:
			decls = append(decls, extractGenDecls(d)...)
		case *dst.FuncDecl:
			md := extractFuncDecl(d, dec, fset)
			decls = append(decls, md)
		}
	}

	return decls, nil
}

func extractGenDecls(d *dst.GenDecl) []*model.Declaration {
	var decls []*model.Declaration

	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *dst.TypeSpec:
			decls = append(decls, &model.Declaration{
				Name:       s.Name.Name,
				Kind:       typeSpecKind(s),
				Visibility: visibility(s.Name.Name),
			})
		case *dst.ValueSpec:
			valKind := "variable"
			if d.Tok == token.CONST {
				valKind = "constant"
			}
			for _, name := range s.Names {
				decls = append(decls, &model.Declaration{
					Name:       name.Name,
					Kind:       valKind,
					Visibility: visibility(name.Name),
				})
			}
		}
	}

	return decls
}

func extractFuncDecl(d *dst.FuncDecl, dec *decorator.Decorator, fset *token.FileSet) *model.Declaration {
	kind := "function"
	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
	}

	md := &model.Declaration{
		Name:       d.Name.Name,
		Kind:       kind,
		Visibility: visibility(d.Name.Name),
	}

	// Cyclomatic complexity
	cc := cyclomaticComplexity(d.Body)
	md.SetQuantity(metric.Name("cyclomatic-complexity"), cc)

	// Function length from AST positions
	astNode := dec.Ast.Nodes[d]
	if astNode != nil {
		startLine := fset.Position(astNode.Pos()).Line
		endLine := fset.Position(astNode.End()).Line
		md.SetQuantity(metric.Name("function-length"), int64(endLine-startLine+1))
	}

	return md
}

func typeSpecKind(s *dst.TypeSpec) string {
	switch s.Type.(type) {
	case *dst.InterfaceType:
		return "interface"
	case *dst.StructType:
		return "struct"
	default:
		return "type"
	}
}

func visibility(name string) string {
	if token.IsExported(name) {
		return "public"
	}
	return "private"
}

// ResetDeclCacheForTesting clears the declaration cache. Test use only.
func ResetDeclCacheForTesting() {
	globalDeclCache = &declCache{
		decls: make(map[string][]*model.Declaration),
		errs:  make(map[string]error),
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/provider/golang/ -run TestPopulateDeclarations -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/provider/golang/declarations.go internal/provider/golang/declarations_test.go internal/provider/golang/testdata/sample.go
git commit -m "feat: add PopulateDeclarations to Go provider for per-declaration metrics"
```

---

## Task 6: Register FilterFunc on Go Declaration Metrics

**Files:**
- Modify: `internal/provider/golang/base_metrics.go`

- [ ] **Step 1: Write test to verify FilterFunc is registered**

Add to `internal/provider/golang/base_metrics_test.go`:

```go
func TestRegisterBase_DeclarationMetrics_HaveFilterFunc(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	golang.RegisterBase()

	desc, ok := provider.GetBase("methods")
	g.Expect(ok).To(BeTrue())
	g.Expect(desc.FilterFunc).NotTo(BeNil())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/provider/golang/ -run TestRegisterBase_DeclarationMetrics_HaveFilterFunc -v`
Expected: FAIL — `desc.FilterFunc` is nil

- [ ] **Step 3: Add FilterFunc to declaration-level Go descriptors**

In `internal/provider/golang/base_metrics.go`, add a shared filter function and set it on all declaration-level descriptors:

```go
// goDeclarationFilter evaluates visibility filters against a Declaration node.
var goDeclarationFilter = func(filter metric.FilterName, node any) bool {
	d, ok := node.(*model.Declaration)
	if !ok {
		return false
	}

	return d.MatchesFilter(filter)
}
```

Then update each declaration-level descriptor in `goBaseMetrics` to include `FilterFunc: goDeclarationFilter`. For example:

```go
{
    Name:           Methods,
    Kind:           metric.Quantity,
    Level:          metric.LevelDeclaration,
    Description:    "Count of method declarations (with receiver).",
    Filters:        goVisibilityNames,
    Aggregations:   goDeclCountAggs,
    DefaultPalette: palette.Neutral,
    FilterFunc:     goDeclarationFilter,
},
```

Apply to: Types, Interfaces, Structs, Functions, Methods, Constants, Variables, CyclomaticComplexity, FunctionLength.

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/base_metrics.go internal/provider/golang/base_metrics_test.go
git commit -m "feat: register FilterFunc on Go declaration-level base metrics"
```

---

## Task 7: Wire Declaration Population into Go Provider Pipeline

**Files:**
- Modify: `internal/provider/golang/go_provider.go`
- Modify: `internal/stages/providers.go`

- [ ] **Step 1: Write integration test**

Add to `internal/stages/aggregation_test.go`:

```go
func TestComputeAggregations_DeclarationLevel_CountsDeclarations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Build a tree with declarations directly
	decl1 := &model.Declaration{Name: "Foo", Kind: "method", Visibility: "public"}
	decl1.SetQuantity("cyclomatic-complexity", 3)

	decl2 := &model.Declaration{Name: "bar", Kind: "method", Visibility: "private"}
	decl2.SetQuantity("cyclomatic-complexity", 7)

	decl3 := &model.Declaration{Name: "Baz", Kind: "method", Visibility: "public"}
	decl3.SetQuantity("cyclomatic-complexity", 2)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{decl1, decl2}},
			{Name: "b.go", Declarations: []*model.Declaration{decl3}},
		},
	}

	// Expression: public.methods.count — count public methods
	filterFunc := func(filter metric.FilterName, node any) bool {
		d, ok := node.(*model.Declaration)
		if !ok {
			return false
		}
		return d.MatchesFilter(filter)
	}

	expr := metric.MetricExpression{
		Filter:      "public",
		Base:        "methods",
		Aggregation: metric.AggCount,
	}

	resolved := provider.ResolvedMetric{
		Expression:       expr,
		Descriptor:       provider.BaseMetricDescriptor{
			Name:       "methods",
			Kind:       metric.Quantity,
			Level:      metric.LevelDeclaration,
			Filters:    []metric.FilterName{"public", "private"},
			FilterFunc: filterFunc,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "public.methods.count",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("public.methods.count")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(2))) // decl1 + decl3 are public methods
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stages/ -run TestComputeAggregations_DeclarationLevel -v`
Expected: FAIL — current code returns error "aggregation of declaration-level metric not yet supported"

- [ ] **Step 3: Update ComputeAggregations to handle LevelDeclaration**

In `internal/stages/aggregation.go`, replace the `LevelDeclaration` error with actual logic:

```go
func ComputeAggregations(root *model.Directory, expressions []provider.ResolvedMetric) error {
	if len(expressions) == 0 {
		return nil
	}

	for _, resolved := range expressions {
		switch resolved.SourceLevel {
		case metric.LevelFile:
			if err := aggregateDirectory(root, resolved); err != nil {
				return err
			}
		case metric.LevelDeclaration:
			if err := aggregateDeclarations(root, resolved); err != nil {
				return err
			}
		case metric.LevelCommit:
			if err := aggregateCommits(root, resolved); err != nil {
				return err
			}
		default:
			return eris.Errorf(
				"aggregation of %s-level metric %q is not supported",
				resolved.SourceLevel, resolved.Expression.Base,
			)
		}
	}

	return nil
}
```

- [ ] **Step 4: Implement aggregateDeclarations**

Add to `internal/stages/aggregation.go`:

```go
func aggregateDeclarations(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateDeclarations(child, resolved); err != nil {
			return err
		}
	}

	switch resolved.Descriptor.Kind {
	case metric.Classification:
		return aggregateDeclarationClassification(dir, resolved)
	case metric.Quantity, metric.Measure:
		return aggregateDeclarationNumeric(dir, resolved)
	default:
		return eris.Errorf(
			"aggregation for declaration metric %q uses unsupported source kind %d",
			resolved.Expression.Base, resolved.Descriptor.Kind,
		)
	}
}

func aggregateDeclarationNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectDeclarationNumericValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for declaration metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
}

func aggregateDeclarationClassification(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectDeclarationClassificationValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	switch resolved.Expression.Aggregation {
	case metric.AggMode:
		dir.SetClassification(resolved.ResultName, metric.AggregateMode(values))
	case metric.AggDistinct:
		dir.SetQuantity(resolved.ResultName, int64(metric.AggregateDistinct(values)))
	default:
		return eris.Errorf(
			"classification aggregation %q for declaration metric %q is unsupported",
			resolved.Expression.Aggregation, resolved.Expression.Base,
		)
	}

	return nil
}

func collectDeclarationNumericValues(dir *model.Directory, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		// Apply filter if present
		if !resolved.Expression.Filter.IsZero() {
			if !resolved.Descriptor.PassesFilter(resolved.Expression.Filter, d) {
				return
			}
		}

		// Apply kind filter: only count declarations matching the base metric's kind category
		if !matchesDeclKind(d, resolved.Expression.Base) {
			return
		}

		switch resolved.Descriptor.Kind {
		case metric.Quantity:
			if v, ok := d.Quantity(resolved.Expression.Base); ok {
				values = append(values, float64(v))
			} else {
				// For count aggregation, the declaration itself is the unit
				if resolved.Expression.Aggregation == metric.AggCount {
					values = append(values, 1)
				}
			}
		case metric.Measure:
			if v, ok := d.Measure(resolved.Expression.Base); ok {
				values = append(values, v)
			}
		}
	})

	return values
}

func collectDeclarationClassificationValues(dir *model.Directory, resolved provider.ResolvedMetric) []string {
	var values []string

	model.WalkDeclarations(dir, func(d *model.Declaration, _ *model.File) {
		if !resolved.Expression.Filter.IsZero() {
			if !resolved.Descriptor.PassesFilter(resolved.Expression.Filter, d) {
				return
			}
		}

		if !matchesDeclKind(d, resolved.Expression.Base) {
			return
		}

		if v, ok := d.Classification(resolved.Expression.Base); ok {
			values = append(values, v)
		}
	})

	return values
}

// matchesDeclKind checks whether a declaration matches the semantic category
// implied by the base metric name. For metrics like "methods", only method
// declarations match; for "cyclomatic-complexity", all functions/methods match.
func matchesDeclKind(d *model.Declaration, baseName metric.Name) bool {
	switch baseName {
	case "types":
		return d.Kind == "type" || d.Kind == "struct" || d.Kind == "interface"
	case "interfaces":
		return d.Kind == "interface"
	case "structs":
		return d.Kind == "struct"
	case "functions":
		return d.Kind == "function"
	case "methods":
		return d.Kind == "method"
	case "constants":
		return d.Kind == "constant"
	case "variables":
		return d.Kind == "variable"
	case "cyclomatic-complexity", "function-length":
		return d.Kind == "function" || d.Kind == "method"
	default:
		// Unknown metric — include all declarations
		return true
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/stages/ -run TestComputeAggregations_DeclarationLevel -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/stages/aggregation.go internal/stages/aggregation_test.go
git commit -m "feat: implement declaration-level aggregation in ComputeAggregations"
```

---

## Task 8: Wire PopulateDeclarations into Pipeline

**Files:**
- Modify: `internal/stages/providers.go` (or new stage file)
- Modify: `internal/stages/compute_aggregations_stage.go`

The declarations need to be populated on files BEFORE `ComputeAggregations` runs. The Go provider's `Load()` already walks all files — but it only sets file-level metrics. We need a stage that populates declarations.

- [ ] **Step 1: Write integration test**

Add to `internal/stages/aggregation_test.go`:

```go
func TestComputeAggregations_DeclarationLevel_MeanCyclomaticComplexity(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// 3 functions with complexity 2, 4, 6 → mean = 4.0
	d1 := &model.Declaration{Name: "A", Kind: "function", Visibility: "public"}
	d1.SetQuantity("cyclomatic-complexity", 2)
	d2 := &model.Declaration{Name: "B", Kind: "function", Visibility: "public"}
	d2.SetQuantity("cyclomatic-complexity", 4)
	d3 := &model.Declaration{Name: "C", Kind: "method", Visibility: "private"}
	d3.SetQuantity("cyclomatic-complexity", 6)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Declarations: []*model.Declaration{d1, d2}},
			{Name: "b.go", Declarations: []*model.Declaration{d3}},
		},
	}

	expr := metric.MetricExpression{
		Base:        "cyclomatic-complexity",
		Aggregation: metric.AggMean,
	}

	resolved := provider.ResolvedMetric{
		Expression: expr,
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "cyclomatic-complexity",
			Kind:  metric.Quantity,
			Level: metric.LevelDeclaration,
		},
		SourceLevel:      metric.LevelDeclaration,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Measure,
		ResultName:       "cyclomatic-complexity.mean",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Measure("cyclomatic-complexity.mean")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 4.0, 0.01))
}
```

- [ ] **Step 2: Create PopulateDeclarations stage**

Create or add to `internal/stages/compute_aggregations_stage.go`:

```go
// PopulateDeclarations walks all Go files and populates per-declaration model nodes.
// Must run after RunProviders (so files are scanned) and before ComputeAggregations.
func PopulateDeclarations(c *CommonState) error {
	if !c.Requested.HasDeclarationExpressions() {
		return nil
	}

	model.WalkFiles(c.Root, func(f *model.File) {
		golang.PopulateDeclarations(f)
	})

	return nil
}
```

Add `HasDeclarationExpressions()` to `RequestedMetrics`:

```go
// HasDeclarationExpressions reports whether any expression needs declaration-level data.
func (r RequestedMetrics) HasDeclarationExpressions() bool {
	for _, expr := range r.Expressions {
		if expr.SourceLevel == metric.LevelDeclaration {
			return true
		}
	}
	return false
}

// HasCommitExpressions reports whether any expression needs commit-level data.
func (r RequestedMetrics) HasCommitExpressions() bool {
	for _, expr := range r.Expressions {
		if expr.SourceLevel == metric.LevelCommit {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3: Wire into all 5 viz commands**

In each command file (`treemap_cmd.go`, `bubbletree_cmd.go`, `radialtree_cmd.go`, `scatter_cmd.go`, `spiral_cmd.go`), add `stages.PopulateDeclarations` after `stages.RunProviders` and before `stages.RunAggregations`:

```go
stages.RunProviders,
stages.PopulateDeclarations,
stages.RunAggregations,
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stages/compute_aggregations_stage.go internal/stages/requested.go \
    cmd/codeviz/treemap_cmd.go cmd/codeviz/bubbletree_cmd.go \
    cmd/codeviz/radialtree_cmd.go cmd/codeviz/scatter_cmd.go cmd/codeviz/spiral_cmd.go
git commit -m "feat: wire PopulateDeclarations stage into all viz pipelines"
```

---

## Task 9: Fix LegacyNames for Declaration-Level Base Metrics

**Files:**
- Modify: `internal/stages/requested.go`

The current `LegacyNames()` includes base metrics from expressions in the list passed to `provider.Run`. But `provider.Run` only knows about file-target metrics. Declaration-level base metrics like `"methods"` don't exist in the legacy registry, causing the "unknown file metric" error.

- [ ] **Step 1: Write failing test**

Add to `internal/stages/requested_test.go`:

```go
func TestClassifyRequestedMetrics_DeclarationLevelBase_NotInLegacyNames(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "public.methods.count" should classify as an expression but NOT
	// put "methods" into LegacyNames (since methods is declaration-level,
	// not a file-level provider)
	names := []metric.Name{"public.methods.count"}
	result := stages.ClassifyRequestedMetrics(names, metric.LevelDirectory)

	g.Expect(result.Expressions).To(HaveLen(1))
	g.Expect(result.LegacyNames()).NotTo(ContainElement(metric.Name("methods")))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/stages/ -run TestClassifyRequestedMetrics_DeclarationLevelBase -v`
Expected: FAIL — `LegacyNames()` currently includes `"methods"`

- [ ] **Step 3: Fix LegacyNames to exclude non-file-level base metrics**

In `internal/stages/requested.go`, change `ClassifyRequestedMetrics` to only add base metrics to `BaseMetrics` when their source level is `LevelFile`:

```go
func ClassifyRequestedMetrics(names []metric.Name, targetLevel metric.MetricLevel) RequestedMetrics {
	var result RequestedMetrics

	baseSeen := make(map[metric.Name]bool)

	for _, name := range names {
		expr, parseErr := metric.ParseExpression(string(name))
		if parseErr != nil {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		resolved, resolveErr := provider.ResolveExpression(expr, targetLevel)
		if resolveErr != nil {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		if !resolved.NeedsAggregation {
			result.Legacy = append(result.Legacy, name)

			continue
		}

		result.Expressions = append(result.Expressions, resolved)

		// Only add to BaseMetrics if the source is file-level (needs provider.Run)
		// Declaration and commit level metrics are populated by separate stages.
		if resolved.SourceLevel == metric.LevelFile && !baseSeen[expr.Base] {
			baseSeen[expr.Base] = true
			result.BaseMetrics = append(result.BaseMetrics, expr.Base)
		}
	}

	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS (existing tests for file-level expressions still work, new test passes)

- [ ] **Step 5: Commit**

```bash
git add internal/stages/requested.go internal/stages/requested_test.go
git commit -m "fix: exclude declaration/commit-level base metrics from LegacyNames"
```

---

## Task 10: Commit Model — Git Provider Populates Commits (Phase 3)

**Files:**
- Create: `internal/provider/git/commits.go`
- Create: `internal/provider/git/commits_test.go`
- Modify: `internal/provider/git/base_metrics.go`

Phase 3 adds commit-level base metrics. The git provider currently registers all metrics at `LevelFile`. We add new commit-level base metrics (`lines-added`, `lines-removed`, `lines-changed`) and a `PopulateCommits` function.

- [ ] **Step 1: Register commit-level base metrics**

Add to `internal/provider/git/base_metrics.go`:

```go
const (
	// Commit-level metrics (new)
	LinesAdded   metric.Name = "lines-added"
	LinesRemoved metric.Name = "lines-removed"
	LinesChanged metric.Name = "lines-changed"
)
```

Register them in `RegisterBase()`:

```go
provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
    Name:         LinesAdded,
    Kind:         metric.Quantity,
    Level:        metric.LevelCommit,
    Description:  "Lines added in a single commit.",
    Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
    DefaultPalette: palette.Temperature,
}, GitProvider)

provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
    Name:         LinesRemoved,
    Kind:         metric.Quantity,
    Level:        metric.LevelCommit,
    Description:  "Lines removed in a single commit.",
    Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
    DefaultPalette: palette.Temperature,
}, GitProvider)

provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
    Name:         LinesChanged,
    Kind:         metric.Quantity,
    Level:        metric.LevelCommit,
    Description:  "Lines changed (added + removed) in a single commit.",
    Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
    DefaultPalette: palette.Temperature,
}, GitProvider)
```

- [ ] **Step 2: Write test for PopulateCommits**

```go
// internal/provider/git/commits_test.go
package git_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

func TestPopulateCommits_SetsCommitsOnFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// This test requires a real git repo — use the code-visualizer repo itself
	f := &model.File{
		Path:      "go.mod",
		Name:      "go.mod",
		Extension: "mod",
	}

	// PopulateCommits needs the repo root
	err := git.PopulateCommits(f, ".")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(f.Commits).NotTo(BeEmpty())

	// Each commit should have lines-added set
	for _, c := range f.Commits {
		g.Expect(c.Hash).NotTo(BeEmpty())
		_, ok := c.Quantity("lines-added")
		g.Expect(ok).To(BeTrue(), "expected lines-added on commit %s", c.Hash)
	}
}
```

- [ ] **Step 3: Implement PopulateCommits**

Create `internal/provider/git/commits.go`:

```go
package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// PopulateCommits walks the git log for a file and populates f.Commits
// with per-commit model nodes including lines-added/removed/changed.
func PopulateCommits(f *model.File, repoRoot string) error {
	svc, err := getOrOpenRepo(repoRoot)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(repoRoot, f.Path)
	if err != nil {
		relPath = f.Path
	}

	commits, err := svc.commitsForFile(relPath)
	if err != nil {
		return err
	}

	f.Commits = commits
	return nil
}
```

The `commitsForFile` method on `repoService` walks the git log for a file and returns `[]*model.Commit` with `lines-added`, `lines-removed`, `lines-changed` set on each.

**Note:** The exact implementation of `commitsForFile` depends on the existing git provider internals. It should reuse the existing `repoService` and git log walking infrastructure. The implementer should check `internal/provider/git/repo_service.go` for available methods and add `commitsForFile` using the same patterns as `totalLinesAdded`/`totalLinesRemoved` but returning per-commit data rather than a sum.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/provider/git/ -run TestPopulateCommits -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/commits.go internal/provider/git/commits_test.go internal/provider/git/base_metrics.go
git commit -m "feat: add commit-level base metrics and PopulateCommits for git provider"
```

---

## Task 11: Implement Commit-Level Aggregation

**Files:**
- Modify: `internal/stages/aggregation.go`
- Modify: `internal/stages/aggregation_test.go`

- [ ] **Step 1: Write test**

```go
func TestComputeAggregations_CommitLevel_MaxLinesChanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c1 := &model.Commit{Hash: "aaa"}
	c1.SetQuantity("lines-changed", 10)
	c2 := &model.Commit{Hash: "bbb"}
	c2.SetQuantity("lines-changed", 50)
	c3 := &model.Commit{Hash: "ccc"}
	c3.SetQuantity("lines-changed", 25)

	root := &model.Directory{
		Name: "root",
		Files: []*model.File{
			{Name: "a.go", Commits: []*model.Commit{c1, c2}},
			{Name: "b.go", Commits: []*model.Commit{c3}},
		},
	}

	expr := metric.MetricExpression{
		Base:        "lines-changed",
		Aggregation: metric.AggMax,
	}

	resolved := provider.ResolvedMetric{
		Expression: expr,
		Descriptor: provider.BaseMetricDescriptor{
			Name:  "lines-changed",
			Kind:  metric.Quantity,
			Level: metric.LevelCommit,
		},
		SourceLevel:      metric.LevelCommit,
		TargetLevel:      metric.LevelDirectory,
		ResultKind:       metric.Quantity,
		ResultName:       "lines-changed.max",
		NeedsAggregation: true,
	}

	err := stages.ComputeAggregations(root, []provider.ResolvedMetric{resolved})
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := root.Quantity("lines-changed.max")
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(50)))
}
```

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL (or error if `aggregateCommits` is not yet stubbed)

- [ ] **Step 3: Implement aggregateCommits**

Add to `internal/stages/aggregation.go`:

```go
func aggregateCommits(dir *model.Directory, resolved provider.ResolvedMetric) error {
	for _, child := range dir.Dirs {
		if err := aggregateCommits(child, resolved); err != nil {
			return err
		}
	}

	switch resolved.Descriptor.Kind {
	case metric.Quantity, metric.Measure:
		return aggregateCommitNumeric(dir, resolved)
	default:
		return eris.Errorf(
			"aggregation for commit metric %q uses unsupported source kind %d",
			resolved.Expression.Base, resolved.Descriptor.Kind,
		)
	}
}

func aggregateCommitNumeric(dir *model.Directory, resolved provider.ResolvedMetric) error {
	values := collectCommitNumericValues(dir, resolved)
	if len(values) == 0 {
		return nil
	}

	result, err := applyNumericAggregation(resolved.Expression.Aggregation, values)
	if err != nil {
		return err
	}

	switch resolved.ResultKind {
	case metric.Quantity:
		dir.SetQuantity(resolved.ResultName, int64(result))
	case metric.Measure:
		dir.SetMeasure(resolved.ResultName, result)
	default:
		return eris.Errorf(
			"aggregation %q for commit metric %q uses unsupported result kind %d",
			resolved.Expression.Aggregation, resolved.Expression.Base, resolved.ResultKind,
		)
	}

	return nil
}

func collectCommitNumericValues(dir *model.Directory, resolved provider.ResolvedMetric) []float64 {
	var values []float64

	model.WalkCommits(dir, func(c *model.Commit, _ *model.File) {
		switch resolved.Descriptor.Kind {
		case metric.Quantity:
			if v, ok := c.Quantity(resolved.Expression.Base); ok {
				values = append(values, float64(v))
			}
		case metric.Measure:
			if v, ok := c.Measure(resolved.Expression.Base); ok {
				values = append(values, v)
			}
		}
	})

	return values
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `task test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/stages/aggregation.go internal/stages/aggregation_test.go
git commit -m "feat: implement commit-level aggregation in ComputeAggregations"
```

---

## Task 12: Wire PopulateCommits into Pipeline

**Files:**
- Modify: `internal/stages/compute_aggregations_stage.go`
- Modify: all 5 viz command files (if not already wired in Task 8)

- [ ] **Step 1: Add PopulateCommits stage function**

```go
// PopulateCommits walks all files and populates per-commit model nodes.
// Must run after RunProviders and before ComputeAggregations.
func PopulateCommits(c *CommonState) error {
	if !c.Requested.HasCommitExpressions() {
		return nil
	}

	repoRoot := c.Root.Path
	model.WalkFiles(c.Root, func(f *model.File) {
		_ = git.PopulateCommits(f, repoRoot)
	})

	return nil
}
```

- [ ] **Step 2: Wire into pipeline (after PopulateDeclarations, before RunAggregations)**

Pipeline order in each viz command:
```go
stages.RunProviders,
stages.PopulateDeclarations,
stages.PopulateCommits,
stages.RunAggregations,
```

- [ ] **Step 3: Run task ci**

Run: `task ci`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/stages/compute_aggregations_stage.go \
    cmd/codeviz/treemap_cmd.go cmd/codeviz/bubbletree_cmd.go \
    cmd/codeviz/radialtree_cmd.go cmd/codeviz/scatter_cmd.go cmd/codeviz/spiral_cmd.go
git commit -m "feat: wire PopulateCommits stage into all viz pipelines"
```

---

## Task 13: End-to-End Verification

- [ ] **Step 1: Run the full test suite**

Run: `task ci`
Expected: All tests pass, lint clean

- [ ] **Step 2: Manual smoke test with the original failing command**

Run: `go run ./cmd/codeviz bubbletree --border public.methods.count .`
Expected: No crash, produces output (the metric may be zero for non-Go directories, which is fine — the point is no "unknown file metric" error)

- [ ] **Step 3: Test other expression combinations**

```bash
go run ./cmd/codeviz bubbletree --border cyclomatic-complexity.mean .
go run ./cmd/codeviz bubbletree --border methods.count .
go run ./cmd/codeviz treemap --fill file-size.sum .
```

Expected: All produce output without errors

- [ ] **Step 4: Push and update PR**

```bash
git push origin feature/metric-expressions-design
```
