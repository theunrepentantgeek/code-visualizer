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
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 14, Y: 24}, Max: geometry.Point{X: 106, Y: 76}}))
	})

	t.Run("wide rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 120, H: 60}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{Min: geometry.Point{X: 0, Y: 0}, Max: geometry.Point{X: 120, Y: 20}}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 4, Y: 20}, Max: geometry.Point{X: 116, Y: 56}}))
	})

	t.Run("square rectangle uses top chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 5, Y: 6, W: 60, H: 60}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelTop))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{Min: geometry.Point{X: 5, Y: 6}, Max: geometry.Point{X: 65, Y: 26}}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 9, Y: 26}, Max: geometry.Point{X: 61, Y: 62}}))
	})

	t.Run("tall rectangle uses left chrome", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 3, Y: 7, W: 60, H: 120}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelLeft))
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{Min: geometry.Point{X: 3, Y: 7}, Max: geometry.Point{X: 23, Y: 127}}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 23, Y: 11}, Max: geometry.Point{X: 59, Y: 123}}))
	})

	t.Run("narrow top content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 100, H: 39}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 4, Y: 4}, Max: geometry.Point{X: 96, Y: 35}}))
	})

	t.Run("narrow left content falls back to border-only", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		chrome := resolveDirectoryChrome(layout.Box{X: 0, Y: 0, W: 39, H: 100}, "alpha")

		g.Expect(chrome.Orientation).To(Equal(DirectoryLabelNone))
		g.Expect(chrome.Text).To(BeEmpty())
		g.Expect(chrome.Rail).To(Equal(geometry.Rect{}))
		g.Expect(chrome.Content).To(Equal(geometry.Rect{Min: geometry.Point{X: 4, Y: 4}, Max: geometry.Point{X: 35, Y: 96}}))
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
