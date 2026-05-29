// Package walk provides traversal and aggregation helpers for model trees.
// Traversal algorithms belong here rather than in the model package,
// which should remain a pure data definition.
package walk

import "github.com/theunrepentantgeek/code-visualizer/internal/model"

// Files calls fn for every file in the tree, depth-first.
func Files(dir *model.Directory, fn func(*model.File)) {
	for _, f := range dir.Files {
		fn(f)
	}

	for _, d := range dir.Dirs {
		Files(d, fn)
	}
}

// CountFiles returns the total number of files in the tree.
func CountFiles(dir *model.Directory) int {
	count := len(dir.Files)
	for _, d := range dir.Dirs {
		count += CountFiles(d)
	}

	return count
}

// CountDirs returns the total number of subdirectories in the tree,
// not counting dir itself. This is the counterpart to CountFiles.
func CountDirs(dir *model.Directory) int {
	count := len(dir.Dirs)
	for _, d := range dir.Dirs {
		count += CountDirs(d)
	}

	return count
}

// Directories calls fn for every directory in the tree, in post-order
// (children before parents). The root directory itself is included as the
// final call. Post-order guarantees that child metrics are fully populated
// before a parent directory is visited — useful for computing roll-up metrics
// such as directory file-counts or aggregated sizes.
func Directories(dir *model.Directory, fn func(*model.Directory)) {
	for _, d := range dir.Dirs {
		Directories(d, fn)
	}

	fn(dir)
}
