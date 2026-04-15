package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/palette"
)

// stubProvider is a minimal Interface implementation for testing.
type stubProvider struct {
	name metric.Name
	kind metric.Kind
}

func (s *stubProvider) Name() metric.Name                 { return s.name }
func (s *stubProvider) Kind() metric.Kind                 { return s.kind }
func (*stubProvider) Dependencies() []metric.Name         { return nil }
func (*stubProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (*stubProvider) Load(_ *model.Directory) error       { return nil }

func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	p := &stubProvider{name: "test-metric", kind: metric.Quantity}
	reg.register(p)

	got, ok := reg.get("test-metric")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).ToNot(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Name()).To(Equal(metric.Name("test-metric")))
	g.Expect(got.Kind()).To(Equal(metric.Quantity))
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
	reg.register(&stubProvider{name: "m1", kind: metric.Quantity})
	reg.register(&stubProvider{name: "m2", kind: metric.Classification})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: metric.Quantity})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: metric.Quantity})
	}).To(Panic())
}

func TestNamesSorted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zebra", kind: metric.Quantity})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity})
	reg.register(&stubProvider{name: "mid", kind: metric.Quantity})

	names := reg.names()
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}
