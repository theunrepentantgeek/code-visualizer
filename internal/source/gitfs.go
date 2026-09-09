package source

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"time"

	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type gitFS struct {
	tree    *object.Tree
	modTime time.Time
}

var (
	_ fs.FS         = (*gitFS)(nil)
	_ fs.ReadDirFS  = (*gitFS)(nil)
	_ fs.ReadFileFS = (*gitFS)(nil)
	_ fs.StatFS     = (*gitFS)(nil)
	_ fs.ReadLinkFS = (*gitFS)(nil)
)

// NewGitFS exposes tree as a read-only standard-library filesystem.
func NewGitFS(tree *object.Tree, modTime time.Time) fs.FS {
	return &gitFS{tree: tree, modTime: modTime}
}

func (g *gitFS) Open(name string) (fs.File, error) {
	if err := validGitFSPath("open", name); err != nil {
		return nil, err
	}

	if name == "." {
		return g.openDir(name, g.tree)
	}

	entry, err := g.tree.FindEntry(name)
	if err != nil {
		return nil, gitFSPathError("open", name, err)
	}

	switch entry.Mode {
	case filemode.Dir:
		tree, treeErr := g.tree.Tree(name)
		if treeErr != nil {
			return nil, gitFSPathError("open", name, treeErr)
		}

		return g.openDir(name, tree)
	case filemode.Submodule:
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	default:
		file, fileErr := g.tree.TreeEntryFile(entry)
		if fileErr != nil {
			return nil, gitFSPathError("open", name, fileErr)
		}

		reader, readerErr := file.Reader()
		if readerErr != nil {
			return nil, gitFSPathError("open", name, readerErr)
		}

		return &gitFile{
			reader: reader,
			info:   g.info(entry.Name, entry.Mode, file.Size),
		}, nil
	}
}

func (g *gitFS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := g.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	dir, ok := file.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
	}

	return dir.ReadDir(-1)
}

func (g *gitFS) ReadFile(name string) ([]byte, error) {
	file, err := g.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, &fs.PathError{Op: "read", Path: name, Err: err}
	}

	return data, nil
}

func (g *gitFS) Stat(name string) (fs.FileInfo, error) {
	if err := validGitFSPath("stat", name); err != nil {
		return nil, err
	}

	if name == "." {
		return g.info(".", filemode.Dir, 0), nil
	}

	entry, err := g.tree.FindEntry(name)
	if err != nil {
		return nil, gitFSPathError("stat", name, err)
	}
	if entry.Mode == filemode.Submodule {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	size := int64(0)
	if entry.Mode != filemode.Dir {
		size, err = g.tree.Size(name)
		if err != nil {
			return nil, gitFSPathError("stat", name, err)
		}
	}

	return g.info(path.Base(name), entry.Mode, size), nil
}

func (g *gitFS) Lstat(name string) (fs.FileInfo, error) {
	return g.Stat(name)
}

func (g *gitFS) ReadLink(name string) (string, error) {
	if err := validGitFSPath("readlink", name); err != nil {
		return "", err
	}

	entry, err := g.tree.FindEntry(name)
	if err != nil {
		return "", gitFSPathError("readlink", name, err)
	}
	if entry.Mode != filemode.Symlink {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrInvalid}
	}

	data, err := g.ReadFile(name)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (g *gitFS) openDir(name string, tree *object.Tree) (fs.File, error) {
	entries := make([]fs.DirEntry, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		if entry.Mode == filemode.Submodule {
			continue
		}

		size := int64(0)
		if entry.Mode != filemode.Dir {
			entryName := entry.Name
			if name != "." {
				entryName = path.Join(name, entry.Name)
			}

			var err error
			size, err = g.tree.Size(entryName)
			if err != nil {
				return nil, gitFSPathError("readdir", entryName, err)
			}
		}

		entries = append(entries, gitDirEntry{info: g.info(entry.Name, entry.Mode, size)})
	}

	return &gitDir{info: g.info(path.Base(name), filemode.Dir, 0), entries: entries}, nil
}

func (g *gitFS) info(name string, mode filemode.FileMode, size int64) gitFileInfo {
	return gitFileInfo{name: name, mode: gitMode(mode), size: size, modTime: g.modTime}
}

func gitMode(mode filemode.FileMode) fs.FileMode {
	switch mode {
	case filemode.Dir:
		return fs.ModeDir | 0o755
	case filemode.Executable:
		return 0o755
	case filemode.Symlink:
		return fs.ModeSymlink | 0o777
	default:
		return 0o644
	}
}

func validGitFSPath(op, name string) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	return nil
}

func gitFSPathError(op, name string, err error) error {
	if errors.Is(err, object.ErrEntryNotFound) ||
		errors.Is(err, object.ErrFileNotFound) ||
		errors.Is(err, object.ErrDirectoryNotFound) {
		err = fs.ErrNotExist
	}

	return &fs.PathError{Op: op, Path: name, Err: err}
}
