package treemap

import "github.com/theunrepentantgeek/code-visualizer/internal/geometry"

// DirectoryLabelOrientation identifies the chrome layout used for a directory label.
type DirectoryLabelOrientation uint8

const (
	DirectoryLabelNone DirectoryLabelOrientation = iota
	DirectoryLabelTop
	DirectoryLabelLeft
)

// DirectoryChrome describes the directory rail, label text, and usable content area.
type DirectoryChrome struct {
	Orientation DirectoryLabelOrientation
	Text        string
	Rail        geometry.Rect
	Content     geometry.Rect
}

// TreemapRectangle is a positioned visual element in the rendered treemap.
type TreemapRectangle struct {
	Bounds geometry.Rect
	// VisibleDepth counts directory nesting levels shown in the render: the
	// synthetic root directory is -1, and each visible child directory
	// increments by one from its parent. Only meaningful for directories;
	// files always retain the int zero value, regardless of nesting depth.
	VisibleDepth int
	Label        string
	IsDirectory  bool
	Chrome       DirectoryChrome
	Children     []TreemapRectangle
}
