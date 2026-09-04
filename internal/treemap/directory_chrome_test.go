package treemap

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/nikolaydubina/treemap/layout"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
)

func TestResolveDirectoryChrome(t *testing.T) {
	t.Parallel()

	t.Run("empty name has no rail and border-only content", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 10, Y: 20, W: 100, H: 60}, "")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(14, 24), Max: geometry.NewPoint(106, 76),
		}))
	})

	t.Run("wide rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 120, H: 60}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(0, 0), Max: geometry.NewPoint(120, 20),
		}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(4, 20), Max: geometry.NewPoint(116, 56),
		}))
	})

	t.Run("square rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 5, Y: 6, W: 60, H: 60}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(5, 6), Max: geometry.NewPoint(65, 26),
		}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(9, 26), Max: geometry.NewPoint(61, 62),
		}))
	})

	t.Run("tall rectangle uses left chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 3, Y: 7, W: 60, H: 120}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelLeft))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(3, 7), Max: geometry.NewPoint(23, 127),
		}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(23, 11), Max: geometry.NewPoint(59, 123),
		}))
	})

	t.Run("narrow top content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 100, H: 39}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(4, 4), Max: geometry.NewPoint(96, 35),
		}))
	})

	t.Run("narrow left content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 39, H: 100}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{
			Min: geometry.NewPoint(4, 4), Max: geometry.NewPoint(35, 96),
		}))
	})
}

func TestFitDirectoryLabel(t *testing.T) {
	t.Parallel()

	t.Run("rejects an empty name", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		text, ok := fitDirectoryLabel("", 10)

		g.Expect(ok).To(BeFalse())
		g.Expect(text).To(BeEmpty())
	})

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
