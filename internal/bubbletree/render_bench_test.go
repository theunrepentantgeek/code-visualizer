package bubbletree

import (
	"fmt"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

// BenchmarkCollectBubbleDirEntries measures the directory-entry collection step
// on a synthetic tree with 50 subdirectories, each containing 20 files.
// This is the path where the accumulator pattern eliminates O(N) per-level
// intermediate slice copies from the naive recursive return-and-append approach.
func BenchmarkCollectBubbleDirEntries(b *testing.B) {
	root := buildBubbleBenchTree(50, 20)
	nodes := Layout(root, 1920, 1080, filesystem.FileSize, LabelNone)
	_, dirIndex := indexBubbleNodes(&nodes)

	b.ResetTimer()

	for range b.N {
		_ = collectBubbleDirEntries(dirIndex, root)
	}
}

func buildBubbleBenchTree(numDirs, filesPerDir int) *model.Directory {
	root := &model.Directory{Name: "root"}

	for d := range numDirs {
		dir := &model.Directory{
			Name: fmt.Sprintf("pkg%d", d),
			Path: fmt.Sprintf("pkg%d", d),
		}

		for f := range filesPerDir {
			file := &model.File{
				Name:      fmt.Sprintf("file%d.go", f),
				Path:      fmt.Sprintf("pkg%d/file%d.go", d, f),
				Extension: "go",
			}
			file.SetQuantity(filesystem.FileSize, int64((d*filesPerDir+f+1)*100))

			dir.Files = append(dir.Files, file)
		}

		root.Dirs = append(root.Dirs, dir)
	}

	return root
}
