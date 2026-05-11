package canvas

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
)

// Backend is re-exported from model for backward compatibility.
type Backend = model.Backend

// DefaultFontSize signals that the backend should use its built-in default
// font size. Callers can set FontSize to this value instead of a bare 0.
const DefaultFontSize = model.DefaultFontSize
