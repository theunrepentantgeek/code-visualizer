package model_test

import (
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// newContainer returns a fresh zero-value File, whose embedded MetricContainer
// starts with all maps nil (lazy-initialised).
func newContainer() *model.File {
	return &model.File{Name: "f.go", Path: "f.go"}
}

// ---------------------------------------------------------------------------
// Quantity
// ---------------------------------------------------------------------------

func TestMetricContainer_Quantity_NilMap_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	v, ok := f.Quantity(metric.Name("lines"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeZero())
}

func TestMetricContainer_Quantity_UnknownName_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetQuantity(metric.Name("lines"), 10)

	v, ok := f.Quantity(metric.Name("size"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeZero())
}

func TestMetricContainer_SetQuantity_ThenGet_ReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetQuantity(metric.Name("lines"), 42)

	v, ok := f.Quantity(metric.Name("lines"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(42)))
}

func TestMetricContainer_SetQuantity_OverwritesPreviousValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetQuantity(metric.Name("lines"), 10)
	f.SetQuantity(metric.Name("lines"), 99)

	v, ok := f.Quantity(metric.Name("lines"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(int64(99)))
}

func TestMetricContainer_SetQuantity_ZeroValue_Stored(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetQuantity(metric.Name("lines"), 0)

	v, ok := f.Quantity(metric.Name("lines"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeZero())
}

// ---------------------------------------------------------------------------
// Measure
// ---------------------------------------------------------------------------

func TestMetricContainer_Measure_NilMap_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	v, ok := f.Measure(metric.Name("coverage"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeZero())
}

func TestMetricContainer_Measure_UnknownName_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetMeasure(metric.Name("coverage"), 0.75)

	v, ok := f.Measure(metric.Name("complexity"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeZero())
}

func TestMetricContainer_SetMeasure_ThenGet_ReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetMeasure(metric.Name("coverage"), 0.95)

	v, ok := f.Measure(metric.Name("coverage"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 0.95, 1e-9))
}

func TestMetricContainer_SetMeasure_OverwritesPreviousValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetMeasure(metric.Name("coverage"), 0.5)
	f.SetMeasure(metric.Name("coverage"), 0.9)

	v, ok := f.Measure(metric.Name("coverage"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(BeNumerically("~", 0.9, 1e-9))
}

// ---------------------------------------------------------------------------
// Classification
// ---------------------------------------------------------------------------

func TestMetricContainer_Classification_NilMap_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	v, ok := f.Classification(metric.Name("type"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeEmpty())
}

func TestMetricContainer_Classification_UnknownName_ReturnsFalse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetClassification(metric.Name("type"), "go")

	v, ok := f.Classification(metric.Name("category"))
	g.Expect(ok).To(BeFalse())
	g.Expect(v).To(BeEmpty())
}

func TestMetricContainer_SetClassification_ThenGet_ReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetClassification(metric.Name("type"), "go")

	v, ok := f.Classification(metric.Name("type"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal("go"))
}

func TestMetricContainer_SetClassification_OverwritesPreviousValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()
	f.SetClassification(metric.Name("type"), "go")
	f.SetClassification(metric.Name("type"), "ts")

	v, ok := f.Classification(metric.Name("type"))
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal("ts"))
}

// ---------------------------------------------------------------------------
// Independence of metric types
// ---------------------------------------------------------------------------

func TestMetricContainer_DifferentKindsDoNotInterfere(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	f := newContainer()

	const name = metric.Name("x")

	f.SetQuantity(name, 7)
	f.SetMeasure(name, 3.14)
	f.SetClassification(name, "alpha")

	q, okQ := f.Quantity(name)
	m, okM := f.Measure(name)
	c, okC := f.Classification(name)

	g.Expect(okQ).To(BeTrue())
	g.Expect(q).To(Equal(int64(7)))
	g.Expect(okM).To(BeTrue())
	g.Expect(m).To(BeNumerically("~", 3.14, 1e-9))
	g.Expect(okC).To(BeTrue())
	g.Expect(c).To(Equal("alpha"))
}

// ---------------------------------------------------------------------------
// Concurrent access (race detector)
// ---------------------------------------------------------------------------

func TestMetricContainer_ConcurrentReadWrite_NoRace(t *testing.T) {
	t.Parallel()

	f := newContainer()

	const name = metric.Name("lines")

	var wg sync.WaitGroup

	for range 20 {
		wg.Go(func() {
			f.SetQuantity(name, 1)
			_, _ = f.Quantity(name)
		})
	}

	wg.Wait()
}
