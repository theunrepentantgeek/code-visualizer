package radialtree

import (
	"fmt"
	"testing"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
)

// BenchmarkCollectDiscs measures the disc-collection step on a synthetic tree
// representative of a mid-sized repository: 50 packages × 20 files each = 1 000 file nodes.
// This is the path where the accumulator pattern eliminates O(N) per-level intermediate
// slice copies compared to the naive recursive return-and-append approach.
func BenchmarkCollectDiscs(b *testing.B) {
	root := buildDiscBenchTree(50, 20)
	nodes := Layout(root, 800, filesystem.FileSize, LabelNone)
	cx := float64(800) / 2.0
	cy := cx

	b.ResetTimer()

	for range b.N {
		_ = collectDiscs(&nodes, root, cx, cy)
	}
}

func buildDiscBenchTree(numDirs, filesPerDir int) *model.Directory {
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
