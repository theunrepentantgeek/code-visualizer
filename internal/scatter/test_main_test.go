package scatter

import (
	"os"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

func TestMain(m *testing.M) {
	filesystem.Register()
	git.Register()
	os.Exit(m.Run())
}
