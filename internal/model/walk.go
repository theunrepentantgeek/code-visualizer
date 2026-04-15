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

// WalkDirectories calls fn for every directory in the tree, post-order (deepest first).
// This ensures child directories are visited before their parents, enabling bottom-up aggregation.
func WalkDirectories(dir *Directory, fn func(*Directory)) {
	for _, d := range dir.Dirs {
		WalkDirectories(d, fn)
	}

	fn(dir)
}
