package model

import (
	"maps"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// WalkFiles calls fn for every file in the tree, depth-first.
func WalkFiles(dir *Directory, fn func(*File)) {
	if dir == nil {
		return
	}

	for _, f := range dir.Files {
		fn(f)
	}

	for _, d := range dir.Dirs {
		if d != nil {
			WalkFiles(d, fn)
		}
	}
}

// CountFiles returns the total number of files in the tree.
func CountFiles(dir *Directory) int {
	count := len(dir.Files)
	for _, d := range dir.Dirs {
		count += CountFiles(d)
	}

	return count
}

// CountDirs returns the total number of subdirectories in the tree,
// not counting dir itself. This is the counterpart to CountFiles.
func CountDirs(dir *Directory) int {
	count := len(dir.Dirs)
	for _, d := range dir.Dirs {
		count += CountDirs(d)
	}

	return count
}

// PruneLayers returns a shallow copy of dir with every directory deeper than
// maxLayers pruned from the render tree. A value of 0 keeps the original tree
// unchanged; a value of 1 keeps the root and its immediate children.
func PruneLayers(dir *Directory, maxLayers int) *Directory {
	if dir == nil || maxLayers <= 0 {
		return dir
	}

	return pruneDirectoryLayers(dir, 0, maxLayers)
}

func pruneDirectoryLayers(dir *Directory, depth, maxLayers int) *Directory {
	if dir == nil {
		return nil
	}

	pruned := &Directory{
		Path:            dir.Path,
		Name:            dir.Name,
		Files:           dir.Files,
		DirectFileCount: dir.DirectFileCount,
		AllFileCount:    dir.AllFileCount,
		AllDirCount:     dir.AllDirCount,
	}
	cloneMetricContainer(&dir.MetricContainer, &pruned.MetricContainer)

	if maxLayers > 0 && depth >= maxLayers {
		pruned.Dirs = nil

		return pruned
	}

	pruned.Dirs = make([]*Directory, 0, len(dir.Dirs))
	for _, child := range dir.Dirs {
		if child == nil {
			continue
		}

		if limited := pruneDirectoryLayers(child, depth+1, maxLayers); limited != nil {
			pruned.Dirs = append(pruned.Dirs, limited)
		}
	}

	return pruned
}

func cloneMetricContainer(src *MetricContainer, dst *MetricContainer) {
	if src == nil || dst == nil {
		return
	}

	src.mu.RLock()
	defer src.mu.RUnlock()

	if src.quantities != nil {
		dst.quantities = make(map[metric.Name]int64, len(src.quantities))
		maps.Copy(dst.quantities, src.quantities)
	}

	if src.measures != nil {
		dst.measures = make(map[metric.Name]float64, len(src.measures))
		maps.Copy(dst.measures, src.measures)
	}

	if src.classifications != nil {
		dst.classifications = make(map[metric.Name]string, len(src.classifications))
		maps.Copy(dst.classifications, src.classifications)
	}
}

// WalkDirectories calls fn for every directory in the tree, in post-order
// (children before parents). The root directory itself is included as the
// final call. Post-order guarantees that child metrics are fully populated
// before a parent directory is visited — useful for computing roll-up metrics
// such as directory file-counts or aggregated sizes.
func WalkDirectories(dir *Directory, fn func(*Directory)) {
	for _, d := range dir.Dirs {
		WalkDirectories(d, fn)
	}

	fn(dir)
}

// CountDeclarations returns the total number of declarations across all files
// in the tree. It is the declaration-level counterpart to CountFiles.
func CountDeclarations(dir *Directory) int {
	count := 0

	WalkFiles(dir, func(f *File) {
		count += len(f.Declarations)
	})

	return count
}

// CountCommits returns the total number of commit records across all files in
// the tree. It is the commit-level counterpart to CountFiles.
func CountCommits(dir *Directory) int {
	count := 0

	WalkFiles(dir, func(f *File) {
		count += len(f.Commits)
	})

	return count
}

// WalkDeclarations calls fn for every declaration in every file in the tree.
func WalkDeclarations(dir *Directory, fn func(*Declaration, *File)) {
	WalkFiles(dir, func(f *File) {
		for _, d := range f.Declarations {
			fn(d, f)
		}
	})
}

// WalkCommits calls fn for every commit record in every file in the tree.
func WalkCommits(dir *Directory, fn func(*Commit, *File)) {
	WalkFiles(dir, func(f *File) {
		for _, c := range f.Commits {
			fn(c, f)
		}
	})
}
