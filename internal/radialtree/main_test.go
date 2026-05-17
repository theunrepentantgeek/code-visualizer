package radialtree

import (
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	m.Run()
}
