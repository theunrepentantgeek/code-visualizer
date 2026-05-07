package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// stubLoader is a minimal Loader implementation for testing.
type stubLoader struct{}

func (*stubLoader) Load(_ *model.Directory) error { return nil }

func testDesc(name metric.Name, kind metric.Kind) MetricDescriptor {
	return MetricDescriptor{
		Name:           name,
		Kind:           kind,
		DefaultPalette: palette.Neutral,
	}
}

func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	desc := testDesc("test-metric", metric.Quantity)
	reg.register(desc, &stubLoader{})

	got, ok := reg.get("test-metric")
	g.Expect(ok).To(BeTrue())
	g.Expect(got.Name).To(Equal(metric.Name("test-metric")))
	g.Expect(got.Kind).To(Equal(metric.Quantity))
}

func TestGetUnregistered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	_, ok := reg.get("nonexistent")
	g.Expect(ok).To(BeFalse())
}

func TestAllProviders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(testDesc("m1", metric.Quantity), &stubLoader{})
	reg.register(testDesc("m2", metric.Classification), &stubLoader{})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(testDesc("dup", metric.Quantity), &stubLoader{})

	g.Expect(func() {
		reg.register(testDesc("dup", metric.Quantity), &stubLoader{})
	}).To(Panic())
}

func TestNamesSorted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(testDesc("zebra", metric.Quantity), &stubLoader{})
	reg.register(testDesc("alpha", metric.Quantity), &stubLoader{})
	reg.register(testDesc("mid", metric.Quantity), &stubLoader{})

	names := reg.names()
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}
