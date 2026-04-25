package model

// WalkFiles calls fn for every file in the tree, depth-first.
func WalkFiles(dir *Directory, fn func(*File)) {
	for _, f := range dir.Files {
		fn(f)
	}

	for _, d := range dir.Dirs {
		WalkFiles(d, fn)
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
