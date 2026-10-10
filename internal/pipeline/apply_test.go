package pipeline

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

type testSink interface {
	Value() string
}

type sinkValue string

func (s sinkValue) Value() string { return string(s) }

func TestSet_StoresAndReplacesInterfaceByStaticType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{})

	state.Set[testSink](sinkValue("first"))
	stored, ok := state.lookup[testSink]()
	g.Expect(ok).To(BeTrue())
	g.Expect(stored.Value()).To(Equal("first"))

	state.Set[testSink](sinkValue("second"))
	stored, ok = state.lookup[testSink]()
	g.Expect(ok).To(BeTrue())
	g.Expect(stored.Value()).To(Equal("second"))
}

func TestSet_PreservesExistingError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{})
	prior := errors.New("prior")
	state.setErr(prior)

	state.Set[testSink](sinkValue("value"))

	g.Expect(state.Err()).To(MatchError(prior))
}

/*
 * ApplyFuncX Tests
 */

func Test_ApplyFuncX_WhenStateDoesNotContainX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var c Color

	state := NewState(c)

	g.Expect(func() {
		state.ApplyFuncX(func(Kind) error { return nil })
	}).To(PanicWith(ContainSubstring("Kind")))
}

func Test_ApplyFuncX_WhenStateContainsX_CallsMethod(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)
	called := false

	state.ApplyFuncX(func(Kind) error {
		called = true

		return nil
	})
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func Test_ApplyFuncX_WhenMethodReturnsError_SetsErrorInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)
	state.ApplyFuncX(func(Kind) error {
		return errors.New("error")
	})
	g.Expect(state.Err()).To(MatchError(ContainSubstring("error")))
}

/*
 * ApplyFuncXR Tests
 */

func Test_ApplyFuncXR_WhenStateDoesNotContainX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var c Color

	state := NewState(c)

	g.Expect(func() {
		state.ApplyFuncXR(SetKind("k"))
	}).To(PanicWith(ContainSubstring("Kind")))
}

func Test_ApplyFuncXR_WhenStateContainsX_CallsMethod(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)
	state.ApplyFuncXR(SetKind("k"))
	g.Expect(state.Err()).ToNot(HaveOccurred())

	var name *string
	state.ApplyFuncXR(ExtractKind(&name))
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(name).ToNot(BeNil())

	if name != nil {
		g.Expect(*name).To(Equal("k"))
	}
}

func Test_ApplyFuncXR_WhenMethodReturnsValue_SavesValueInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)
	state.ApplyFuncXR(SetKind("k"))
	g.Expect(state.Err()).ToNot(HaveOccurred())

	v, ok := state.lookup[Kind]()
	g.Expect(ok).To(BeTrue())
	g.Expect(v.name).To(Equal("k"))
}

func Test_ApplyFuncXR_WhenMethodReturnsError_SetsErrorInState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)
	state.ApplyFuncXR(func(Kind) (Kind, error) {
		return Kind{}, errors.New("error")
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
		state.ApplyFuncXYR(CreateTexture)
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
		state.ApplyFuncXYR(CreateTexture)
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
	state.store(k)

	state.ApplyFuncXYR(CreateTexture)
	g.Expect(state.Err()).ToNot(HaveOccurred())

	var name *string
	state.ApplyFuncXR(ExtractTexture(&name))
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(name).ToNot(BeNil())

	if name != nil {
		g.Expect(*name).To(Equal("red-stripes"))
	}
}

/*
 * ApplyFuncXY Tests (in-place mutation, no return value)
 */

func Test_ApplyFuncXY_WhenStateMissingX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind

	state := NewState(k)

	g.Expect(func() {
		state.ApplyFuncXY(func(Color, Kind) error { return nil })
	}).To(PanicWith(ContainSubstring("Color")))
}

func Test_ApplyFuncXY_WhenBothPresent_CallsFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{name: "k"}, Color{name: "c"})
	called := false

	state.ApplyFuncXY(func(Kind, Color) error {
		called = true

		return nil
	})
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func Test_ApplyFuncXY_WhenFuncReturnsError_SetsStateErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})
	state.ApplyFuncXY(func(Kind, Color) error { return errors.New("boom") })
	g.Expect(state.Err()).To(MatchError(ContainSubstring("boom")))
}

func Test_ApplyFuncXY_WhenAlreadyErrored_ShortCircuits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})
	state.setErr(errors.New("prior"))

	called := false

	state.ApplyFuncXY(func(Kind, Color) error {
		called = true

		return nil
	})
	g.Expect(called).To(BeFalse())
}

/*
 * ApplyFuncXYZ Tests
 */

func Test_ApplyFuncXYZ_WhenStateMissingZ_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})

	g.Expect(func() {
		state.ApplyFuncXYZ(func(Kind, Color, Texture) error { return nil })
	}).To(PanicWith(ContainSubstring("Texture")))
}

func Test_ApplyFuncXYZ_WhenAllPresent_CallsFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{name: "k"}, Color{name: "c"}, Texture{name: "t"})
	called := false

	state.ApplyFuncXYZ(func(Kind, Color, Texture) error {
		called = true

		return nil
	})
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func Test_ApplyFuncXYZ_WhenFuncReturnsError_SetsStateErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{}, Texture{})
	state.ApplyFuncXYZ(func(Kind, Color, Texture) error { return errors.New("boom") })
	g.Expect(state.Err()).To(MatchError(ContainSubstring("boom")))
}

func Test_ApplyFuncs_ReturnSameStateForChaining(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{}, Texture{})
	callCount := 0

	returned := state.
		ApplyFuncX(func(Kind) error {
			callCount++

			return nil
		}).
		ApplyFuncXR(func(kind Kind) (Kind, error) {
			callCount++

			return kind, nil
		}).
		ApplyFuncXYR(func(Color, Kind) (Texture, error) {
			callCount++

			return Texture{}, nil
		}).
		ApplyFuncXY(func(Kind, Color) error {
			callCount++

			return nil
		}).
		ApplyFuncXYZ(func(Kind, Color, Texture) error {
			callCount++

			return nil
		})

	g.Expect(returned).To(BeIdenticalTo(state))
	g.Expect(callCount).To(Equal(5))
	g.Expect(state.Err()).ToNot(HaveOccurred())
}

func Test_ApplyFuncs_ChainingShortCircuitsAfterError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{})
	called := false

	returned := state.
		ApplyFuncX(func(Kind) error {
			return errors.New("boom")
		}).
		ApplyFuncX(func(Kind) error {
			called = true

			return nil
		})

	g.Expect(returned).To(BeIdenticalTo(state))
	g.Expect(called).To(BeFalse())
	g.Expect(state.Err()).To(MatchError("boom"))
}
