package goldentest

import (
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

// TestMain registers all base-metric providers exactly as the CLI does, so the
// global metric registry is populated before any test resolves a metric.
func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	golang.Register()
	m.Run()
}
