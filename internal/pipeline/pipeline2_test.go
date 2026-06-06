package pipeline

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
)

/*
 * ApplyFuncXR Tests
 */

func Test_ApplyFuncXR_WhenStateDoesNotContainX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var c Color
	state := NewState(c)

	g.Expect(func() {
		ApplyFuncXR(state, SetKind("k"))
	}).To(PanicWith(ContainSubstring("Kind")))
}

func Test_ApplyFuncXR_WhenStateContainsX_CallsMethod(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind
	state := NewState(k)
	ApplyFuncXR(state, SetKind("k"))
	g.Expect(state.Err()).ToNot(HaveOccurred())

	var name *string
	ApplyFuncXR(state, ExtractKind(&name))
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(name).ToNot(BeNil())
	g.Expect(*name).To(Equal("k"))
}

func Test_ApplyFuncXR_WhenMethodReturnsValue_SavesValueInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind
	state := NewState(k)
	ApplyFuncXR(state, SetKind("k"))
	g.Expect(state.Err()).ToNot(HaveOccurred())

	v, ok := Lookup[Kind](state)
	g.Expect(ok).To(BeTrue())
	g.Expect(v.name).To(Equal("k"))
}

func Test_ApplyFuncXR_WhenMethodReturnsError_SetsErrorInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind
	state := NewState(k)
	ApplyFuncXR(state, func(Kind) (Kind, error) {
		return Kind{}, fmt.Errorf("error")
	})
	g.Expect(state.Err()).To(MatchError(ContainSubstring("error")))
}

/*
 * ApplyFuncXYR Tests
 */

func Test_ApplyFuncXYR_WhenStateDoesNotContainX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	k := Kind{
		name: "stripes",
	}

	state := NewState(k)

	g.Expect(func() {
		ApplyFuncXYR(state, CreateTexture)
	}).To(PanicWith(ContainSubstring("Color")))
}

func Test_ApplyFuncXYR_WhenStateDoesNotContainY_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := Color{
		name: "red",
	}

	state := NewState(c)

	g.Expect(func() {
		ApplyFuncXYR(state, CreateTexture)
	}).To(PanicWith(ContainSubstring("Kind")))
}

func Test_ApplyFuncXYR_WhenStateContainsXAndY_StoresResultInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := Color{
		name: "red",
	}

	k := Kind{
		name: "stripes",
	}

	state := NewState(c)
	store(state, k)

	ApplyFuncXYR(state, CreateTexture)
	g.Expect(state.Err()).ToNot(HaveOccurred())

	var name *string
	ApplyFuncXR(state, ExtractTexture(&name))
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(name).ToNot(BeNil())
	g.Expect(*name).To(Equal("red-stripes"))
}
