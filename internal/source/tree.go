// Package source defines read-only trees consumed by scanners and providers.
package source

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"
)

// Tree identifies a content filesystem and its repository-relative root.
type Tree struct {
	FS       fs.FS
	RootName string
	RootPath string
	RepoRoot string
	RepoBase string
	Clock    time.Time
}

// WorkingTree wraps path as an OS-backed content source.
func WorkingTree(rootPath string) (Tree, error) {
	absolute, err := filepath.Abs(rootPath)
	if err != nil {
		return Tree{}, eris.Wrap(err, "failed to resolve source path")
	}

	return Tree{
		FS:       os.DirFS(absolute),
		RootName: filepath.Base(absolute),
		RootPath: absolute,
	}, nil
}

// RepoPath returns name relative to the containing repository.
func (t Tree) RepoPath(name string) string {
	if t.RepoBase == "" || t.RepoBase == "." {
		return path.Clean(name)
	}

	return path.Join(t.RepoBase, name)
}
