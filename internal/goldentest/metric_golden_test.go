package goldentest

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sebdah/goldie/v2"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// candidateExpressions builds every expression worth probing for each base
// metric: the bare metric, base×aggregation, filter×base, and
// filter×base×aggregation. Mirrors cmd/codeviz/render_matrix_test.go so the set
// tracks the registry automatically.
func candidateExpressions() []string {
	names := make([]string, 0)
	for _, desc := range provider.AllBase() {
		base := string(desc.Name)
		names = append(names, base)
		for _, agg := range desc.Aggregations {
			names = append(names, base+"."+string(agg))
		}
		for _, fn := range desc.Filters {
			filtered := string(fn) + "." + base
			names = append(names, filtered)
			for _, agg := range desc.Aggregations {
				names = append(names, filtered+"."+string(agg))
			}
		}
	}

	return names
}

// validExpressions resolves each candidate at directory level and keeps the
// ones the registry accepts, de-duplicated and deterministic.
func validExpressions(t *testing.T) []provider.ResolvedMetric {
	t.Helper()

	seen := make(map[string]bool)
	resolved := make([]provider.ResolvedMetric, 0)
	for _, name := range candidateExpressions() {
		if seen[name] {
			continue
		}
		seen[name] = true

		r, err := provider.ResolveForValidation(metric.Name(name))
		if err != nil {
			continue
		}
		resolved = append(resolved, r)
	}

	return resolved
}

// requestedNames returns every metric name to include in the JSON snapshot:
// the file-level base names (so file rows show their inputs) plus every
// resolved expression's ResultName (so directory aggregates appear).
func requestedNames(resolved []provider.ResolvedMetric) []metric.Name {
	seen := make(map[metric.Name]bool)
	names := make([]metric.Name, 0)
	add := func(n metric.Name) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, desc := range provider.AllBase() {
		add(desc.Name)
	}
	for _, r := range resolved {
		add(r.ResultName)
	}

	return names
}

// roundMeasures rounds every Measure value in the tree to 6 decimal places, to
// keep the JSON snapshot robust to last-bit floating-point differences from
// aggregation-order changes. Quantities and classifications are exact already.
func roundMeasures(root *model.Directory, names []metric.Name) {
	round := func(mc interface {
		Measure(metric.Name) (float64, bool)
		SetMeasure(metric.Name, float64)
	},
	) {
		for _, n := range names {
			if v, ok := mc.Measure(n); ok {
				mc.SetMeasure(n, math.Round(v*1e6)/1e6)
			}
		}
	}

	var walkDir func(d *model.Directory)
	walkDir = func(d *model.Directory) {
		round(&d.MetricContainer)
		for _, f := range d.Files {
			round(&f.MetricContainer)
		}
		for _, sub := range d.Dirs {
			walkDir(sub)
		}
	}
	walkDir(root)
}

func TestGolden_MetricExpressions(t *testing.T) {
	g := NewGomegaWithT(t)

	root := buildMetricTree()
	resolved := validExpressions(t)
	g.Expect(resolved).NotTo(BeEmpty(), "registry should yield valid expressions")

	// ComputeAggregations only processes metrics that roll up from a finer
	// level; bare file-level base values already live on the files (and still
	// appear in the JSON export). This mirrors the production RunAggregations
	// input, whose expression list is likewise aggregation-only.
	aggregatable := make([]provider.ResolvedMetric, 0, len(resolved))
	for _, r := range resolved {
		if r.NeedsAggregation {
			aggregatable = append(aggregatable, r)
		}
	}
	g.Expect(aggregatable).NotTo(BeEmpty(), "registry should yield aggregatable expressions")

	g.Expect(stages.ComputeAggregations(root, aggregatable)).To(Succeed())

	names := requestedNames(resolved)
	roundMeasures(root, names)

	out := filepath.Join(t.TempDir(), "metrics.json")
	g.Expect(export.Export(root, names, out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())

	gold := goldie.New(t)
	gold.Assert(t, "metric-expressions", data)
}
