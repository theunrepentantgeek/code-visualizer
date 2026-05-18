package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestPtrInt_NilReturnsDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.PtrInt(nil, 42)).To(Equal(42))
}

func TestPtrInt_NonNilReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	v := 7
	g.Expect(stages.PtrInt(&v, 42)).To(Equal(7))
}

func TestPtrInt_ZeroValuePtr(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	v := 0
	g.Expect(stages.PtrInt(&v, 42)).To(Equal(0))
}

func TestPtrString_NilReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.PtrString(nil)).To(BeEmpty())
}

func TestPtrString_NonNilReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := "hello"
	g.Expect(stages.PtrString(&s)).To(Equal("hello"))
}

func TestPtrString_EmptyStringPtr(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := ""
	g.Expect(stages.PtrString(&s)).To(BeEmpty())
}
