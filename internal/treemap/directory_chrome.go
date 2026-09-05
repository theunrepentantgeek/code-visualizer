package treemap

import (
	"github.com/nikolaydubina/treemap/layout"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
)

const (
	directoryRailThickness    = 20.0
	directoryPadding          = 4.0
	directoryLabelFontSize    = 12.0
	minDirectoryContentSize   = 20.0
	minTruncatedRunes         = 4
	directoryLabelEllipsis    = "…"
	directoryLabelEndpointPad = 2 * directoryPadding
)

// resolveDirectoryChrome and its helpers below compute in third-party
// layout.Box (position+size) terms throughout, matching the exact
// floating-point arithmetic Squarify itself uses. geometry.Rect is applied
// only once, at the boundary where results are stored on DirectoryChrome, per
// the migration's "convert third-party layout.Box values through
// RectFromPositionSize" guidance. Re-deriving a width from a stored Rect's
// Max-Min is not guaranteed to reproduce a box's original width bit-for-bit,
// which would perturb the Squarify calls for descendants and change golden
// raster output.
func resolveDirectoryChrome(box layout.Box, name string) DirectoryChrome {
	if name == "" {
		return directoryChromeBorderOnly(box)
	}

	if box.W >= box.H {
		chrome, ok := resolveTopDirectoryChrome(box, name)
		if !ok {
			return directoryChromeBorderOnly(box)
		}

		return chrome
	}

	chrome, ok := resolveLeftDirectoryChrome(box, name)
	if !ok {
		return directoryChromeBorderOnly(box)
	}

	return chrome
}

func resolveTopDirectoryChrome(box layout.Box, name string) (DirectoryChrome, bool) {
	content := directoryContentBox(box, DirectoryLabelTop)
	if content.W < minDirectoryContentSize || content.H < minDirectoryContentSize {
		return DirectoryChrome{}, false
	}

	text, ok := fitDirectoryLabel(name, box.W-directoryLabelEndpointPad)
	if !ok {
		return DirectoryChrome{}, false
	}

	rail := layout.Box{X: box.X, Y: box.Y, W: box.W, H: directoryRailThickness}

	return DirectoryChrome{
		Orientation: DirectoryLabelTop,
		Text:        text,
		Rail:        boxToRect(rail),
		Content:     boxToRect(content),
	}, true
}

func resolveLeftDirectoryChrome(box layout.Box, name string) (DirectoryChrome, bool) {
	content := directoryContentBox(box, DirectoryLabelLeft)
	if content.W < minDirectoryContentSize || content.H < minDirectoryContentSize {
		return DirectoryChrome{}, false
	}

	text, ok := fitDirectoryLabel(name, box.H-directoryLabelEndpointPad)
	if !ok {
		return DirectoryChrome{}, false
	}

	rail := layout.Box{X: box.X, Y: box.Y, W: directoryRailThickness, H: box.H}

	return DirectoryChrome{
		Orientation: DirectoryLabelLeft,
		Text:        text,
		Rail:        boxToRect(rail),
		Content:     boxToRect(content),
	}, true
}

func fitDirectoryLabel(name string, maxWidth float64) (string, bool) {
	if maxWidth <= 0 {
		return "", false
	}

	if name == "" {
		return "", false
	}

	runes := []rune(name)
	candidates := make([]string, 0, 1)
	candidates = append(candidates, name)

	if len(runes) >= minTruncatedRunes {
		for truncated := len(runes) - 1; truncated >= minTruncatedRunes; truncated-- {
			candidates = append(candidates, string(runes[:truncated])+directoryLabelEllipsis)
		}
	}

	widths, _ := textlayout.MeasureStrings(candidates, directoryLabelFontSize)
	for i, width := range widths {
		if width <= maxWidth {
			return candidates[i], true
		}
	}

	return "", false
}

func directoryChromeBorderOnly(box layout.Box) DirectoryChrome {
	return DirectoryChrome{
		Orientation: DirectoryLabelNone,
		Content:     boxToRect(directoryContentBox(box, DirectoryLabelNone)),
	}
}

// directoryContentBox returns the usable content area within box for the
// given chrome orientation, in layout.Box terms so callers can feed it
// straight into layout.Squarify without a Rect round-trip.
func directoryContentBox(box layout.Box, orientation DirectoryLabelOrientation) layout.Box {
	switch orientation {
	case DirectoryLabelTop:
		return layout.Box{
			X: box.X + directoryPadding,
			Y: box.Y + directoryRailThickness,
			W: box.W - 2*directoryPadding,
			H: box.H - directoryRailThickness - directoryPadding,
		}
	case DirectoryLabelLeft:
		return layout.Box{
			X: box.X + directoryRailThickness,
			Y: box.Y + directoryPadding,
			W: box.W - directoryRailThickness - directoryPadding,
			H: box.H - 2*directoryPadding,
		}
	default:
		return layout.Box{
			X: box.X + directoryPadding,
			Y: box.Y + directoryPadding,
			W: box.W - 2*directoryPadding,
			H: box.H - 2*directoryPadding,
		}
	}
}
