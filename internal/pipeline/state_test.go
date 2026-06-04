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

	v, ok := Lookup[Kind](state)
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

	_, ok := Lookup[Color](state)
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

	v, ok := Lookup[Kind](state)

	g.Expect(ok).To(BeTrue())
	g.Expect(v).To(Equal(beta))
}
