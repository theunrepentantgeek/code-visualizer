# Go-Specific Code Metrics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 35 Go-specific metrics that parse `.go` files with `github.com/dave/dst` and extract declaration counts, type taxonomy, import classification, cyclomatic complexity, function length, and comment ratio.

**Architecture:** A new `internal/provider/golang/` package follows the data-driven pattern from `internal/provider/git/` — one `goProvider` struct backed by a `providerDefs` map. A shared `statsCache` (guarded by `singleflight`) parses each `.go` file exactly once via `dst`; every metric provider reads from the cached `fileStats`. Module path lookup (for `internal-import-count`) walks up to `go.mod` and caches per-directory.

**Tech Stack:** Go 1.26.1, `github.com/dave/dst` (AST parsing), `golang.org/x/sync/singleflight` (already available via `golang.org/x/sync`), Gomega (test assertions)

**Spec:** `docs/superpowers/specs/2026-05-23-go-metrics-design.md`

**Important:** All AST parsing uses `dst` exclusively. Do NOT import `go/parser` directly. The decorator internally calls `go/parser` once — accessing `decorator.Ast.Nodes` or `decorator.Fset` for position info is reading from that single parse, not a second parse. The only `go/ast` usage is type assertions on the decorator's node map.

---

## File Structure

```
internal/provider/golang/
  metrics.go          — 35 metric.Name constants + IsGoMetric()
  metrics_test.go     — IsGoMetric tests
  file_stats.go       — fileStats struct, analyzeFile(), countDeclarations, isPublic, computeAggregates
  file_stats_test.go  — declaration counting + analyzeFile tests
  cyclomatic.go       — cyclomaticComplexity() visitor
  cyclomatic_test.go  — table-driven complexity tests
  comments.go         — computeCommentRatio()
  comments_test.go    — comment ratio tests
  imports.go          — classifyImports(), findModulePath(), moduleCache
  imports_test.go     — import classification + module path tests
  go_provider.go      — goProvider struct, statsCache, walkGoFiles, getOrAnalyze, ResetCacheForTesting
  provider_defs.go    — providerDefs map (all 35 entries), quantityField/measureField helpers
  register.go         — Register()
  go_provider_test.go — integration tests

cmd/codeviz/main.go   — add golang.Register() call
```

---

### Task 1: Add dst dependency

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add the dst module**

```bash
cd /home/bevan/github/code-visualizer
go get github.com/dave/dst@latest
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 3: Create the package directory**

```bash
mkdir -p internal/provider/golang
```

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add github.com/dave/dst dependency for Go AST parsing (#289)"
```

---

### Task 2: Metric name constants

**Files:**
- Create: `internal/provider/golang/metrics.go`
- Create: `internal/provider/golang/metrics_test.go`

- [ ] **Step 1: Write the test**

```go
package golang

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestIsGoMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(IsGoMetric(TypeCount)).To(BeTrue())
	g.Expect(IsGoMetric(PublicTypeCount)).To(BeTrue())
	g.Expect(IsGoMetric(CommentRatio)).To(BeTrue())
	g.Expect(IsGoMetric(CyclomaticComplexityMean)).To(BeTrue())
	g.Expect(IsGoMetric(FunctionLengthMax)).To(BeTrue())
	g.Expect(IsGoMetric(InternalImportCount)).To(BeTrue())
	g.Expect(IsGoMetric("file-size")).To(BeFalse())
	g.Expect(IsGoMetric("file-lines")).To(BeFalse())
	g.Expect(IsGoMetric("unknown-metric")).To(BeFalse())
}

func TestAllGoMetricCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(allMetrics).To(HaveLen(35))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run TestIsGoMetric -v -count=1
```

Expected: FAIL — `IsGoMetric` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/provider/golang/metrics.go`:

```go
// Package golang provides metric providers for Go-specific code metrics.
package golang

import "github.com/theunrepentantgeek/code-visualizer/internal/metric"

const (
	TypeCount               metric.Name = "type-count"
	PublicTypeCount         metric.Name = "public-type-count"
	PrivateTypeCount        metric.Name = "private-type-count"
	InterfaceCount          metric.Name = "interface-count"
	PublicInterfaceCount    metric.Name = "public-interface-count"
	PrivateInterfaceCount   metric.Name = "private-interface-count"
	StructCount             metric.Name = "struct-count"
	PublicStructCount       metric.Name = "public-struct-count"
	PrivateStructCount      metric.Name = "private-struct-count"
	FunctionCount           metric.Name = "function-count"
	PublicFunctionCount     metric.Name = "public-function-count"
	PrivateFunctionCount    metric.Name = "private-function-count"
	MethodCount             metric.Name = "method-count"
	PublicMethodCount       metric.Name = "public-method-count"
	PrivateMethodCount      metric.Name = "private-method-count"
	ConstantCount           metric.Name = "constant-count"
	PublicConstantCount     metric.Name = "public-constant-count"
	PrivateConstantCount    metric.Name = "private-constant-count"
	VariableCount           metric.Name = "variable-count"
	PublicVariableCount     metric.Name = "public-variable-count"
	PrivateVariableCount    metric.Name = "private-variable-count"
	ImportCount             metric.Name = "import-count"
	StdlibImportCount       metric.Name = "stdlib-import-count"
	ExternalImportCount     metric.Name = "external-import-count"
	InternalImportCount     metric.Name = "internal-import-count"
	DeclarationCount        metric.Name = "declaration-count"
	PublicDeclarationCount  metric.Name = "public-declaration-count"
	PrivateDeclarationCount metric.Name = "private-declaration-count"
	CyclomaticComplexitySum  metric.Name = "cyclomatic-complexity-sum"
	CyclomaticComplexityMax  metric.Name = "cyclomatic-complexity-max"
	CyclomaticComplexityMean metric.Name = "cyclomatic-complexity-mean"
	FunctionLengthSum       metric.Name = "function-length-sum"
	FunctionLengthMax       metric.Name = "function-length-max"
	FunctionLengthMean      metric.Name = "function-length-mean"
	CommentRatio            metric.Name = "comment-ratio"
)

// IsGoMetric reports whether name is a Go-specific metric.
func IsGoMetric(name metric.Name) bool {
	_, ok := allMetrics[name]
	return ok
}

var allMetrics = map[metric.Name]struct{}{
	TypeCount:                {},
	PublicTypeCount:          {},
	PrivateTypeCount:         {},
	InterfaceCount:           {},
	PublicInterfaceCount:     {},
	PrivateInterfaceCount:    {},
	StructCount:              {},
	PublicStructCount:        {},
	PrivateStructCount:       {},
	FunctionCount:            {},
	PublicFunctionCount:      {},
	PrivateFunctionCount:     {},
	MethodCount:              {},
	PublicMethodCount:        {},
	PrivateMethodCount:       {},
	ConstantCount:            {},
	PublicConstantCount:      {},
	PrivateConstantCount:     {},
	VariableCount:            {},
	PublicVariableCount:      {},
	PrivateVariableCount:     {},
	ImportCount:              {},
	StdlibImportCount:        {},
	ExternalImportCount:      {},
	InternalImportCount:      {},
	DeclarationCount:         {},
	PublicDeclarationCount:   {},
	PrivateDeclarationCount:  {},
	CyclomaticComplexitySum:  {},
	CyclomaticComplexityMax:  {},
	CyclomaticComplexityMean: {},
	FunctionLengthSum:        {},
	FunctionLengthMax:        {},
	FunctionLengthMean:       {},
	CommentRatio:             {},
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/provider/golang/ -run TestIsGoMetric -v -count=1
go test ./internal/provider/golang/ -run TestAllGoMetricCount -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/metrics.go internal/provider/golang/metrics_test.go
git commit -m "feat(golang): add 35 Go metric name constants and IsGoMetric (#289)"
```

---

### Task 3: Declaration counting

**Files:**
- Create: `internal/provider/golang/file_stats.go`
- Create: `internal/provider/golang/file_stats_test.go`

- [ ] **Step 1: Write the test**

```go
package golang

import (
	"go/token"
	"testing"

	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)

func TestCountDeclarations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

type Exported struct{}
type unexported interface{}

func PublicFunc() {}
func privateFunc() {}

func (e *Exported) PublicMethod() {}
func (e *Exported) privateMethod() {}

const (
	PublicConst  = 1
	privateConst = 2
)

var (
	PublicVar  int
	privateVar string
)
`

	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	countDeclarations(dstFile, stats)

	g.Expect(stats.types).To(Equal(int64(2)))
	g.Expect(stats.publicTypes).To(Equal(int64(1)))
	g.Expect(stats.privateTypes).To(Equal(int64(1)))
	g.Expect(stats.structs).To(Equal(int64(1)))
	g.Expect(stats.publicStructs).To(Equal(int64(1)))
	g.Expect(stats.privateStructs).To(Equal(int64(0)))
	g.Expect(stats.interfaces).To(Equal(int64(1)))
	g.Expect(stats.publicInterfaces).To(Equal(int64(0)))
	g.Expect(stats.privateInterfaces).To(Equal(int64(1)))
	g.Expect(stats.functions).To(Equal(int64(2)))
	g.Expect(stats.publicFunctions).To(Equal(int64(1)))
	g.Expect(stats.privateFunctions).To(Equal(int64(1)))
	g.Expect(stats.methods).To(Equal(int64(2)))
	g.Expect(stats.publicMethods).To(Equal(int64(1)))
	g.Expect(stats.privateMethods).To(Equal(int64(1)))
	g.Expect(stats.constants).To(Equal(int64(2)))
	g.Expect(stats.publicConstants).To(Equal(int64(1)))
	g.Expect(stats.privateConstants).To(Equal(int64(1)))
	g.Expect(stats.variables).To(Equal(int64(2)))
	g.Expect(stats.publicVariables).To(Equal(int64(1)))
	g.Expect(stats.privateVariables).To(Equal(int64(1)))
}

func TestCountDeclarationsMultipleNames(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

var X, Y, z int
const A, b = 1, 2
`

	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	countDeclarations(dstFile, stats)

	g.Expect(stats.variables).To(Equal(int64(3)))
	g.Expect(stats.publicVariables).To(Equal(int64(2)))
	g.Expect(stats.privateVariables).To(Equal(int64(1)))
	g.Expect(stats.constants).To(Equal(int64(2)))
	g.Expect(stats.publicConstants).To(Equal(int64(1)))
	g.Expect(stats.privateConstants).To(Equal(int64(1)))
}

func TestComputeAggregates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	stats := &fileStats{
		types:           2,
		publicTypes:     1,
		privateTypes:    1,
		functions:       3,
		publicFunctions: 2,
		privateFunctions: 1,
		methods:         1,
		publicMethods:   1,
		privateMethods:  0,
		constants:       4,
		publicConstants: 2,
		privateConstants: 2,
		variables:       2,
		publicVariables: 1,
		privateVariables: 1,
	}

	stats.computeAggregates()

	g.Expect(stats.declarations).To(Equal(int64(12)))
	g.Expect(stats.publicDeclarations).To(Equal(int64(7)))
	g.Expect(stats.privateDeclarations).To(Equal(int64(5)))
}

func TestIsPublic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(isPublic("Exported")).To(BeTrue())
	g.Expect(isPublic("unexported")).To(BeFalse())
	g.Expect(isPublic("X")).To(BeTrue())
	g.Expect(isPublic("x")).To(BeFalse())
	g.Expect(isPublic("_private")).To(BeFalse())
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run TestCountDeclarations -v -count=1
```

Expected: FAIL — `fileStats` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/provider/golang/file_stats.go`:

```go
package golang

import (
	"go/token"

	"github.com/dave/dst"
)

type fileStats struct {
	types               int64
	publicTypes         int64
	privateTypes        int64
	interfaces          int64
	publicInterfaces    int64
	privateInterfaces   int64
	structs             int64
	publicStructs       int64
	privateStructs      int64
	functions           int64
	publicFunctions     int64
	privateFunctions    int64
	methods             int64
	publicMethods       int64
	privateMethods      int64
	constants           int64
	publicConstants     int64
	privateConstants    int64
	variables           int64
	publicVariables     int64
	privateVariables    int64
	imports             int64
	stdlibImports       int64
	externalImports     int64
	internalImports     int64
	declarations        int64
	publicDeclarations  int64
	privateDeclarations int64
	cyclomaticSum       int64
	cyclomaticMax       int64
	cyclomaticMean      float64
	funcLengthSum       int64
	funcLengthMax       int64
	funcLengthMean      float64
	commentRatio        float64
}

func countDeclarations(dstFile *dst.File, stats *fileStats) {
	for _, decl := range dstFile.Decls {
		switch d := decl.(type) {
		case *dst.GenDecl:
			countGenDecl(d, stats)
		case *dst.FuncDecl:
			countFuncDecl(d, stats)
		}
	}
}

func countGenDecl(d *dst.GenDecl, stats *fileStats) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *dst.TypeSpec:
			countTypeSpec(s, stats)
		case *dst.ValueSpec:
			countValueSpec(s, d.Tok, stats)
		}
	}
}

func countTypeSpec(s *dst.TypeSpec, stats *fileStats) {
	pub := isPublic(s.Name.Name)

	stats.types++
	if pub {
		stats.publicTypes++
	} else {
		stats.privateTypes++
	}

	switch s.Type.(type) {
	case *dst.InterfaceType:
		stats.interfaces++
		if pub {
			stats.publicInterfaces++
		} else {
			stats.privateInterfaces++
		}
	case *dst.StructType:
		stats.structs++
		if pub {
			stats.publicStructs++
		} else {
			stats.privateStructs++
		}
	}
}

func countValueSpec(s *dst.ValueSpec, tok token.Token, stats *fileStats) {
	for _, name := range s.Names {
		pub := isPublic(name.Name)

		switch tok {
		case token.CONST:
			stats.constants++
			if pub {
				stats.publicConstants++
			} else {
				stats.privateConstants++
			}
		case token.VAR:
			stats.variables++
			if pub {
				stats.publicVariables++
			} else {
				stats.privateVariables++
			}
		}
	}
}

func countFuncDecl(d *dst.FuncDecl, stats *fileStats) {
	pub := isPublic(d.Name.Name)

	if d.Recv != nil && len(d.Recv.List) > 0 {
		stats.methods++
		if pub {
			stats.publicMethods++
		} else {
			stats.privateMethods++
		}
	} else {
		stats.functions++
		if pub {
			stats.publicFunctions++
		} else {
			stats.privateFunctions++
		}
	}
}

func (s *fileStats) computeAggregates() {
	s.declarations = s.types + s.functions + s.methods + s.constants + s.variables
	s.publicDeclarations = s.publicTypes + s.publicFunctions + s.publicMethods +
		s.publicConstants + s.publicVariables
	s.privateDeclarations = s.privateTypes + s.privateFunctions + s.privateMethods +
		s.privateConstants + s.privateVariables
}

func isPublic(name string) bool {
	return token.IsExported(name)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/provider/golang/ -run "TestCountDeclarations|TestComputeAggregates|TestIsPublic" -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/file_stats.go internal/provider/golang/file_stats_test.go
git commit -m "feat(golang): add Go declaration counting and fileStats struct (#289)"
```

---

### Task 4: Cyclomatic complexity

**Files:**
- Create: `internal/provider/golang/cyclomatic.go`
- Create: `internal/provider/golang/cyclomatic_test.go`

- [ ] **Step 1: Write the test**

```go
package golang

import (
	"go/token"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want int64
	}{
		{
			name: "empty function",
			src:  `package p; func f() {}`,
			want: 1,
		},
		{
			name: "single if",
			src:  `package p; func f() { if true {} }`,
			want: 2,
		},
		{
			name: "if with else",
			src:  `package p; func f() { if true {} else {} }`,
			want: 2,
		},
		{
			name: "for loop",
			src:  `package p; func f() { for i := 0; i < 10; i++ {} }`,
			want: 2,
		},
		{
			name: "range loop",
			src:  `package p; func f() { for range []int{} {} }`,
			want: 2,
		},
		{
			name: "switch with 2 cases",
			src:  `package p; func f() { switch { case true: case false: } }`,
			want: 3,
		},
		{
			name: "switch with default only",
			src:  `package p; func f() { switch { default: } }`,
			want: 1,
		},
		{
			name: "select with 2 cases",
			src: `package p
import "time"
func f() {
	ch := make(chan int)
	select {
	case <-ch:
	case <-time.After(0):
	}
}`,
			want: 3,
		},
		{
			name: "logical AND",
			src:  `package p; func f() { var a, b bool; if a && b {} }`,
			want: 3,
		},
		{
			name: "logical OR",
			src:  `package p; func f() { var a, b bool; if a || b {} }`,
			want: 3,
		},
		{
			name: "nested if-for",
			src: `package p
func f() {
	for i := 0; i < 10; i++ {
		if i > 5 {}
	}
}`,
			want: 3,
		},
		{
			name: "nil body",
			src:  `package p; func f()`,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			dec := decorator.NewDecorator(token.NewFileSet())
			dstFile, err := dec.Parse(tt.src)
			g.Expect(err).NotTo(HaveOccurred())

			var funcDecl *dst.FuncDecl
			for _, decl := range dstFile.Decls {
				if fd, ok := decl.(*dst.FuncDecl); ok {
					funcDecl = fd
					break
				}
			}
			g.Expect(funcDecl).NotTo(BeNil(), "no func decl found in source")

			g.Expect(cyclomaticComplexity(funcDecl.Body)).To(Equal(tt.want))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run TestCyclomaticComplexity -v -count=1
```

Expected: FAIL — `cyclomaticComplexity` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/provider/golang/cyclomatic.go`:

```go
package golang

import (
	"go/token"

	"github.com/dave/dst"
)

// cyclomaticComplexity computes cyclomatic complexity for a single function body.
// Base complexity is 1, plus 1 for each decision point:
// if, for, range, case (non-default), &&, ||.
func cyclomaticComplexity(body *dst.BlockStmt) int64 {
	if body == nil {
		return 1
	}

	complexity := int64(1)

	dst.Inspect(body, func(n dst.Node) bool {
		switch node := n.(type) {
		case *dst.IfStmt:
			complexity++
		case *dst.ForStmt:
			complexity++
		case *dst.RangeStmt:
			complexity++
		case *dst.CaseClause:
			if node.List != nil {
				complexity++
			}
		case *dst.CommClause:
			if node.Comm != nil {
				complexity++
			}
		case *dst.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}

		return true
	})

	return complexity
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/provider/golang/ -run TestCyclomaticComplexity -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/cyclomatic.go internal/provider/golang/cyclomatic_test.go
git commit -m "feat(golang): add cyclomatic complexity calculation (#289)"
```

---

### Task 5: Comment ratio

**Files:**
- Create: `internal/provider/golang/comments.go`
- Create: `internal/provider/golang/comments_test.go`

The comment ratio uses AST comment positions from the decorator's node mapping (not a separate parse). `go/ast` is imported only for type assertions — the comments were already parsed by the decorator's single parse pass.

- [ ] **Step 1: Write the test**

```go
package golang

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)

func TestComputeCommentRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want float64
	}{
		{
			name: "no comments",
			src:  "package p\n\nfunc f() {}\n",
			want: 0.0,
		},
		{
			name: "all comments",
			src:  "package p\n// comment only\n// another comment\n",
			want: 2.0,
		},
		{
			name: "mixed code and comments",
			src: `package p
// a comment
func f() {}
`,
			want: 0.5,
		},
		{
			name: "inline comment counts both",
			src: `package p
func f() {} // inline
`,
			want: 0.5,
		},
		{
			name: "block comment",
			src: `package p
/* block comment */
func f() {}
`,
			want: 0.5,
		},
		{
			name: "multi-line block comment",
			src: `package p
/*
multi-line
block
*/
func f() {}
`,
			want: 2.0,
		},
		{
			name: "blank lines ignored",
			src: `package p

// comment

func f() {}

`,
			want: 0.5,
		},
		{
			name: "code only",
			src: `package p

func f() {}
func g() {}
`,
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			fset := token.NewFileSet()
			dec := decorator.NewDecorator(fset)
			dstFile, err := dec.Parse(tt.src)
			g.Expect(err).NotTo(HaveOccurred())

			astFile := dec.Ast.Nodes[dstFile].(*ast.File)

			ratio := computeCommentRatio([]byte(tt.src), astFile.Comments, fset)
			g.Expect(ratio).To(BeNumerically("~", tt.want, 0.01))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run TestComputeCommentRatio -v -count=1
```

Expected: FAIL — `computeCommentRatio` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/provider/golang/comments.go`:

```go
package golang

import (
	"bytes"
	"go/ast"
	"go/token"
)

// computeCommentRatio computes the ratio of comment lines to code lines.
// Blank lines are excluded from both counts. Lines with both code and a comment
// count for both totals. Comment positions come from the AST produced by the
// decorator's single parse pass — not a separate parse.
func computeCommentRatio(
	src []byte,
	comments []*ast.CommentGroup,
	fset *token.FileSet,
) float64 {
	commentLineSet := buildCommentLineSet(comments, fset)
	commentOnlySet := buildCommentOnlyLineSet(src, comments, fset)

	srcLines := bytes.Split(src, []byte("\n"))

	var codeCount int64
	var commentCount int64

	for i, line := range srcLines {
		lineNum := i + 1

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		if commentLineSet[lineNum] {
			commentCount++
		}

		if !commentOnlySet[lineNum] {
			codeCount++
		}
	}

	if codeCount == 0 {
		return 0.0
	}

	return float64(commentCount) / float64(codeCount)
}

// buildCommentLineSet returns the set of line numbers that contain comment text.
func buildCommentLineSet(comments []*ast.CommentGroup, fset *token.FileSet) map[int]bool {
	set := make(map[int]bool)

	for _, cg := range comments {
		for _, c := range cg.List {
			start := fset.Position(c.Pos()).Line
			end := fset.Position(c.End()).Line

			for line := start; line <= end; line++ {
				set[line] = true
			}
		}
	}

	return set
}

// buildCommentOnlyLineSet returns the set of line numbers where the entire
// non-whitespace content is comment text (no code on the same line).
func buildCommentOnlyLineSet(
	src []byte,
	comments []*ast.CommentGroup,
	fset *token.FileSet,
) map[int]bool {
	set := make(map[int]bool)
	srcLines := bytes.Split(src, []byte("\n"))

	for _, cg := range comments {
		for _, c := range cg.List {
			startPos := fset.Position(c.Pos())
			endPos := fset.Position(c.End())

			// Interior lines of multi-line block comments are always comment-only.
			for line := startPos.Line + 1; line < endPos.Line; line++ {
				set[line] = true
			}

			// Check the start line: comment-only if trimmed line starts at comment.
			if startPos.Line <= len(srcLines) {
				line := srcLines[startPos.Line-1]
				trimmed := bytes.TrimSpace(line)

				if bytes.HasPrefix(trimmed, []byte("//")) || bytes.HasPrefix(trimmed, []byte("/*")) {
					set[startPos.Line] = true
				}
			}

			// End line of multi-line block: comment-only if no code follows "*/".
			if endPos.Line != startPos.Line && endPos.Line <= len(srcLines) {
				line := srcLines[endPos.Line-1]

				if idx := bytes.Index(line, []byte("*/")); idx >= 0 {
					after := bytes.TrimSpace(line[idx+2:])
					if len(after) == 0 {
						set[endPos.Line] = true
					}
				}
			}
		}
	}

	return set
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/provider/golang/ -run TestComputeCommentRatio -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/comments.go internal/provider/golang/comments_test.go
git commit -m "feat(golang): add comment ratio calculation (#289)"
```

---

### Task 6: Import classification

**Files:**
- Create: `internal/provider/golang/imports.go`
- Create: `internal/provider/golang/imports_test.go`

- [ ] **Step 1: Write the test**

```go
package golang

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)

func TestClassifyImports(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

import (
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/example/external"
	"github.com/other/pkg"

	"github.com/myorg/mymod/internal/foo"
	"github.com/myorg/mymod/pkg/bar"
)
`
	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	classifyImports(dstFile, "github.com/myorg/mymod", stats)

	g.Expect(stats.imports).To(Equal(int64(7)))
	g.Expect(stats.stdlibImports).To(Equal(int64(3)))
	g.Expect(stats.externalImports).To(Equal(int64(2)))
	g.Expect(stats.internalImports).To(Equal(int64(2)))
}

func TestClassifyImportsNoModule(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	src := `package example

import (
	"fmt"
	"github.com/other/pkg"
)
`
	dec := decorator.NewDecorator(token.NewFileSet())
	dstFile, err := dec.Parse(src)
	g.Expect(err).NotTo(HaveOccurred())

	stats := &fileStats{}
	classifyImports(dstFile, "", stats)

	g.Expect(stats.imports).To(Equal(int64(2)))
	g.Expect(stats.stdlibImports).To(Equal(int64(1)))
	g.Expect(stats.externalImports).To(Equal(int64(1)))
	g.Expect(stats.internalImports).To(Equal(int64(0)))
}

func TestIsStdlib(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(isStdlib("fmt")).To(BeTrue())
	g.Expect(isStdlib("net/http")).To(BeTrue())
	g.Expect(isStdlib("encoding/json")).To(BeTrue())
	g.Expect(isStdlib("github.com/foo/bar")).To(BeFalse())
	g.Expect(isStdlib("golang.org/x/sync")).To(BeFalse())
}

func TestFindModulePath(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "deep")
	_ = os.MkdirAll(sub, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/mod\n\ngo 1.26\n"), 0o600)

	mc := newModuleCache()

	path := mc.findModulePath(sub)
	g.Expect(path).To(Equal("github.com/test/mod"))

	// Cached: same result for parent dir
	path2 := mc.findModulePath(filepath.Join(dir, "sub"))
	g.Expect(path2).To(Equal("github.com/test/mod"))
}

func TestFindModulePathNoGoMod(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	mc := newModuleCache()

	path := mc.findModulePath(dir)
	g.Expect(path).To(Equal(""))
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run "TestClassifyImports|TestIsStdlib|TestFindModulePath" -v -count=1
```

Expected: FAIL — `classifyImports` not defined.

- [ ] **Step 3: Write the implementation**

Create `internal/provider/golang/imports.go`:

```go
package golang

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dave/dst"
)

// classifyImports categorizes each import in dstFile as stdlib, internal, or
// external, and populates the corresponding stats fields.
func classifyImports(dstFile *dst.File, modulePath string, stats *fileStats) {
	for _, imp := range dstFile.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		stats.imports++

		switch {
		case isStdlib(path):
			stats.stdlibImports++
		case modulePath != "" && strings.HasPrefix(path, modulePath):
			stats.internalImports++
		default:
			stats.externalImports++
		}
	}
}

// isStdlib reports whether importPath is a Go standard library package.
// Stdlib packages have no dot in the first path element.
func isStdlib(importPath string) bool {
	firstElem := importPath
	if i := strings.IndexByte(importPath, '/'); i >= 0 {
		firstElem = importPath[:i]
	}

	return !strings.Contains(firstElem, ".")
}

// moduleCache caches go.mod module path lookups per directory.
type moduleCache struct {
	mu      sync.RWMutex
	modules map[string]string
}

func newModuleCache() *moduleCache {
	return &moduleCache{
		modules: make(map[string]string),
	}
}

// findModulePath walks up from dir looking for go.mod and returns the module
// path. Returns "" if no go.mod is found. Results are cached per directory.
func (mc *moduleCache) findModulePath(dir string) string {
	mc.mu.RLock()
	if path, ok := mc.modules[dir]; ok {
		mc.mu.RUnlock()
		return path
	}
	mc.mu.RUnlock()

	result := mc.scanForModulePath(dir)

	return result
}

func (mc *moduleCache) scanForModulePath(startDir string) string {
	var visited []string

	dir := startDir
	for {
		mc.mu.RLock()
		if path, ok := mc.modules[dir]; ok {
			mc.mu.RUnlock()
			mc.cacheAll(visited, path)

			return path
		}
		mc.mu.RUnlock()

		visited = append(visited, dir)

		goModPath := filepath.Join(dir, "go.mod")
		if modPath := readModulePath(goModPath); modPath != "" {
			mc.cacheAll(visited, modPath)

			return modPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	mc.cacheAll(visited, "")

	return ""
}

func (mc *moduleCache) cacheAll(dirs []string, modulePath string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, d := range dirs {
		mc.modules[d] = modulePath
	}
}

// readModulePath reads the module path from a go.mod file.
// Returns "" if the file doesn't exist or doesn't contain a module directive.
func readModulePath(goModPath string) string {
	f, err := os.Open(goModPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/provider/golang/ -run "TestClassifyImports|TestIsStdlib|TestFindModulePath" -v -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/imports.go internal/provider/golang/imports_test.go
git commit -m "feat(golang): add import classification and module path lookup (#289)"
```

---

### Task 7: analyzeFile orchestration

**Files:**
- Modify: `internal/provider/golang/file_stats.go`
- Modify: `internal/provider/golang/file_stats_test.go`

This task wires together declaration counting, cyclomatic complexity, function length, comment ratio, and import classification into a single `analyzeFile` function. Function length uses the decorator's `Ast.Nodes` mapping to look up AST positions — this is accessing the result of the single `dst` parse, not a second parse.

- [ ] **Step 1: Write the test**

Append to `internal/provider/golang/file_stats_test.go`:

```go
func TestAnalyzeFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/example\n\ngo 1.26\n"), 0o600)

	src := `package example

import (
	"fmt"

	"github.com/other/pkg"

	"github.com/test/example/internal/foo"
)

// Greet returns a greeting.
type Greeter interface {
	Greet() string
}

type greeterImpl struct {
	name string
}

func (g *greeterImpl) Greet() string {
	return fmt.Sprintf("Hello, %s", g.name)
}

func Public() {
	if true {
		for i := 0; i < 10; i++ {
			fmt.Println(pkg.Do(), foo.Bar())
		}
	}
}

const ExportedConst = 1

var unexportedVar int
`
	goFile := filepath.Join(dir, "example.go")
	_ = os.WriteFile(goFile, []byte(src), 0o600)

	stats, err := analyzeFile(goFile, "github.com/test/example")
	g.Expect(err).NotTo(HaveOccurred())

	// Types
	g.Expect(stats.types).To(Equal(int64(2)))
	g.Expect(stats.publicTypes).To(Equal(int64(1)))
	g.Expect(stats.interfaces).To(Equal(int64(1)))
	g.Expect(stats.structs).To(Equal(int64(1)))

	// Functions and methods
	g.Expect(stats.functions).To(Equal(int64(1)))
	g.Expect(stats.methods).To(Equal(int64(1)))

	// Constants and variables
	g.Expect(stats.constants).To(Equal(int64(1)))
	g.Expect(stats.variables).To(Equal(int64(1)))

	// Imports
	g.Expect(stats.imports).To(Equal(int64(3)))
	g.Expect(stats.stdlibImports).To(Equal(int64(1)))
	g.Expect(stats.externalImports).To(Equal(int64(1)))
	g.Expect(stats.internalImports).To(Equal(int64(1)))

	// Cyclomatic: Greet()=1, Public()=3 (if + for)
	g.Expect(stats.cyclomaticSum).To(Equal(int64(4)))
	g.Expect(stats.cyclomaticMax).To(Equal(int64(3)))
	g.Expect(stats.cyclomaticMean).To(BeNumerically("~", 2.0, 0.01))

	// Function length > 0
	g.Expect(stats.funcLengthSum).To(BeNumerically(">", 0))
	g.Expect(stats.funcLengthMax).To(BeNumerically(">", 0))
	g.Expect(stats.funcLengthMean).To(BeNumerically(">", 0))

	// Comment ratio: 1 comment line / several code lines
	g.Expect(stats.commentRatio).To(BeNumerically(">", 0))
	g.Expect(stats.commentRatio).To(BeNumerically("<", 1.0))

	// Aggregates
	g.Expect(stats.declarations).To(Equal(int64(6)))
	g.Expect(stats.publicDeclarations).To(Equal(int64(3)))
	g.Expect(stats.privateDeclarations).To(Equal(int64(3)))
}

func TestAnalyzeFileNotGo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "bad.go"), []byte("not go code at all"), 0o600)

	_, err := analyzeFile(filepath.Join(dir, "bad.go"), "")
	g.Expect(err).To(HaveOccurred())
}
```

Add these imports at the top of `file_stats_test.go`:

```go
import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/dst/decorator"
	. "github.com/onsi/gomega"
)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/provider/golang/ -run TestAnalyzeFile -v -count=1
```

Expected: FAIL — `analyzeFile` not defined.

- [ ] **Step 3: Write the implementation**

Add to `internal/provider/golang/file_stats.go`:

```go
import (
	"go/ast"
	"go/token"
	"os"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/rotisserie/eris"
)

// analyzeFile parses a .go file with dst and extracts all metrics in a single
// pass. The modulePath is used for internal import classification.
func analyzeFile(path string, modulePath string) (*fileStats, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, eris.Wrapf(err, "reading Go file %s", path)
	}

	fset := token.NewFileSet()
	dec := decorator.NewDecorator(fset)

	dstFile, err := dec.ParseFile(path, src, 0)
	if err != nil {
		return nil, eris.Wrapf(err, "parsing Go file %s", path)
	}

	stats := &fileStats{}

	countDeclarations(dstFile, stats)
	computeFunctionMetrics(dstFile, dec, fset, stats)
	classifyImports(dstFile, modulePath, stats)

	astFile, ok := dec.Ast.Nodes[dstFile].(*ast.File)
	if ok {
		stats.commentRatio = computeCommentRatio(src, astFile.Comments, fset)
	}

	stats.computeAggregates()

	return stats, nil
}

// computeFunctionMetrics computes cyclomatic complexity and function length
// for all functions/methods, then aggregates to sum/max/mean.
func computeFunctionMetrics(
	dstFile *dst.File,
	dec *decorator.Decorator,
	fset *token.FileSet,
	stats *fileStats,
) {
	var complexities []int64
	var lengths []int64

	for _, decl := range dstFile.Decls {
		funcDecl, ok := decl.(*dst.FuncDecl)
		if !ok {
			continue
		}

		cc := cyclomaticComplexity(funcDecl.Body)
		complexities = append(complexities, cc)

		astNode := dec.Ast.Nodes[funcDecl]
		if astNode != nil {
			startLine := fset.Position(astNode.Pos()).Line
			endLine := fset.Position(astNode.End()).Line
			length := int64(endLine - startLine + 1)
			lengths = append(lengths, length)
		}
	}

	aggregateInt64s(complexities, &stats.cyclomaticSum, &stats.cyclomaticMax, &stats.cyclomaticMean)
	aggregateInt64s(lengths, &stats.funcLengthSum, &stats.funcLengthMax, &stats.funcLengthMean)
}

func aggregateInt64s(values []int64, sum *int64, max *int64, mean *float64) {
	if len(values) == 0 {
		return
	}

	for _, v := range values {
		*sum += v
		if v > *max {
			*max = v
		}
	}

	*mean = float64(*sum) / float64(len(values))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/provider/golang/ -run "TestAnalyzeFile|TestCountDeclarations|TestCyclomaticComplexity|TestComputeCommentRatio|TestClassifyImports" -v -count=1
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/file_stats.go internal/provider/golang/file_stats_test.go
git commit -m "feat(golang): add analyzeFile orchestration with function metrics (#289)"
```

---

### Task 8: Provider infrastructure and definitions

**Files:**
- Create: `internal/provider/golang/go_provider.go`
- Create: `internal/provider/golang/provider_defs.go`

- [ ] **Step 1: Write the provider infrastructure**

Create `internal/provider/golang/go_provider.go`:

```go
package golang

import (
	"log/slog"
	"path/filepath"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// goExtractor extracts one metric value from fileStats and sets it on the model file.
type goExtractor func(name metric.Name, stats *fileStats, f *model.File)

// goProvider is a data-driven implementation of provider.Interface for Go metrics.
type goProvider struct {
	name           metric.Name
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	extract        goExtractor
	onFile         func()
}

func (p *goProvider) Name() metric.Name                   { return p.name }
func (p *goProvider) Kind() metric.Kind                   { return p.kind }
func (p *goProvider) Description() string                 { return p.description }
func (*goProvider) Dependencies() []metric.Name           { return nil }
func (p *goProvider) DefaultPalette() palette.PaletteName { return p.defaultPalette }
func (p *goProvider) SetOnFileProcessed(fn func())        { p.onFile = fn }

func (p *goProvider) Load(root *model.Directory) error {
	walkGoFiles(root, p.name, p.onFile, p.extract)

	return nil
}

// providerDef holds the static fields for one goProvider.
type providerDef struct {
	kind           metric.Kind
	description    string
	defaultPalette palette.PaletteName
	extract        goExtractor
}

// newProvider creates a fresh goProvider for the given metric name.
func newProvider(name metric.Name) *goProvider {
	def, ok := providerDefs[name]
	if !ok {
		panic("newProvider: unknown Go metric name: " + string(name))
	}

	return &goProvider{
		name:           name,
		kind:           def.kind,
		description:    def.description,
		defaultPalette: def.defaultPalette,
		extract:        def.extract,
	}
}

// statsCache caches parsed fileStats per file path.
type statsCache struct {
	mu    sync.Mutex
	group singleflight.Group
	stats map[string]*fileStats
}

var globalCache = &statsCache{
	stats: make(map[string]*fileStats),
}

var globalModuleCache = newModuleCache()

// getOrAnalyze returns the cached fileStats for path, parsing if necessary.
// Concurrent requests for the same path are deduplicated via singleflight.
func getOrAnalyze(path string) (*fileStats, error) {
	globalCache.mu.Lock()
	if s, ok := globalCache.stats[path]; ok {
		globalCache.mu.Unlock()

		return s, nil
	}
	globalCache.mu.Unlock()

	result, err, _ := globalCache.group.Do(path, func() (any, error) {
		dir := filepath.Dir(path)
		modulePath := globalModuleCache.findModulePath(dir)

		s, err := analyzeFile(path, modulePath)
		if err != nil {
			return nil, err
		}

		globalCache.mu.Lock()
		globalCache.stats[path] = s
		globalCache.mu.Unlock()

		return s, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*fileStats), nil
}

// walkGoFiles walks all .go files under root and calls the extract function
// with cached fileStats for each. Non-.go files are silently skipped.
func walkGoFiles(
	root *model.Directory,
	name metric.Name,
	onFile func(),
	extract goExtractor,
) {
	model.WalkFiles(root, func(f *model.File) {
		if onFile != nil {
			defer onFile()
		}

		if f.Extension != "go" {
			return
		}

		stats, err := getOrAnalyze(f.Path)
		if err != nil {
			slog.Warn("could not analyze Go file", "path", f.Path, "error", err)

			return
		}

		extract(name, stats, f)
	})
}

// quantityField returns a goExtractor that reads an int64 field from fileStats.
func quantityField(fn func(*fileStats) int64) goExtractor {
	return func(name metric.Name, stats *fileStats, f *model.File) {
		f.SetQuantity(name, fn(stats))
	}
}

// measureField returns a goExtractor that reads a float64 field from fileStats.
func measureField(fn func(*fileStats) float64) goExtractor {
	return func(name metric.Name, stats *fileStats, f *model.File) {
		f.SetMeasure(name, fn(stats))
	}
}

// ResetCacheForTesting clears the global caches. Test use only.
func ResetCacheForTesting() {
	globalCache = &statsCache{
		stats: make(map[string]*fileStats),
	}
	globalModuleCache = newModuleCache()
}
```

- [ ] **Step 2: Write the provider definitions**

Create `internal/provider/golang/provider_defs.go`:

```go
package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// providerDefs is the authoritative map of all Go metric providers.
// Adding a new Go metric requires only a new entry here.
var providerDefs = map[metric.Name]providerDef{
	TypeCount: {
		kind:           metric.Quantity,
		description:    "Total type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.types }),
	},
	PublicTypeCount: {
		kind:           metric.Quantity,
		description:    "Exported type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicTypes }),
	},
	PrivateTypeCount: {
		kind:           metric.Quantity,
		description:    "Unexported type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateTypes }),
	},
	InterfaceCount: {
		kind:           metric.Quantity,
		description:    "Interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.interfaces }),
	},
	PublicInterfaceCount: {
		kind:           metric.Quantity,
		description:    "Exported interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicInterfaces }),
	},
	PrivateInterfaceCount: {
		kind:           metric.Quantity,
		description:    "Unexported interface type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateInterfaces }),
	},
	StructCount: {
		kind:           metric.Quantity,
		description:    "Struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.structs }),
	},
	PublicStructCount: {
		kind:           metric.Quantity,
		description:    "Exported struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicStructs }),
	},
	PrivateStructCount: {
		kind:           metric.Quantity,
		description:    "Unexported struct type declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateStructs }),
	},
	FunctionCount: {
		kind:           metric.Quantity,
		description:    "Function declarations (no receiver) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.functions }),
	},
	PublicFunctionCount: {
		kind:           metric.Quantity,
		description:    "Exported function declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicFunctions }),
	},
	PrivateFunctionCount: {
		kind:           metric.Quantity,
		description:    "Unexported function declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateFunctions }),
	},
	MethodCount: {
		kind:           metric.Quantity,
		description:    "Method declarations (with receiver) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.methods }),
	},
	PublicMethodCount: {
		kind:           metric.Quantity,
		description:    "Exported method declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicMethods }),
	},
	PrivateMethodCount: {
		kind:           metric.Quantity,
		description:    "Unexported method declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateMethods }),
	},
	ConstantCount: {
		kind:           metric.Quantity,
		description:    "Constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.constants }),
	},
	PublicConstantCount: {
		kind:           metric.Quantity,
		description:    "Exported constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicConstants }),
	},
	PrivateConstantCount: {
		kind:           metric.Quantity,
		description:    "Unexported constant declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateConstants }),
	},
	VariableCount: {
		kind:           metric.Quantity,
		description:    "Variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.variables }),
	},
	PublicVariableCount: {
		kind:           metric.Quantity,
		description:    "Exported variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicVariables }),
	},
	PrivateVariableCount: {
		kind:           metric.Quantity,
		description:    "Unexported variable declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateVariables }),
	},
	ImportCount: {
		kind:           metric.Quantity,
		description:    "Total import paths in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.imports }),
	},
	StdlibImportCount: {
		kind:           metric.Quantity,
		description:    "Standard library import count in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.stdlibImports }),
	},
	ExternalImportCount: {
		kind:           metric.Quantity,
		description:    "External (third-party) import count in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.externalImports }),
	},
	InternalImportCount: {
		kind:           metric.Quantity,
		description:    "Internal import count (same module) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.internalImports }),
	},
	DeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total declarations (types + functions + methods + constants + variables) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.declarations }),
	},
	PublicDeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total exported declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.publicDeclarations }),
	},
	PrivateDeclarationCount: {
		kind:           metric.Quantity,
		description:    "Total unexported declarations in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.privateDeclarations }),
	},
	CyclomaticComplexitySum: {
		kind:           metric.Quantity,
		description:    "Sum of cyclomatic complexity across all functions in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.cyclomaticSum }),
	},
	CyclomaticComplexityMax: {
		kind:           metric.Quantity,
		description:    "Maximum cyclomatic complexity of any single function in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.cyclomaticMax }),
	},
	CyclomaticComplexityMean: {
		kind:           metric.Measure,
		description:    "Mean cyclomatic complexity per function in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.cyclomaticMean }),
	},
	FunctionLengthSum: {
		kind:           metric.Quantity,
		description:    "Sum of function lengths (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.funcLengthSum }),
	},
	FunctionLengthMax: {
		kind:           metric.Quantity,
		description:    "Length of longest function (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        quantityField(func(s *fileStats) int64 { return s.funcLengthMax }),
	},
	FunctionLengthMean: {
		kind:           metric.Measure,
		description:    "Mean function length (lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.funcLengthMean }),
	},
	CommentRatio: {
		kind:           metric.Measure,
		description:    "Ratio of comment lines to code lines (ignoring blank lines) in Go files.",
		defaultPalette: palette.Neutral,
		extract:        measureField(func(s *fileStats) float64 { return s.commentRatio }),
	},
}
```

- [ ] **Step 3: Verify build**

```bash
go build ./internal/provider/golang/
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/golang/go_provider.go internal/provider/golang/provider_defs.go
git commit -m "feat(golang): add Go metric provider infrastructure with 35 definitions (#289)"
```

---

### Task 9: Registration and wiring

**Files:**
- Create: `internal/provider/golang/register.go`
- Modify: `cmd/codeviz/main.go:12-15` (imports), `cmd/codeviz/main.go:78-80` (Register calls)

- [ ] **Step 1: Create register.go**

```go
package golang

import "github.com/theunrepentantgeek/code-visualizer/internal/provider"

// Register adds all Go metric providers to the global registry.
func Register() {
	for name := range providerDefs {
		gp := newProvider(name)
		provider.Register(gp)
	}
}
```

- [ ] **Step 2: Add golang.Register() to main.go**

In `cmd/codeviz/main.go`, add the import:

```go
import (
	// ... existing imports ...
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)
```

And add the Register call after the existing ones:

```go
func main() {
	filesystem.Register()
	git.Register()
	golang.Register()
	// ... rest unchanged ...
```

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/codeviz/
```

Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/golang/register.go cmd/codeviz/main.go
git commit -m "feat(golang): register Go metric providers in CLI (#289)"
```

---

### Task 10: Integration test

**Files:**
- Create: `internal/provider/golang/go_provider_test.go`

- [ ] **Step 1: Write the integration test**

```go
package golang

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

func TestGoProviderIntegration(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

import (
	"fmt"
	"github.com/test/proj/internal/sub"
)

// Hello is exported.
type Hello struct{}

func (h *Hello) Greet() string {
	return fmt.Sprint(sub.Name())
}

func helper() {}
`
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# readme\n"), 0o600)

	f1 := &model.File{Path: filepath.Join(dir, "main.go"), Name: "main.go", Extension: "go"}
	f2 := &model.File{Path: filepath.Join(dir, "readme.md"), Name: "readme.md", Extension: "md"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f1, f2},
	}

	ResetCacheForTesting()

	// Run type-count provider
	p := newProvider(TypeCount)
	g.Expect(p.Name()).To(Equal(TypeCount))
	g.Expect(p.Kind()).To(Equal(metric.Quantity))
	g.Expect(p.Description()).NotTo(BeEmpty())
	g.Expect(p.DefaultPalette()).NotTo(BeEmpty())
	g.Expect(p.Dependencies()).To(BeNil())

	err := p.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	v, ok := f1.Quantity(TypeCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(1)))

	// Non-.go file should have no Go metrics
	_, ok = f2.Quantity(TypeCount)
	g.Expect(ok).To(BeFalse())
}

func TestGoProviderCacheConsistency(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/test/proj\n\ngo 1.26\n"), 0o600)

	src := `package proj

type Foo struct{}
type bar interface{}

func Public() {}
func private() {}
`
	_ = os.WriteFile(filepath.Join(dir, "code.go"), []byte(src), 0o600)

	f := &model.File{Path: filepath.Join(dir, "code.go"), Name: "code.go", Extension: "go"}
	root := &model.Directory{
		Path:  dir,
		Name:  "root",
		Files: []*model.File{f},
	}

	ResetCacheForTesting()

	// Run multiple providers on the same file
	typeP := newProvider(TypeCount)
	err := typeP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	funcP := newProvider(FunctionCount)
	err = funcP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	structP := newProvider(StructCount)
	err = structP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	interfaceP := newProvider(InterfaceCount)
	err = interfaceP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	commentP := newProvider(CommentRatio)
	err = commentP.Load(root)
	g.Expect(err).NotTo(HaveOccurred())

	// All metrics should be set from the same cached parse
	typeVal, ok := f.Quantity(TypeCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(typeVal).To(Equal(int64(2)))

	funcVal, ok := f.Quantity(FunctionCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(funcVal).To(Equal(int64(2)))

	structVal, ok := f.Quantity(StructCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(structVal).To(Equal(int64(1)))

	interfaceVal, ok := f.Quantity(InterfaceCount)
	g.Expect(ok).To(BeTrue())
	g.Expect(interfaceVal).To(Equal(int64(1)))

	ratioVal, ok := f.Measure(CommentRatio)
	g.Expect(ok).To(BeTrue())
	g.Expect(ratioVal).To(BeNumerically(">=", 0))
}

func TestGoProviderAllMetadata(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	for name := range providerDefs {
		p := newProvider(name)
		g.Expect(p.Name()).To(Equal(name), "name mismatch for "+string(name))
		g.Expect(p.Description()).NotTo(BeEmpty(), "empty description for "+string(name))
		g.Expect(p.DefaultPalette()).NotTo(BeEmpty(), "empty palette for "+string(name))
		g.Expect(p.Dependencies()).To(BeNil(), "unexpected dependencies for "+string(name))
	}
}
```

- [ ] **Step 2: Run the integration tests**

```bash
go test ./internal/provider/golang/ -run "TestGoProvider" -v -count=1
```

Expected: PASS.

- [ ] **Step 3: Run ALL tests in the package**

```bash
go test ./internal/provider/golang/ -v -count=1
```

Expected: all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/golang/go_provider_test.go
git commit -m "test(golang): add Go metric provider integration tests (#289)"
```

---

### Task 11: CI validation

- [ ] **Step 1: Run full test suite**

```bash
task test
```

Expected: all tests pass including new golang package tests.

- [ ] **Step 2: Run linting**

Run via subagent (high-volume output):

```bash
task lint
```

Expected: no new lint failures. Fix any issues that arise (likely formatting or funlen).

- [ ] **Step 3: Run full CI**

```bash
task ci
```

Expected: build + test + lint all pass.

- [ ] **Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(golang): address lint findings for Go metrics (#289)"
```
