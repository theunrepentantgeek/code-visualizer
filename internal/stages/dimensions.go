package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

// PtrInt safely dereferences *int, returning fallback if nil.
func PtrInt(p *int, fallback int) int {
	if p == nil {
		return fallback
	}

	return *p
}

// PtrString safely dereferences *string, returning "" if nil.
func PtrString(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

// ResolveDimensions populates c.Width and c.Height from RootConfig, applying
// the documented defaults (1920x1080).
func ResolveDimensions(c *CommonState) error {
	var imageSize *config.ImageSize
	if c.RootConfig != nil {
		imageSize = c.RootConfig.ImageSize
	}

	var width, height *int
	if imageSize != nil {
		width = imageSize.Width
		height = imageSize.Height
	}

	c.Width = PtrInt(width, 1920)
	c.Height = PtrInt(height, 1080)

	return nil
}
