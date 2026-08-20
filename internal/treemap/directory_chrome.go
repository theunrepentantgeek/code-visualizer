package treemap

import "github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"

const (
	directoryRailThickness    = 20.0
	directoryPadding          = 4.0
	directoryLabelFontSize    = 12.0
	minDirectoryContentSize   = 20.0
	minTruncatedRunes         = 4
	directoryLabelEllipsis    = "…"
	directoryLabelEndpointPad = 2 * directoryPadding
)

func resolveDirectoryChrome(rect RectangleBounds, name string, isRoot bool) DirectoryChrome {
	if isRoot || name == "" {
		return directoryChromeBorderOnly(rect)
	}

	if rect.W >= rect.H {
		chrome, ok := resolveTopDirectoryChrome(rect, name)
		if !ok {
			return directoryChromeBorderOnly(rect)
		}

		return chrome
	}

	chrome, ok := resolveLeftDirectoryChrome(rect, name)
	if !ok {
		return directoryChromeBorderOnly(rect)
	}

	return chrome
}

func resolveTopDirectoryChrome(rect RectangleBounds, name string) (DirectoryChrome, bool) {
	content := RectangleBounds{
		X: rect.X + directoryPadding,
		Y: rect.Y + directoryRailThickness,
		W: rect.W - 2*directoryPadding,
		H: rect.H - directoryRailThickness - directoryPadding,
	}
	if content.W < minDirectoryContentSize || content.H < minDirectoryContentSize {
		return DirectoryChrome{}, false
	}

	text, ok := fitDirectoryLabel(name, rect.W-directoryLabelEndpointPad)
	if !ok {
		return DirectoryChrome{}, false
	}

	return DirectoryChrome{
		Orientation: DirectoryLabelTop,
		Text:        text,
		Rail: RectangleBounds{
			X: rect.X,
			Y: rect.Y,
			W: rect.W,
			H: directoryRailThickness,
		},
		Content: content,
	}, true
}

func resolveLeftDirectoryChrome(rect RectangleBounds, name string) (DirectoryChrome, bool) {
	content := RectangleBounds{
		X: rect.X + directoryRailThickness,
		Y: rect.Y + directoryPadding,
		W: rect.W - directoryRailThickness - directoryPadding,
		H: rect.H - 2*directoryPadding,
	}
	if content.W < minDirectoryContentSize || content.H < minDirectoryContentSize {
		return DirectoryChrome{}, false
	}

	text, ok := fitDirectoryLabel(name, rect.H-directoryLabelEndpointPad)
	if !ok {
		return DirectoryChrome{}, false
	}

	return DirectoryChrome{
		Orientation: DirectoryLabelLeft,
		Text:        text,
		Rail: RectangleBounds{
			X: rect.X,
			Y: rect.Y,
			W: directoryRailThickness,
			H: rect.H,
		},
		Content: content,
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

func directoryChromeBorderOnly(rect RectangleBounds) DirectoryChrome {
	return DirectoryChrome{
		Orientation: DirectoryLabelNone,
		Content: RectangleBounds{
			X: rect.X + directoryPadding,
			Y: rect.Y + directoryPadding,
			W: rect.W - 2*directoryPadding,
			H: rect.H - 2*directoryPadding,
		},
	}
}
