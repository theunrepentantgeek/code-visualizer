package stages

import "github.com/theunrepentantgeek/code-visualizer/internal/pipeline"

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

// ResolveDimensions populates Common().Width and Common().Height from
// RootConfig, applying the documented defaults (1920x1080).
func ResolveDimensions[S VizState](s S) error {
	c := s.Common()
	c.Width = PtrInt(c.RootConfig.Width, 1920)
	c.Height = PtrInt(c.RootConfig.Height, 1080)

	return nil
}

var _ pipeline.Stage[VizState] = ResolveDimensions[VizState]
