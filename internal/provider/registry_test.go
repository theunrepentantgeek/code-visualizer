package provider

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// stubProvider is a minimal Interface implementation for testing.
type stubProvider struct {
	name   metric.Name
	kind   metric.Kind
	target metric.Target
}

func (s *stubProvider) Name() metric.Name                 { return s.name }
func (s *stubProvider) Kind() metric.Kind                 { return s.kind }
func (s *stubProvider) Target() metric.Target             { return s.target }
func (*stubProvider) Description() string                 { return "" }
func (*stubProvider) Dependencies() []metric.Name         { return nil }
func (*stubProvider) DefaultPalette() palette.PaletteName { return palette.Neutral }
func (*stubProvider) Load(_ *model.Directory) error       { return nil }

func TestRegisterAndGet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	p := &stubProvider{name: "test-metric", kind: metric.Quantity, target: metric.File}
	reg.register(p)

	got, ok := reg.get("test-metric")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).ToNot(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Name()).To(Equal(metric.Name("test-metric")))
	g.Expect(got.Kind()).To(Equal(metric.Quantity))
	g.Expect(got.Target()).To(Equal(metric.File))
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
	reg.register(&stubProvider{name: "m1", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "m2", kind: metric.Classification, target: metric.File})

	all := reg.all()
	g.Expect(all).To(HaveLen(2))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})
	}).To(Panic())
}

func TestNamesSorted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zebra", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "mid", kind: metric.Quantity, target: metric.File})

	names := reg.names()
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}

func TestDescriptorIncludesTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	desc := Descriptor(&stubProvider{name: "test-metric", kind: metric.Quantity, target: metric.Directory})

	g.Expect(desc.Target).To(Equal(metric.Directory))
}
