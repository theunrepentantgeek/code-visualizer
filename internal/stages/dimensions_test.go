package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
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

// ResolveDimensions

func TestResolveDimensions_NilConfig_UsesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{RootConfig: nil}
	g.Expect(stages.ResolveDimensions(c)).To(Succeed())
	g.Expect(c.Width).To(Equal(1920))
	g.Expect(c.Height).To(Equal(1080))
}

func TestResolveDimensions_ConfigNilImageSize_UsesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := &config.Config{}
	c := &stages.CommonState{RootConfig: cfg}
	g.Expect(stages.ResolveDimensions(c)).To(Succeed())
	g.Expect(c.Width).To(Equal(1920))
	g.Expect(c.Height).To(Equal(1080))
}

func TestResolveDimensions_ExplicitWidthAndHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideWidth(2560)
	cfg.OverrideHeight(1440)

	c := &stages.CommonState{RootConfig: cfg}
	g.Expect(stages.ResolveDimensions(c)).To(Succeed())
	g.Expect(c.Width).To(Equal(2560))
	g.Expect(c.Height).To(Equal(1440))
}

func TestResolveDimensions_ExplicitWidthOnly_HeightDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideWidth(3840)

	c := &stages.CommonState{RootConfig: cfg}
	g.Expect(stages.ResolveDimensions(c)).To(Succeed())
	g.Expect(c.Width).To(Equal(3840))
	g.Expect(c.Height).To(Equal(1080))
}

func TestResolveDimensions_ExplicitHeightOnly_WidthDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideHeight(2160)

	c := &stages.CommonState{RootConfig: cfg}
	g.Expect(stages.ResolveDimensions(c)).To(Succeed())
	g.Expect(c.Width).To(Equal(1920))
	g.Expect(c.Height).To(Equal(2160))
}
