package pipeline

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestNewState_GivenValue_ReturnsValueViaLookup(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	k := Kind{
		name: "test",
	}

	state := NewState(k)

	v, ok := lookup[Kind](state)
	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(k))
}

func TestState_Lookup_WhenValueNotPresent_ReturnsZeroValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	alpha := Kind{
		name: "alpha",
	}

	state := NewState(alpha)

	_, ok := lookup[Color](state)
	g.Expect(ok).To(BeFalse())
}

func TestState_Store_WhenValuePresent_OverwritesValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	alpha := Kind{
		name: "alpha",
	}

	state := NewState(alpha)

	beta := Kind{
		name: "beta",
	}

	store(state, beta)

	v, ok := lookup[Kind](state)

	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(beta))
}

func TestNewState_GivenMultipleValues_StoresAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	k := Kind{name: "k"}
	c := Color{name: "c"}

	state := NewState(k, c)

	kv, kok := lookup[Kind](state)
	cv, cok := lookup[Color](state)

	g.Expect(kok).To(BeTrue())
	g.Expect(cok).To(BeTrue())
	g.Expect(kv).To(Equal(k))
	g.Expect(cv).To(Equal(c))
}

func TestNewState_GivenNilValue_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(func() { NewState(nil) }).To(PanicWith(ContainSubstring("nil value")))
}

func TestNewState_GivenDuplicateType_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := Kind{name: "a"}
	b := Kind{name: "b"}

	g.Expect(func() { NewState(a, b) }).To(PanicWith(ContainSubstring("duplicate value for type")))
}

func Test_store_StoresValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState()
	store(state, Kind{name: "x"})

	v, ok := lookup[Kind](state)
	g.Expect(ok).To(BeTrue())
	g.Expect(v.name).To(Equal("x"))
}
