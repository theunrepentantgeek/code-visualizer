package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
)

func TestResolveDirectoryChrome(t *testing.T) {
	t.Parallel()

	t.Run("root has no rail and border-only content", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 10, Y: 20, W: 100, H: 60}, "root", true)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 14, Y: 24, W: 92, H: 52}))
	})

	t.Run("wide rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 0, Y: 0, W: 120, H: 60}, "alpha", false)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{X: 0, Y: 0, W: 120, H: 20}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 4, Y: 20, W: 112, H: 36}))
	})

	t.Run("square rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 5, Y: 6, W: 60, H: 60}, "alpha", false)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{X: 5, Y: 6, W: 60, H: 20}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 9, Y: 26, W: 52, H: 36}))
	})

	t.Run("tall rectangle uses left chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 3, Y: 7, W: 60, H: 120}, "alpha", false)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelLeft))
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{X: 3, Y: 7, W: 20, H: 120}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 23, Y: 11, W: 36, H: 112}))
	})

	t.Run("narrow top content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 0, Y: 0, W: 100, H: 39}, "alpha", false)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 4, Y: 4, W: 92, H: 31}))
	})

	t.Run("narrow left content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(RectangleBounds{X: 0, Y: 0, W: 39, H: 100}, "alpha", false)

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(RectangleBounds{}))
		g.Expect(chrome.Content).To(Equal(RectangleBounds{X: 4, Y: 4, W: 31, H: 92}))
	})
}

func TestFitDirectoryLabel(t *testing.T) {
	t.Parallel()

	t.Run("retains a short complete name", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		width, _ := textlayout.MeasureString("cmd", directoryLabelFontSize)
		text, ok := fitDirectoryLabel("cmd", width+1)

		g.Expect(ok).To(BeTrue())
		g.Expect(text).To(Equal("cmd"))
	})

	t.Run("truncates an ascii name at a measured boundary", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		shortWidth, _ := textlayout.MeasureString("abcd…", directoryLabelFontSize)
		longWidth, _ := textlayout.MeasureString("abcde…", directoryLabelFontSize)
		text, ok := fitDirectoryLabel("abcdefgh", (shortWidth+longWidth)/2)

		g.Expect(ok).To(BeTrue())
		g.Expect(text).To(Equal("abcd…"))
	})

	t.Run("omits text below the truncated boundary", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		width, _ := textlayout.MeasureString("abcd…", directoryLabelFontSize)
		text, ok := fitDirectoryLabel("abcdefgh", width-0.01)

		g.Expect(ok).To(BeFalse())
		g.Expect(text).To(BeEmpty())
	})

	t.Run("truncates unicode names rune-safely", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		shortWidth, _ := textlayout.MeasureString("αβγδ…", directoryLabelFontSize)
		longWidth, _ := textlayout.MeasureString("αβγδε…", directoryLabelFontSize)
		text, ok := fitDirectoryLabel("αβγδεζη", (shortWidth+longWidth)/2)

		g.Expect(ok).To(BeTrue())
		g.Expect(text).To(Equal("αβγδ…"))
	})
}
