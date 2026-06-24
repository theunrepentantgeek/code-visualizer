package scatter

import (
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/golang"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	golang.Register()
	m.Run()
}
