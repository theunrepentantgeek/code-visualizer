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

	got, ok := reg.get("test-metric", metric.File)
	g.Expect(ok).To(BeTrue())
	g.Expect(got).ToNot(BeNil())

	if got == nil {
		return
	}

	g.Expect(got.Name()).To(Equal(metric.Name("test-metric")))
}

func TestGetWithWrongTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "test-metric", kind: metric.Quantity, target: metric.File})

	_, ok := reg.get("test-metric", metric.Directory)
	g.Expect(ok).To(BeFalse())
}

func TestGetUnregistered(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	_, ok := reg.get("nonexistent", metric.File)
	g.Expect(ok).To(BeFalse())
}

func TestSameNameDifferentTargets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	fileP := &stubProvider{name: "size", kind: metric.Quantity, target: metric.File}
	dirP := &stubProvider{name: "size", kind: metric.Quantity, target: metric.Directory}

	reg.register(fileP)
	reg.register(dirP)

	gotFile, ok := reg.get("size", metric.File)
	g.Expect(ok).To(BeTrue())
	g.Expect(gotFile).ToNot(BeNil())

	if gotFile == nil {
		return
	}

	g.Expect(gotFile.Target()).To(Equal(metric.File))

	gotDir, ok := reg.get("size", metric.Directory)
	g.Expect(ok).To(BeTrue())
	g.Expect(gotDir).ToNot(BeNil())

	if gotDir == nil {
		return
	}

	g.Expect(gotDir.Target()).To(Equal(metric.Directory))
}

func TestAllProviders(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "m1", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "m2", kind: metric.Classification, target: metric.File})
	reg.register(&stubProvider{name: "m3", kind: metric.Quantity, target: metric.Directory})

	all := reg.allFor(metric.File)
	g.Expect(all).To(HaveLen(2))

	allDir := reg.allFor(metric.Directory)
	g.Expect(allDir).To(HaveLen(1))
}

func TestAllProvidersAcrossTargets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zeta", kind: metric.Quantity, target: metric.Directory})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "mid", kind: metric.Classification, target: metric.Directory})

	all := reg.all()
	g.Expect(all).To(HaveLen(3))

	names := make([]metric.Name, len(all))
	for i, p := range all {
		names[i] = p.Name()
	}

	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zeta"}))
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

func TestDuplicateNameDifferentTargetDoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.File})

	g.Expect(func() {
		reg.register(&stubProvider{name: "dup", kind: metric.Quantity, target: metric.Directory})
	}).ToNot(Panic())
}

func TestNamesSorted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zebra", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "mid", kind: metric.Quantity, target: metric.File})

	names := reg.namesFor(metric.File)
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}

func TestNamesDeduplicateAcrossTargets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "zebra", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "alpha", kind: metric.Quantity, target: metric.Directory})
	reg.register(&stubProvider{name: "mid", kind: metric.Quantity, target: metric.Directory})

	names := reg.names()
	g.Expect(names).To(Equal([]metric.Name{"alpha", "mid", "zebra"}))
}

func TestHasName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "exists", kind: metric.Quantity, target: metric.File})

	g.Expect(reg.hasName("exists")).To(BeTrue())
	g.Expect(reg.hasName("missing")).To(BeFalse())
}

func TestTargetsForName(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	reg := newRegistry()
	reg.register(&stubProvider{name: "size", kind: metric.Quantity, target: metric.File})
	reg.register(&stubProvider{name: "size", kind: metric.Quantity, target: metric.Directory})

	targets := reg.targetsForName("size")
	g.Expect(targets).To(ConsistOf(metric.File, metric.Directory))

	targets = reg.targetsForName("missing")
	g.Expect(targets).To(BeEmpty())
}
