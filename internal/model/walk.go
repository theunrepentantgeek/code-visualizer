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
