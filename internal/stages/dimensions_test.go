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

// TestResolveDimensions_PartialDimensions_UsesDefaultForMissing verifies
// that when only one dimension is set in config, the other falls back to
// the default (1920×1080). scan_test.go covers nil-config and full-config cases.
func TestResolveDimensions_PartialDimensions_UsesDefaultForMissing(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w := 2560
	c := &stages.CommonState{
		RootConfig: &config.Config{
			ImageSize: &config.ImageSize{Width: &w},
		},
	}
	err := stages.ResolveDimensions(c)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.Width).To(Equal(2560))
	g.Expect(c.Height).To(Equal(1080))
}

func TestResolveDimensions_HeightOnly_WidthDefault(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	h := 2160
	c := &stages.CommonState{
		RootConfig: &config.Config{
			ImageSize: &config.ImageSize{Height: &h},
		},
	}
	err := stages.ResolveDimensions(c)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.Width).To(Equal(1920))
	g.Expect(c.Height).To(Equal(2160))
}

// ---------------------------------------------------------------------------
// DrawingBounds
// ---------------------------------------------------------------------------

func TestDrawingBoundsWidth_ReturnsMaxXMinusMinX(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := stages.DrawingBounds{MinX: 10, MinY: 0, MaxX: 90, MaxY: 0}
	g.Expect(b.Width()).To(Equal(80))
}

func TestDrawingBoundsWidth_ZeroWidth_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := stages.DrawingBounds{MinX: 50, MaxX: 50}
	g.Expect(b.Width()).To(Equal(0))
}

func TestDrawingBoundsHeight_ReturnsMaxYMinusMinY(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := stages.DrawingBounds{MinX: 0, MinY: 20, MaxX: 0, MaxY: 180}
	g.Expect(b.Height()).To(Equal(160))
}

func TestDrawingBoundsHeight_ZeroHeight_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := stages.DrawingBounds{MinY: 100, MaxY: 100}
	g.Expect(b.Height()).To(Equal(0))
}

func TestDrawingBounds_FullCanvas_WidthAndHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	b := stages.DrawingBounds{MinX: 0, MinY: 0, MaxX: 1920, MaxY: 1080}
	g.Expect(b.Width()).To(Equal(1920))
	g.Expect(b.Height()).To(Equal(1080))
}

