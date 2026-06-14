# Metric Expressions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace flat metric names with a composable `[filter.]base-metric[.aggregation]` expression syntax, reducing the Go provider from 34 to 11 registrations while expanding available metric combinations.

**Architecture:** New types in `internal/metric/` define the expression grammar and aggregation functions. A parallel `BaseMetricDescriptor` registry coexists with the current `Interface` registry during migration, then replaces it. The resolution phase sits between config parsing and provider execution in the pipeline, validating expressions and planning computation. Providers migrate one at a time from flat registrations to base metric declarations.

**Tech Stack:** Go 1.26+, Gomega (assertions), eris (error wrapping), existing `internal/table` package (help output)

---

### Task 1: Add FilterName and AggregationName types

**Files:**
- Create: `internal/metric/filter.go`
- Create: `internal/metric/filter_test.go`
- Create: `internal/metric/aggregation_name.go`
- Create: `internal/metric/aggregation_name_test.go`

- [ ] **Step 1: Write the FilterName type and test**

```go
// internal/metric/filter_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestFilterName_StringConversion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := FilterName("public")
	g.Expect(string(f)).To(Equal("public"))
}

func TestFilterName_EmptyIsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var f FilterName
	g.Expect(f.IsZero()).To(BeTrue())
	g.Expect(FilterName("public").IsZero()).To(BeFalse())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -run TestFilterName -v`
Expected: FAIL — `FilterName` type does not exist.

- [ ] **Step 3: Implement FilterName**

```go
// internal/metric/filter.go
package metric

// FilterName identifies a filter/qualifier applied to a base metric (e.g., "public", "stdlib").
type FilterName string

// IsZero reports whether the filter name is empty (no filter applied).
func (f FilterName) IsZero() bool {
	return f == ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metric/ -run TestFilterName -v`
Expected: PASS

- [ ] **Step 5: Write the AggregationName type and test**

```go
// internal/metric/aggregation_name_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAggregationName_StringConversion(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := AggregationName("sum")
	g.Expect(string(a)).To(Equal("sum"))
}

func TestAggregationName_IsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var a AggregationName
	g.Expect(a.IsZero()).To(BeTrue())
	g.Expect(AggregationName("sum").IsZero()).To(BeFalse())
}

func TestAggregationName_IsKnown(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregationName("sum").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("min").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("max").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("mean").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("count").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("mode").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("distinct").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("range").IsKnown()).To(BeTrue())
	g.Expect(AggregationName("bogus").IsKnown()).To(BeFalse())
	g.Expect(AggregationName("").IsKnown()).To(BeFalse())
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/metric/ -run TestAggregationName -v`
Expected: FAIL — `AggregationName` type does not exist.

- [ ] **Step 7: Implement AggregationName**

```go
// internal/metric/aggregation_name.go
package metric

// AggregationName identifies an aggregation function (e.g., "sum", "max", "mean").
type AggregationName string

// Standard aggregation names.
const (
	AggSum      AggregationName = "sum"
	AggMin      AggregationName = "min"
	AggMax      AggregationName = "max"
	AggMean     AggregationName = "mean"
	AggCount    AggregationName = "count"
	AggMode     AggregationName = "mode"
	AggDistinct AggregationName = "distinct"
	AggRange    AggregationName = "range"
)

// knownAggregations is the fixed set of valid aggregation verbs.
var knownAggregations = map[AggregationName]struct{}{
	AggSum:      {},
	AggMin:      {},
	AggMax:      {},
	AggMean:     {},
	AggCount:    {},
	AggMode:     {},
	AggDistinct: {},
	AggRange:    {},
}

// IsZero reports whether the aggregation name is empty (no aggregation).
func (a AggregationName) IsZero() bool {
	return a == ""
}

// IsKnown reports whether the aggregation name is one of the recognized verbs.
func (a AggregationName) IsKnown() bool {
	_, ok := knownAggregations[a]
	return ok
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/metric/ -run TestAggregationName -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/metric/filter.go internal/metric/filter_test.go \
        internal/metric/aggregation_name.go internal/metric/aggregation_name_test.go
git commit -m "feat(metric): add FilterName and AggregationName types

FilterName identifies filters/qualifiers (e.g. 'public', 'stdlib').
AggregationName identifies aggregation verbs (sum, min, max, mean,
count, mode, distinct, range) with an IsKnown() validator."
```

---

### Task 2: Add MetricLevel type

**Files:**
- Create: `internal/metric/level.go`
- Create: `internal/metric/level_test.go`

- [ ] **Step 1: Write tests for MetricLevel**

```go
// internal/metric/level_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestMetricLevel_String(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(LevelFile.String()).To(Equal("file"))
	g.Expect(LevelDeclaration.String()).To(Equal("declaration"))
	g.Expect(LevelCommit.String()).To(Equal("commit"))
	g.Expect(LevelDirectory.String()).To(Equal("directory"))
}

func TestMetricLevel_UnknownString(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(MetricLevel(99).String()).To(Equal("unknown"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -run TestMetricLevel -v`
Expected: FAIL — `MetricLevel` type does not exist.

- [ ] **Step 3: Implement MetricLevel**

```go
// internal/metric/level.go
package metric

// MetricLevel identifies where raw data lives in the model hierarchy.
type MetricLevel int

const (
	LevelFile        MetricLevel = iota // native to files (file-size, file-lines)
	LevelDeclaration                    // native to declarations (cyclomatic-complexity)
	LevelCommit                         // native to commits (commit-date)
	LevelDirectory                      // native to directories (computed aggregates)
)

// String returns the human-readable name of the level.
func (l MetricLevel) String() string {
	switch l {
	case LevelFile:
		return "file"
	case LevelDeclaration:
		return "declaration"
	case LevelCommit:
		return "commit"
	case LevelDirectory:
		return "directory"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metric/ -run TestMetricLevel -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metric/level.go internal/metric/level_test.go
git commit -m "feat(metric): add MetricLevel type

Identifies where raw metric data lives in the model hierarchy:
file, declaration, commit, or directory."
```

---

### Task 3: Add MetricExpression type and parser

**Files:**
- Create: `internal/metric/expression.go`
- Create: `internal/metric/expression_test.go`

- [ ] **Step 1: Write tests for expression parsing**

```go
// internal/metric/expression_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseExpression_BareMetricName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("file-size")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("")))
	g.Expect(expr.Base).To(Equal(Name("file-size")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("")))
}

func TestParseExpression_MetricWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("file-size.sum")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("")))
	g.Expect(expr.Base).To(Equal(Name("file-size")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("sum")))
}

func TestParseExpression_FullExpression(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr, err := ParseExpression("public.types.count")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("public")))
	g.Expect(expr.Base).To(Equal(Name("types")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("count")))
}

func TestParseExpression_TwoSegmentsNonAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "public.types" — last segment "types" is not a known aggregation
	// so this is filter="public", base="types", aggregation=""
	expr, err := ParseExpression("public.types")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(expr.Filter).To(Equal(FilterName("public")))
	g.Expect(expr.Base).To(Equal(Name("types")))
	g.Expect(expr.Aggregation).To(Equal(AggregationName("")))
}

func TestParseExpression_EmptyStringReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty"))
}

func TestParseExpression_TooManySegmentsReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("a.b.c.d")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("too many segments"))
}

func TestParseExpression_InvalidCharactersReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := ParseExpression("file size")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid"))
}

func TestMetricExpression_String_BareMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Base: "file-size"}
	g.Expect(expr.String()).To(Equal("file-size"))
}

func TestMetricExpression_String_WithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Base: "file-size", Aggregation: "sum"}
	g.Expect(expr.String()).To(Equal("file-size.sum"))
}

func TestMetricExpression_String_Full(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Filter: "public", Base: "types", Aggregation: "count"}
	g.Expect(expr.String()).To(Equal("public.types.count"))
}

func TestMetricExpression_ResultName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	expr := MetricExpression{Filter: "public", Base: "types", Aggregation: "count"}
	g.Expect(expr.ResultName()).To(Equal(Name("public.types.count")))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -run "TestParseExpression|TestMetricExpression" -v`
Expected: FAIL — `ParseExpression` and `MetricExpression` do not exist.

- [ ] **Step 3: Implement MetricExpression and parser**

```go
// internal/metric/expression.go
package metric

import (
	"regexp"
	"strings"

	"github.com/rotisserie/eris"
)

// MetricExpression is the parsed form of a user-provided metric string.
// Format: [filter.]base-metric[.aggregation]
type MetricExpression struct {
	Filter      FilterName
	Base        Name
	Aggregation AggregationName
}

// validSegment matches a kebab-case identifier: lowercase letters, digits, and hyphens.
var validSegment = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ParseExpression parses a metric expression string into its components.
// The grammar is: [filter.]base-name[.aggregation]
// where each component is a kebab-case word separated by dots.
func ParseExpression(input string) (MetricExpression, error) {
	if input == "" {
		return MetricExpression{}, eris.New("metric expression must not be empty")
	}

	segments := strings.Split(input, ".")

	if len(segments) > 3 {
		return MetricExpression{}, eris.Errorf(
			"metric expression %q has too many segments (max 3: filter.base.aggregation)", input)
	}

	for _, seg := range segments {
		if !validSegment.MatchString(seg) {
			return MetricExpression{}, eris.Errorf(
				"metric expression %q contains invalid segment %q (must be lowercase kebab-case)", input, seg)
		}
	}

	switch len(segments) {
	case 1:
		return MetricExpression{Base: Name(segments[0])}, nil
	case 2:
		return parseTwoSegments(segments)
	case 3:
		return parseThreeSegments(segments)
	default:
		return MetricExpression{}, eris.Errorf("metric expression %q is invalid", input)
	}
}

func parseTwoSegments(segments []string) (MetricExpression, error) {
	last := AggregationName(segments[1])
	if last.IsKnown() {
		return MetricExpression{
			Base:        Name(segments[0]),
			Aggregation: last,
		}, nil
	}

	// Last segment is not a known aggregation — treat as filter.base
	return MetricExpression{
		Filter: FilterName(segments[0]),
		Base:   Name(segments[1]),
	}, nil
}

func parseThreeSegments(segments []string) (MetricExpression, error) {
	last := AggregationName(segments[2])
	if !last.IsKnown() {
		return MetricExpression{}, eris.Errorf(
			"metric expression segment %q is not a known aggregation function", segments[2])
	}

	return MetricExpression{
		Filter:      FilterName(segments[0]),
		Base:        Name(segments[1]),
		Aggregation: last,
	}, nil
}

// String returns the canonical string form of the expression.
func (e MetricExpression) String() string {
	var b strings.Builder

	if !e.Filter.IsZero() {
		b.WriteString(string(e.Filter))
		b.WriteByte('.')
	}

	b.WriteString(string(e.Base))

	if !e.Aggregation.IsZero() {
		b.WriteByte('.')
		b.WriteString(string(e.Aggregation))
	}

	return b.String()
}

// ResultName returns the metric.Name used as the storage key in MetricContainer.
func (e MetricExpression) ResultName() Name {
	return Name(e.String())
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metric/ -run "TestParseExpression|TestMetricExpression" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metric/expression.go internal/metric/expression_test.go
git commit -m "feat(metric): add MetricExpression parser

Parses [filter.]base-metric[.aggregation] syntax using dot as the
structural separator. Validates kebab-case segments and recognizes
known aggregation verbs."
```

---

### Task 4: Add aggregation function implementations

**Files:**
- Create: `internal/metric/aggregation.go`
- Create: `internal/metric/aggregation_test.go`

- [ ] **Step 1: Write tests for numeric aggregation functions**

```go
// internal/metric/aggregation_test.go
package metric

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestAggregateSum_IntValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateSum([]float64{1, 2, 3, 4})).To(Equal(10.0))
}

func TestAggregateSum_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateSum(nil)).To(Equal(0.0))
}

func TestAggregateMin(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMin([]float64{5, 2, 8, 1, 7})).To(Equal(1.0))
}

func TestAggregateMin_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMin(nil)).To(Equal(0.0))
}

func TestAggregateMax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMax([]float64{5, 2, 8, 1, 7})).To(Equal(8.0))
}

func TestAggregateMax_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMax(nil)).To(Equal(0.0))
}

func TestAggregateMean(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMean([]float64{2, 4, 6})).To(Equal(4.0))
}

func TestAggregateMean_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMean(nil)).To(Equal(0.0))
}

func TestAggregateCount(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateCount([]float64{1, 2, 3})).To(Equal(3.0))
}

func TestAggregateCount_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateCount(nil)).To(Equal(0.0))
}

func TestAggregateRange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange([]float64{3, 1, 7, 2})).To(Equal(6.0))
}

func TestAggregateRange_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange(nil)).To(Equal(0.0))
}

func TestAggregateRange_SingleElement(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateRange([]float64{5})).To(Equal(0.0))
}

func TestAggregateMode(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMode([]string{"go", "md", "go", "py", "go"})).To(Equal("go"))
}

func TestAggregateMode_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateMode(nil)).To(Equal(""))
}

func TestAggregateMode_Tie_ReturnsLexFirst(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "go" and "py" both appear twice; "go" is lexicographically first
	g.Expect(AggregateMode([]string{"go", "py", "go", "py"})).To(Equal("go"))
}

func TestAggregateDistinct(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateDistinct([]string{"go", "md", "go", "py", "md"})).To(Equal(3))
}

func TestAggregateDistinct_EmptySlice(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(AggregateDistinct(nil)).To(Equal(0))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/metric/ -run "TestAggregate" -v`
Expected: FAIL — aggregation functions do not exist.

- [ ] **Step 3: Implement aggregation functions**

```go
// internal/metric/aggregation.go
package metric

import (
	"maps"
	"math"
	"slices"
)

// AggregateSum returns the sum of all values.
func AggregateSum(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}

	return total
}

// AggregateMin returns the minimum value, or 0 if empty.
func AggregateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	result := math.MaxFloat64
	for _, v := range values {
		if v < result {
			result = v
		}
	}

	return result
}

// AggregateMax returns the maximum value, or 0 if empty.
func AggregateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	result := -math.MaxFloat64
	for _, v := range values {
		if v > result {
			result = v
		}
	}

	return result
}

// AggregateMean returns the arithmetic mean, or 0 if empty.
func AggregateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	return AggregateSum(values) / float64(len(values))
}

// AggregateCount returns the number of values.
func AggregateCount(values []float64) float64 {
	return float64(len(values))
}

// AggregateRange returns max − min, or 0 if fewer than 2 values.
func AggregateRange(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	return AggregateMax(values) - AggregateMin(values)
}

// AggregateMode returns the most common string value.
// On a tie, returns the lexicographically first tied value.
// Returns "" for an empty slice.
func AggregateMode(values []string) string {
	if len(values) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, v := range values {
		counts[v]++
	}

	best := ""
	bestCount := 0

	for _, key := range slices.Sorted(maps.Keys(counts)) {
		if counts[key] > bestCount {
			best = key
			bestCount = counts[key]
		}
	}

	return best
}

// AggregateDistinct returns the number of distinct string values.
func AggregateDistinct(values []string) int {
	if len(values) == 0 {
		return 0
	}

	unique := make(map[string]struct{})
	for _, v := range values {
		unique[v] = struct{}{}
	}

	return len(unique)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/metric/ -run "TestAggregate" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/metric/aggregation.go internal/metric/aggregation_test.go
git commit -m "feat(metric): add generic aggregation functions

Implements sum, min, max, mean, count, range for numeric values,
and mode, distinct for string (classification) values."
```

---

### Task 5: Add BaseMetricDescriptor and ProviderDescriptor types

**Files:**
- Create: `internal/provider/base_descriptor.go`
- Create: `internal/provider/base_descriptor_test.go`

- [ ] **Step 1: Write tests for BaseMetricDescriptor**

```go
// internal/provider/base_descriptor_test.go
package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestProviderDescriptor_HasFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	pd := ProviderDescriptor{
		Name: "go",
		Filters: map[metric.FilterName]string{
			"public":  "Exported declarations only",
			"private": "Unexported declarations only",
		},
	}

	g.Expect(pd.HasFilter("public")).To(BeTrue())
	g.Expect(pd.HasFilter("private")).To(BeTrue())
	g.Expect(pd.HasFilter("stdlib")).To(BeFalse())
}

func TestBaseMetricDescriptor_SupportsAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	bmd := BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Description:  "Size of each file in bytes.",
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}

	g.Expect(bmd.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(bmd.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(bmd.SupportsAggregation(metric.AggMode)).To(BeFalse())
}

func TestBaseMetricDescriptor_SupportsFilter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	bmd := BaseMetricDescriptor{
		Name:    "types",
		Kind:    metric.Quantity,
		Level:   metric.LevelDeclaration,
		Filters: []metric.FilterName{"public", "private"},
	}

	g.Expect(bmd.SupportsFilter("public")).To(BeTrue())
	g.Expect(bmd.SupportsFilter("private")).To(BeTrue())
	g.Expect(bmd.SupportsFilter("stdlib")).To(BeFalse())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/ -run "TestProviderDescriptor_HasFilter|TestBaseMetricDescriptor" -v`
Expected: FAIL — types do not exist.

- [ ] **Step 3: Implement BaseMetricDescriptor and ProviderDescriptor**

```go
// internal/provider/base_descriptor.go
package provider

import (
	"slices"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// ProviderDescriptor declares the shared metadata for a metric provider package,
// including the filter vocabulary available to all metrics in that provider.
type ProviderDescriptor struct {
	Name    string
	Filters map[metric.FilterName]string // filter name → human description
}

// HasFilter reports whether this provider defines the given filter name.
func (pd ProviderDescriptor) HasFilter(name metric.FilterName) bool {
	_, ok := pd.Filters[name]
	return ok
}

// BaseMetricDescriptor is the static metadata for a composable base metric.
type BaseMetricDescriptor struct {
	Name           metric.Name
	Kind           metric.Kind
	Level          metric.MetricLevel
	Description    string
	Filters        []metric.FilterName
	Aggregations   []metric.AggregationName
	Dependencies   []metric.Name
	DefaultPalette palette.PaletteName
}

// SupportsAggregation reports whether this base metric declares the given
// aggregation as valid.
func (d BaseMetricDescriptor) SupportsAggregation(agg metric.AggregationName) bool {
	return slices.Contains(d.Aggregations, agg)
}

// SupportsFilter reports whether this base metric declares the given filter
// as valid.
func (d BaseMetricDescriptor) SupportsFilter(filter metric.FilterName) bool {
	return slices.Contains(d.Filters, filter)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run "TestProviderDescriptor_HasFilter|TestBaseMetricDescriptor" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/base_descriptor.go internal/provider/base_descriptor_test.go
git commit -m "feat(provider): add BaseMetricDescriptor and ProviderDescriptor types

BaseMetricDescriptor declares a base metric's level, valid filters,
and valid aggregations. ProviderDescriptor holds the shared filter
vocabulary for a provider package."
```

---

### Task 6: Add base metric registry

**Files:**
- Create: `internal/provider/base_registry.go`
- Create: `internal/provider/base_registry_test.go`

- [ ] **Step 1: Write tests for the base metric registry**

```go
// internal/provider/base_registry_test.go
package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestBaseRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	desc := BaseMetricDescriptor{
		Name:           "file-size",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Size of each file in bytes.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax},
		DefaultPalette: palette.Neutral,
	}

	reg.register(desc)

	got, ok := reg.get("file-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(got.Name).To(Equal(metric.Name("file-size")))
	g.Expect(got.Level).To(Equal(metric.LevelFile))
}

func TestBaseRegistry_GetUnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()

	_, ok := reg.get("nonexistent")
	g.Expect(ok).To(BeFalse())
}

func TestBaseRegistry_DuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	desc := BaseMetricDescriptor{
		Name: "file-size",
		Kind: metric.Quantity,
	}

	reg.register(desc)
	g.Expect(func() { reg.register(desc) }).To(Panic())
}

func TestBaseRegistry_All(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{Name: "b-metric", Kind: metric.Quantity})
	reg.register(BaseMetricDescriptor{Name: "a-metric", Kind: metric.Measure})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
	g.Expect(all[0].Name).To(Equal(metric.Name("a-metric")))
	g.Expect(all[1].Name).To(Equal(metric.Name("b-metric")))
}

func TestBaseRegistry_AllForLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{Name: "file-size", Level: metric.LevelFile})
	reg.register(BaseMetricDescriptor{Name: "types", Level: metric.LevelDeclaration})
	reg.register(BaseMetricDescriptor{Name: "file-lines", Level: metric.LevelFile})

	fileMetrics := reg.allForLevel(metric.LevelFile)
	g.Expect(fileMetrics).To(HaveLen(2))

	declMetrics := reg.allForLevel(metric.LevelDeclaration)
	g.Expect(declMetrics).To(HaveLen(1))
	g.Expect(declMetrics[0].Name).To(Equal(metric.Name("types")))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/ -run "TestBaseRegistry" -v`
Expected: FAIL — `newBaseRegistry` does not exist.

- [ ] **Step 3: Implement base registry**

```go
// internal/provider/base_registry.go
package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// baseRegistry holds registered base metric descriptors.
type baseRegistry struct {
	mu          sync.RWMutex
	descriptors map[metric.Name]BaseMetricDescriptor
	providers   map[metric.Name]ProviderDescriptor // keyed by base metric name → owning provider
}

func newBaseRegistry() *baseRegistry {
	return &baseRegistry{
		descriptors: make(map[metric.Name]BaseMetricDescriptor),
		providers:   make(map[metric.Name]ProviderDescriptor),
	}
}

func (r *baseRegistry) register(desc BaseMetricDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[desc.Name]; exists {
		panic(fmt.Sprintf("base metric %q already registered", desc.Name))
	}

	r.descriptors[desc.Name] = desc
}

func (r *baseRegistry) registerWithProvider(desc BaseMetricDescriptor, pd ProviderDescriptor) {
	r.register(desc)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[desc.Name] = pd
}

func (r *baseRegistry) get(name metric.Name) (BaseMetricDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.descriptors[name]
	return d, ok
}

func (r *baseRegistry) providerFor(name metric.Name) (ProviderDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pd, ok := r.providers[name]
	return pd, ok
}

func (r *baseRegistry) all() []BaseMetricDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := slices.Collect(maps.Values(r.descriptors))
	slices.SortFunc(result, func(a, b BaseMetricDescriptor) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

func (r *baseRegistry) allForLevel(level metric.MetricLevel) []BaseMetricDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []BaseMetricDescriptor
	for _, d := range r.descriptors {
		if d.Level == level {
			result = append(result, d)
		}
	}

	slices.SortFunc(result, func(a, b BaseMetricDescriptor) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

func (r *baseRegistry) names() []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := slices.Collect(maps.Keys(r.descriptors))
	slices.SortFunc(names, cmp.Compare)

	return names
}

// globalBaseRegistry is the process-wide base metric registry.
var globalBaseRegistry = newBaseRegistry()

// RegisterBase adds a base metric descriptor to the global registry.
// Panics on duplicate name.
func RegisterBase(desc BaseMetricDescriptor) {
	globalBaseRegistry.register(desc)
}

// RegisterBaseWithProvider adds a base metric descriptor and associates it
// with the given provider descriptor.
func RegisterBaseWithProvider(desc BaseMetricDescriptor, pd ProviderDescriptor) {
	globalBaseRegistry.registerWithProvider(desc, pd)
}

// GetBase retrieves a base metric descriptor by name.
func GetBase(name metric.Name) (BaseMetricDescriptor, bool) {
	return globalBaseRegistry.get(name)
}

// GetBaseProvider retrieves the provider descriptor for a base metric.
func GetBaseProvider(name metric.Name) (ProviderDescriptor, bool) {
	return globalBaseRegistry.providerFor(name)
}

// AllBase returns all registered base metric descriptors, sorted by name.
func AllBase() []BaseMetricDescriptor {
	return globalBaseRegistry.all()
}

// AllBaseForLevel returns base metrics at the given native level.
func AllBaseForLevel(level metric.MetricLevel) []BaseMetricDescriptor {
	return globalBaseRegistry.allForLevel(level)
}

// BaseNames returns the sorted names of all registered base metrics.
func BaseNames() []metric.Name {
	return globalBaseRegistry.names()
}

// ResetBaseRegistryForTesting clears the global base registry. Test use only.
func ResetBaseRegistryForTesting() {
	globalBaseRegistry = newBaseRegistry()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run "TestBaseRegistry" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/base_registry.go internal/provider/base_registry_test.go
git commit -m "feat(provider): add base metric registry

Stores BaseMetricDescriptor entries with ProviderDescriptor association.
Supports lookup by name, listing by level, and provider retrieval.
Coexists with the existing Interface registry during migration."
```

---

### Task 7: Add expression resolution and validation

**Files:**
- Create: `internal/provider/resolution.go`
- Create: `internal/provider/resolution_test.go`

- [ ] **Step 1: Write tests for expression resolution**

```go
// internal/provider/resolution_test.go
package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestResolveExpression_BareMetricAtNativeLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:           "file-size",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	})

	expr := metric.MetricExpression{Base: "file-size"}
	resolved, err := resolveExpression(reg, expr, metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.Expression).To(Equal(expr))
	g.Expect(resolved.ResultName).To(Equal(metric.Name("file-size")))
	g.Expect(resolved.NeedsAggregation).To(BeFalse())
}

func TestResolveExpression_MetricWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggSum}
	resolved, err := resolveExpression(reg, expr, metric.LevelDirectory)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.NeedsAggregation).To(BeTrue())
	g.Expect(resolved.ResultName).To(Equal(metric.Name("file-size.sum")))
	g.Expect(resolved.ResultKind).To(Equal(metric.Quantity))
}

func TestResolveExpression_MeanChangesKindToMeasure(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggMean}
	resolved, err := resolveExpression(reg, expr, metric.LevelDirectory)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultKind).To(Equal(metric.Measure))
}

func TestResolveExpression_InvalidAggregationError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "file-size",
		Kind:         metric.Quantity,
		Level:        metric.LevelFile,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "file-size", Aggregation: metric.AggMode}
	_, err := resolveExpression(reg, expr, metric.LevelDirectory)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("mode"))
	g.Expect(err.Error()).To(ContainSubstring("not a valid aggregation"))
	g.Expect(err.Error()).To(ContainSubstring("sum, min, max, mean"))
}

func TestResolveExpression_InvalidFilterError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:    "types",
		Kind:    metric.Quantity,
		Level:   metric.LevelDeclaration,
		Filters: []metric.FilterName{"public", "private"},
		Aggregations: []metric.AggregationName{metric.AggCount},
	})

	expr := metric.MetricExpression{Filter: "stdlib", Base: "types", Aggregation: metric.AggCount}
	_, err := resolveExpression(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("stdlib"))
	g.Expect(err.Error()).To(ContainSubstring("not a valid filter"))
}

func TestResolveExpression_MissingAggregationAtHigherLevel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "cyclomatic-complexity",
		Kind:         metric.Quantity,
		Level:        metric.LevelDeclaration,
		Aggregations: []metric.AggregationName{metric.AggSum, metric.AggMax, metric.AggMean},
	})

	expr := metric.MetricExpression{Base: "cyclomatic-complexity"}
	_, err := resolveExpression(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("requires aggregation"))
	g.Expect(err.Error()).To(ContainSubstring("sum, max, mean"))
}

func TestResolveExpression_UnknownBaseMetricError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()

	expr := metric.MetricExpression{Base: "nonexistent"}
	_, err := resolveExpression(reg, expr, metric.LevelFile)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown base metric"))
}

func TestResolveExpression_FilterWithAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newBaseRegistry()
	reg.register(BaseMetricDescriptor{
		Name:         "types",
		Kind:         metric.Quantity,
		Level:        metric.LevelDeclaration,
		Filters:      []metric.FilterName{"public", "private"},
		Aggregations: []metric.AggregationName{metric.AggCount, metric.AggSum},
	})

	expr := metric.MetricExpression{Filter: "public", Base: "types", Aggregation: metric.AggCount}
	resolved, err := resolveExpression(reg, expr, metric.LevelFile)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resolved.ResultName).To(Equal(metric.Name("public.types.count")))
	g.Expect(resolved.NeedsAggregation).To(BeTrue())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/ -run "TestResolveExpression" -v`
Expected: FAIL — `resolveExpression` does not exist.

- [ ] **Step 3: Implement resolution logic**

```go
// internal/provider/resolution.go
package provider

import (
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// ResolvedMetric is a fully validated metric ready for computation.
type ResolvedMetric struct {
	Expression       metric.MetricExpression
	Descriptor       BaseMetricDescriptor
	SourceLevel      metric.MetricLevel
	TargetLevel      metric.MetricLevel
	ResultKind       metric.Kind
	ResultName       metric.Name
	NeedsAggregation bool
}

// ResolveExpression validates and resolves a parsed metric expression against
// the global base metric registry for the given target level.
func ResolveExpression(expr metric.MetricExpression, targetLevel metric.MetricLevel) (ResolvedMetric, error) {
	return resolveExpression(globalBaseRegistry, expr, targetLevel)
}

func resolveExpression(
	reg *baseRegistry,
	expr metric.MetricExpression,
	targetLevel metric.MetricLevel,
) (ResolvedMetric, error) {
	desc, ok := reg.get(expr.Base)
	if !ok {
		return ResolvedMetric{}, eris.Errorf("unknown base metric %q", expr.Base)
	}

	if err := validateFilter(desc, expr.Filter); err != nil {
		return ResolvedMetric{}, err
	}

	needsAgg := !expr.Aggregation.IsZero() || targetLevel > desc.Level
	if err := validateAggregation(desc, expr, targetLevel, needsAgg); err != nil {
		return ResolvedMetric{}, err
	}

	resultKind := computeResultKind(desc.Kind, expr.Aggregation)

	return ResolvedMetric{
		Expression:       expr,
		Descriptor:       desc,
		SourceLevel:      desc.Level,
		TargetLevel:      targetLevel,
		ResultKind:       resultKind,
		ResultName:       expr.ResultName(),
		NeedsAggregation: needsAgg,
	}, nil
}

func validateFilter(desc BaseMetricDescriptor, filter metric.FilterName) error {
	if filter.IsZero() {
		return nil
	}

	if !desc.SupportsFilter(filter) {
		if len(desc.Filters) == 0 {
			return eris.Errorf(
				"%q is not a valid filter for %q; %q has no filters",
				filter, desc.Name, desc.Name)
		}

		return eris.Errorf(
			"%q is not a valid filter for %q; valid filters: %s",
			filter, desc.Name, formatFilterNames(desc.Filters))
	}

	return nil
}

func validateAggregation(
	desc BaseMetricDescriptor,
	expr metric.MetricExpression,
	targetLevel metric.MetricLevel,
	needsAgg bool,
) error {
	if !expr.Aggregation.IsZero() && !desc.SupportsAggregation(expr.Aggregation) {
		return eris.Errorf(
			"%q is not a valid aggregation for %q; valid aggregations: %s",
			expr.Aggregation, desc.Name, formatAggregationNames(desc.Aggregations))
	}

	if needsAgg && expr.Aggregation.IsZero() && targetLevel > desc.Level {
		return eris.Errorf(
			"metric %q requires aggregation at %s level (native level: %s); try: %s",
			desc.Name,
			targetLevel.String(),
			desc.Level.String(),
			formatAggregationSuggestions(desc))
	}

	return nil
}

func computeResultKind(sourceKind metric.Kind, agg metric.AggregationName) metric.Kind {
	switch agg {
	case metric.AggMean:
		return metric.Measure
	case metric.AggCount:
		return metric.Quantity
	case metric.AggDistinct:
		return metric.Quantity
	case metric.AggMode:
		return metric.Classification
	default:
		return sourceKind
	}
}

func formatFilterNames(filters []metric.FilterName) string {
	strs := make([]string, len(filters))
	for i, f := range filters {
		strs[i] = string(f)
	}

	return strings.Join(strs, ", ")
}

func formatAggregationNames(aggs []metric.AggregationName) string {
	strs := make([]string, len(aggs))
	for i, a := range aggs {
		strs[i] = string(a)
	}

	return strings.Join(strs, ", ")
}

func formatAggregationSuggestions(desc BaseMetricDescriptor) string {
	strs := make([]string, len(desc.Aggregations))
	for i, a := range desc.Aggregations {
		strs[i] = string(desc.Name) + "." + string(a)
	}

	return strings.Join(strs, ", ")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/ -run "TestResolveExpression" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/resolution.go internal/provider/resolution_test.go
git commit -m "feat(provider): add expression resolution and validation

Resolves MetricExpression against the base registry, validates filters
and aggregations, computes result kind, and produces ResolvedMetric
with actionable error messages on invalid combinations."
```

---

### Task 8: Register filesystem metrics as base metrics

**Files:**
- Create: `internal/provider/filesystem/base_metrics.go`
- Create: `internal/provider/filesystem/base_metrics_test.go`

- [ ] **Step 1: Write test verifying filesystem base metrics are registered**

```go
// internal/provider/filesystem/base_metrics_test.go
package filesystem

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestRegisterBase_FilesystemMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	RegisterBase()

	fs, ok := provider.GetBase("file-size")
	g.Expect(ok).To(BeTrue())
	g.Expect(fs.Kind).To(Equal(metric.Quantity))
	g.Expect(fs.Level).To(Equal(metric.LevelFile))
	g.Expect(fs.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(fs.SupportsAggregation(metric.AggMean)).To(BeTrue())

	fl, ok := provider.GetBase("file-lines")
	g.Expect(ok).To(BeTrue())
	g.Expect(fl.Kind).To(Equal(metric.Quantity))
	g.Expect(fl.Level).To(Equal(metric.LevelFile))

	ft, ok := provider.GetBase("file-type")
	g.Expect(ok).To(BeTrue())
	g.Expect(ft.Kind).To(Equal(metric.Classification))
	g.Expect(ft.SupportsAggregation(metric.AggMode)).To(BeTrue())
	g.Expect(ft.SupportsAggregation(metric.AggDistinct)).To(BeTrue())
	g.Expect(ft.SupportsAggregation(metric.AggSum)).To(BeFalse())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/filesystem/ -run "TestRegisterBase" -v`
Expected: FAIL — `RegisterBase` does not exist.

- [ ] **Step 3: Implement filesystem base metric registration**

```go
// internal/provider/filesystem/base_metrics.go
package filesystem

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// FilesystemProvider is the provider descriptor for filesystem metrics.
var FilesystemProvider = provider.ProviderDescriptor{
	Name:    "filesystem",
	Filters: nil, // filesystem metrics have no filters
}

// RegisterBase adds filesystem base metric descriptors to the global base registry.
func RegisterBase() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileSize,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Size of each file in bytes.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, FilesystemProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileLines,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of lines in each text file.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, FilesystemProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileType,
		Kind:           metric.Classification,
		Level:          metric.LevelFile,
		Description:    "File extension category (e.g. go, md, png).",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggMode, metric.AggDistinct},
		DefaultPalette: palette.Categorization,
	}, FilesystemProvider)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/filesystem/ -run "TestRegisterBase" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/filesystem/base_metrics.go internal/provider/filesystem/base_metrics_test.go
git commit -m "feat(filesystem): register base metric descriptors

Declares file-size, file-lines, and file-type as base metrics with
their valid aggregations. No filters for filesystem metrics."
```

---

### Task 9: Register git metrics as base metrics

**Files:**
- Create: `internal/provider/git/base_metrics.go`
- Create: `internal/provider/git/base_metrics_test.go`

- [ ] **Step 1: Write test verifying git base metrics are registered**

```go
// internal/provider/git/base_metrics_test.go
package git

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestRegisterBase_GitMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	RegisterBase()

	cc, ok := provider.GetBase("commit-count")
	g.Expect(ok).To(BeTrue())
	g.Expect(cc.Kind).To(Equal(metric.Quantity))
	g.Expect(cc.Level).To(Equal(metric.LevelFile))
	g.Expect(cc.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMean)).To(BeTrue())

	cd, ok := provider.GetBase("commit-density")
	g.Expect(ok).To(BeTrue())
	g.Expect(cd.Kind).To(Equal(metric.Measure))
	g.Expect(cd.SupportsAggregation(metric.AggMin)).To(BeTrue())
	g.Expect(cd.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cd.SupportsAggregation(metric.AggSum)).To(BeFalse())
	g.Expect(cd.SupportsAggregation(metric.AggMean)).To(BeFalse())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/git/ -run "TestRegisterBase" -v`
Expected: FAIL — `RegisterBase` does not exist.

- [ ] **Step 3: Implement git base metric registration**

```go
// internal/provider/git/base_metrics.go
package git

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// GitProvider is the provider descriptor for git metrics.
var GitProvider = provider.ProviderDescriptor{
	Name:    "git",
	Filters: nil, // no filters for git metrics yet
}

// RegisterBase adds git base metric descriptors to the global base registry.
func RegisterBase() {
	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileAge,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Time since first commit (days); older files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           FileFreshness,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Time since most recent commit (days); recently changed files score higher.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Temperature,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           AuthorCount,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of distinct authors who have committed to the file.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           CommitCount,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Number of commits touching a file.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           TotalLinesAdded,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Total lines added across all commits.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           TotalLinesRemoved,
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Total lines removed across all commits.",
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GitProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           CommitDensity,
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Commits per day since file creation.",
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax},
		DefaultPalette: palette.Neutral,
	}, GitProvider)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/git/ -run "TestRegisterBase" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/git/base_metrics.go internal/provider/git/base_metrics_test.go
git commit -m "feat(git): register base metric descriptors

Declares 7 git metrics with their valid aggregations. Notably,
commit-density only supports min/max (sum and mean are meaningless)."
```

---

### Task 10: Register Go metrics as base metrics

**Files:**
- Create: `internal/provider/golang/base_metrics.go`
- Create: `internal/provider/golang/base_metrics_test.go`

- [ ] **Step 1: Write test verifying Go base metrics are registered**

```go
// internal/provider/golang/base_metrics_test.go
package golang

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

func TestRegisterBase_GoMetrics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	RegisterBase()

	types, ok := provider.GetBase("types")
	g.Expect(ok).To(BeTrue())
	g.Expect(types.Kind).To(Equal(metric.Quantity))
	g.Expect(types.Level).To(Equal(metric.LevelDeclaration))
	g.Expect(types.SupportsFilter("public")).To(BeTrue())
	g.Expect(types.SupportsFilter("private")).To(BeTrue())
	g.Expect(types.SupportsAggregation(metric.AggCount)).To(BeTrue())
	g.Expect(types.SupportsAggregation(metric.AggSum)).To(BeTrue())

	cc, ok := provider.GetBase("cyclomatic-complexity")
	g.Expect(ok).To(BeTrue())
	g.Expect(cc.Kind).To(Equal(metric.Quantity))
	g.Expect(cc.Level).To(Equal(metric.LevelDeclaration))
	g.Expect(cc.SupportsFilter("public")).To(BeFalse())
	g.Expect(cc.SupportsAggregation(metric.AggSum)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMax)).To(BeTrue())
	g.Expect(cc.SupportsAggregation(metric.AggMean)).To(BeTrue())

	cr, ok := provider.GetBase("comment-ratio")
	g.Expect(ok).To(BeTrue())
	g.Expect(cr.Kind).To(Equal(metric.Measure))
	g.Expect(cr.Level).To(Equal(metric.LevelFile))

	imports, ok := provider.GetBase("imports")
	g.Expect(ok).To(BeTrue())
	g.Expect(imports.Level).To(Equal(metric.LevelFile))
	g.Expect(imports.SupportsFilter("stdlib")).To(BeTrue())
	g.Expect(imports.SupportsFilter("external")).To(BeTrue())
	g.Expect(imports.SupportsFilter("internal")).To(BeTrue())
}

func TestRegisterBase_GoProvider_HasFilters(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	provider.ResetBaseRegistryForTesting()
	RegisterBase()

	pd, ok := provider.GetBaseProvider("types")
	g.Expect(ok).To(BeTrue())
	g.Expect(pd.Name).To(Equal("go"))
	g.Expect(pd.HasFilter("public")).To(BeTrue())
	g.Expect(pd.HasFilter("private")).To(BeTrue())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/provider/golang/ -run "TestRegisterBase" -v`
Expected: FAIL — `RegisterBase` does not exist in golang package.

- [ ] **Step 3: Implement Go base metric registration**

```go
// internal/provider/golang/base_metrics.go
package golang

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// GoProvider is the provider descriptor for Go metrics.
var GoProvider = provider.ProviderDescriptor{
	Name: "go",
	Filters: map[metric.FilterName]string{
		"public":   "Exported declarations only",
		"private":  "Unexported declarations only",
		"stdlib":   "Standard library imports only",
		"external": "External (third-party) imports only",
		"internal": "Internal (same module) imports only",
	},
}

// RegisterBase adds Go base metric descriptors to the global base registry.
func RegisterBase() {
	declCountAggs := []metric.AggregationName{metric.AggCount, metric.AggSum}
	visibilityFilters := []metric.FilterName{"public", "private"}

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "types",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of type declarations.",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "interfaces",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of interface type declarations.",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "structs",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of struct type declarations.",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "functions",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of function declarations (no receiver).",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "methods",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of method declarations (with receiver).",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "constants",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of constant declarations.",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "variables",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Count of variable declarations.",
		Filters:        visibilityFilters,
		Aggregations:   declCountAggs,
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "imports",
		Kind:           metric.Quantity,
		Level:          metric.LevelFile,
		Description:    "Total import paths in Go files.",
		Filters:        []metric.FilterName{"stdlib", "external", "internal"},
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "cyclomatic-complexity",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Cyclomatic complexity per function.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "function-length",
		Kind:           metric.Quantity,
		Level:          metric.LevelDeclaration,
		Description:    "Function length in lines.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggSum, metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GoProvider)

	provider.RegisterBaseWithProvider(provider.BaseMetricDescriptor{
		Name:           "comment-ratio",
		Kind:           metric.Measure,
		Level:          metric.LevelFile,
		Description:    "Ratio of comment lines to code lines in Go files.",
		Filters:        nil,
		Aggregations:   []metric.AggregationName{metric.AggMin, metric.AggMax, metric.AggMean},
		DefaultPalette: palette.Neutral,
	}, GoProvider)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/provider/golang/ -run "TestRegisterBase" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/provider/golang/base_metrics.go internal/provider/golang/base_metrics_test.go
git commit -m "feat(golang): register base metric descriptors

11 base metrics replace 34 flat providers. Declaration-level metrics
support public/private filters. Import metrics support stdlib/external/
internal filters."
```

---

### Task 11: Wire base metric registration into application startup

**Files:**
- Modify: `cmd/codeviz/main.go` (add `RegisterBase()` calls alongside existing `Register()` calls)

- [ ] **Step 1: Find where providers are registered at startup**

Look at `cmd/codeviz/main.go` for `init()` or similar registration code that calls `filesystem.Register()`, `git.Register()`, and `golang.Register()`.

- [ ] **Step 2: Add base metric registration calls**

Add calls to `filesystem.RegisterBase()`, `git.RegisterBase()`, and `golang.RegisterBase()` alongside the existing `Register()` calls. Both registries coexist during migration.

```go
// Add alongside existing Register() calls:
filesystem.RegisterBase()
git.RegisterBase()
golang.RegisterBase()
```

- [ ] **Step 3: Run full test suite**

Run: `task test`
Expected: PASS — both registries populated, no conflicts.

- [ ] **Step 4: Commit**

```bash
git add cmd/codeviz/main.go
git commit -m "feat: wire base metric registration into startup

Both the legacy Interface registry and the new base metric registry
are populated at startup. They coexist during the migration period."
```

---

### Task 12: Update MetricSpec to parse expressions

**Files:**
- Modify: `internal/config/metric_spec.go`
- Modify: `internal/config/metric_spec_test.go` (or the existing test file for MetricSpec)

- [ ] **Step 1: Write tests for expression-aware MetricSpec validation**

Add tests that verify `MetricSpec.Validate` accepts expression syntax and rejects invalid combinations. The validation should try the new base registry first, falling back to the legacy registry for backward compatibility during migration.

```go
func TestMetricSpec_Validate_ExpressionSyntax(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Assuming base metrics are registered (via TestMain or init)
	spec := &MetricSpec{Metric: "file-size.sum"}
	err := spec.Validate("size")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestMetricSpec_Validate_InvalidAggregation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	spec := &MetricSpec{Metric: "file-size.mode"}
	err := spec.Validate("fill")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not a valid aggregation"))
}

func TestMetricSpec_Validate_BareMetricStillWorks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	spec := &MetricSpec{Metric: "file-size"}
	err := spec.Validate("size")
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run "TestMetricSpec_Validate_Expression" -v`
Expected: FAIL — current validation uses legacy registry only.

- [ ] **Step 3: Update MetricSpec.Validate to use expression parsing**

Modify `Validate` in `internal/config/metric_spec.go` to:
1. Try parsing as a `MetricExpression`
2. If parsing succeeds, resolve against base registry
3. If base registry resolves, accept
4. If base registry fails (metric not found), fall back to legacy `provider.FindWithHint`

```go
func (m *MetricSpec) Validate(label string) error {
	if m == nil {
		return nil
	}

	if m.Metric != "" {
		if err := m.validateMetric(label); err != nil {
			return err
		}
	}

	if m.Palette != "" {
		if !m.Palette.IsValid() {
			return eris.Errorf("invalid %s palette %q", label, m.Palette)
		}
	}

	return nil
}

func (m *MetricSpec) validateMetric(label string) error {
	expr, parseErr := metric.ParseExpression(string(m.Metric))
	if parseErr == nil {
		_, resolveErr := provider.ResolveExpression(expr, metric.LevelFile)
		if resolveErr == nil {
			return nil
		}

		// If the base metric is simply unknown in new registry, fall back to legacy
		if _, legacyOk := provider.Get(m.Metric, metric.File); legacyOk {
			return nil
		}

		return eris.Wrapf(resolveErr, "invalid %s metric", label)
	}

	// Parse failed — try legacy lookup
	if _, err := provider.FindWithHint(m.Metric, metric.File); err != nil {
		return eris.Wrapf(err, "invalid %s metric", label)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run "TestMetricSpec" -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `task test`
Expected: PASS — existing tests still work via legacy fallback.

- [ ] **Step 6: Commit**

```bash
git add internal/config/metric_spec.go internal/config/metric_spec_test.go
git commit -m "feat(config): expression-aware MetricSpec validation

Validates metric expressions against the base registry first, falling
back to the legacy provider registry for backward compatibility."
```

---

### Task 13: Update help metrics command

**Files:**
- Modify: `cmd/codeviz/help_metrics_cmd.go`
- Modify: `cmd/codeviz/help_metrics_cmd_test.go`

- [ ] **Step 1: Write test for new help metrics output format**

```go
func TestHelpMetricsCmdRun_ShowsBaseMetricsWithAggregations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	output := captureStdout(t, func() {
		err := (HelpMetricsCmd{}).Run(&Flags{})
		g.Expect(err).NotTo(HaveOccurred())
	})

	// Verify syntax reference
	g.Expect(output).To(ContainSubstring("Syntax:"))
	g.Expect(output).To(ContainSubstring("[filter.]metric[.aggregation]"))

	// Verify sections exist
	g.Expect(output).To(ContainSubstring("Filesystem"))
	g.Expect(output).To(ContainSubstring("Git"))
	g.Expect(output).To(ContainSubstring("Go"))

	// Verify aggregations are shown
	g.Expect(output).To(ContainSubstring("sum"))
	g.Expect(output).To(ContainSubstring("min"))
	g.Expect(output).To(ContainSubstring("max"))

	// Verify filter info
	g.Expect(output).To(ContainSubstring("public"))
	g.Expect(output).To(ContainSubstring("private"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/codeviz/ -run "TestHelpMetricsCmdRun_ShowsBaseMetrics" -v`
Expected: FAIL — current output doesn't include aggregation info.

- [ ] **Step 3: Rewrite help metrics command**

Replace the body of `help_metrics_cmd.go` with code that reads from the base metric registry and formats using the grouped display described in the spec. Show syntax reference, provider sections with filter vocabulary headers, and per-metric aggregation/filter lists.

- [ ] **Step 4: Update existing test**

Update `TestHelpMetricsCmdRun_GroupsMetricsByProvider` to match the new output format (base metric names like `types` instead of `type-count`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/codeviz/ -run "TestHelpMetricsCmd" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/codeviz/help_metrics_cmd.go cmd/codeviz/help_metrics_cmd_test.go
git commit -m "feat(help): rewrite metrics listing with expression syntax

Shows base metrics grouped by provider, with per-metric aggregation
and filter information. Includes syntax reference header."
```

---

### Task 14: Run full CI and fix any issues

**Files:**
- Potentially modify any files with lint or test issues

- [ ] **Step 1: Run full CI**

Run: `task ci`
Expected: Build succeeds, all tests pass, lint clean.

- [ ] **Step 2: Fix any issues found**

Address any failing tests, lint warnings, or build errors.

- [ ] **Step 3: Commit fixes if needed**

```bash
git add -A
git commit -m "fix: address CI issues from metric expressions implementation"
```

---

### Task 15: Push branch and create PR

- [ ] **Step 1: Push the feature branch**

```bash
git push -u origin feature/metric-expressions-design
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "feat: add metric expressions system" \
  --body "Implements the composable [filter.]base-metric[.aggregation] syntax per the design spec.

## Summary
- New types: FilterName, AggregationName, MetricLevel, MetricExpression
- Expression parser with validation
- Generic aggregation functions (sum, min, max, mean, count, mode, distinct, range)
- Base metric registry coexisting with legacy registry
- Expression-aware MetricSpec validation with legacy fallback
- Filesystem, git, and Go providers register base metrics
- Updated help metrics display showing aggregations and filters

## What's NOT in this PR (future work)
- Actual aggregation computation stage (requires model changes for declarations/commits)
- Migration of existing flat providers to the new system (clean removal)
- Directory-level metric computation pipeline"
```
