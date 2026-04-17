package render

import (
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
)

// ImageFormat represents a supported output image format.
type ImageFormat int

const (
	FormatPNG ImageFormat = iota
	FormatJPG
	FormatSVG
)

// FormatFromPath determines the image format from the file extension of the
// given path. The match is case-insensitive. Both ".jpg" and ".jpeg" map to
// FormatJPG. An error is returned for unsupported or missing extensions.
func FormatFromPath(path string) (ImageFormat, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".png":
		return FormatPNG, nil
	case ".jpg", ".jpeg":
		return FormatJPG, nil
	case ".svg":
		return FormatSVG, nil
	case "":
		return 0, eris.New("output path has no file extension; supported formats: png, jpg, jpeg, svg")
	default:
		return 0, eris.Errorf("unsupported image format %q; supported formats: png, jpg, jpeg, svg", ext)
	}
}
