package scan

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
)

type fsWalker struct {
	tree          source.Tree
	rules         []filter.Rule
	progress      Progress
	includeBinary bool
}

func newFSWalker(tree source.Tree, rules []filter.Rule, progress Progress, includeBinary bool) fsWalker {
	return fsWalker{tree: tree, rules: rules, progress: progress, includeBinary: includeBinary}
}

func (w fsWalker) scanDir(name string) (*model.Directory, error) {
	entries, err := fs.ReadDir(w.tree.FS, name)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", w.displayPath(name))
	}

	node := &model.Directory{
		Path:     w.displayPath(name),
		RepoPath: w.tree.RepoPath(name),
		Name:     w.directoryName(name),
		Source:   w.tree.FS,
		Files:    make([]*model.File, 0, len(entries)),
		Dirs:     make([]*model.Directory, 0, len(entries)),
	}

	for _, entry := range entries {
		entryName := path.Join(name, entry.Name())
		if name == "." {
			entryName = entry.Name()
		}
		if !filter.IsIncluded(filepath.FromSlash(entryName), w.rules) {
			continue
		}

		if entry.Type()&fs.ModeSymlink != 0 {
			if err := w.processSymlink(node, entryName, entry); err != nil {
				return nil, err
			}

			continue
		}
		if entry.IsDir() {
			child, childErr := w.scanDir(entryName)
			if childErr != nil {
				return nil, childErr
			}
			if hasFiles(child) {
				node.Dirs = append(node.Dirs, child)
			}

			continue
		}
		if entry.Type().IsRegular() {
			if err := w.processFile(node, entryName, entry.Name()); err != nil {
				return nil, err
			}
		}
	}

	w.populateCounts(node)
	if w.progress != nil {
		w.progress.OnDirectoryScanned(node.Path, len(node.Files))
	}

	return node, nil
}

func (w fsWalker) processFile(node *model.Directory, sourcePath, fileName string) error {
	info, err := fs.Stat(w.tree.FS, sourcePath)
	if err != nil {
		return eris.Wrapf(err, "failed to stat %s", sourcePath)
	}

	binary, err := isBinaryFS(w.tree.FS, sourcePath)
	if err != nil {
		slog.Warn("binary probe failed, assuming text", "path", w.displayPath(sourcePath), "error", err)
	}
	if binary && !w.includeBinary {
		return nil
	}

	ext := strings.TrimPrefix(path.Ext(fileName), ".")
	fileType := ext
	if fileType == "" {
		fileType = "no-extension"
	}

	file := &model.File{
		Path:       w.displayPath(sourcePath),
		RepoPath:   w.tree.RepoPath(sourcePath),
		SourcePath: sourcePath,
		Name:       fileName,
		Extension:  ext,
		IsBinary:   binary,
		Source:     w.tree.FS,
	}
	file.SetQuantity(filesystem.FileSize, info.Size())
	file.SetClassification(filesystem.FileType, fileType)
	node.Files = append(node.Files, file)

	return nil
}

func (w fsWalker) processSymlink(node *model.Directory, sourcePath string, entry fs.DirEntry) error {
	target, err := fs.ReadLink(w.tree.FS, sourcePath)
	if err != nil {
		slog.Warn("skipping file", "path", w.displayPath(sourcePath), "error", err)

		return nil
	}
	if path.IsAbs(target) {
		slog.Warn("skipping symlink outside source", "path", w.displayPath(sourcePath))

		return nil
	}

	resolved := path.Clean(path.Join(path.Dir(sourcePath), target))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		slog.Warn("skipping symlink outside source", "path", w.displayPath(sourcePath))

		return nil
	}
	info, err := fs.Stat(w.tree.FS, resolved)
	if err != nil || info.IsDir() {
		slog.Debug("skipping directory or broken symlink", "path", w.displayPath(sourcePath), "error", err)

		return nil
	}

	return w.processFile(node, resolved, entry.Name())
}

func (w fsWalker) displayPath(name string) string {
	if name == "." {
		return w.tree.RootPath
	}

	return filepath.Join(w.tree.RootPath, filepath.FromSlash(name))
}

func (w fsWalker) directoryName(name string) string {
	if name == "." {
		return w.tree.RootName
	}

	return path.Base(name)
}

func (w fsWalker) populateCounts(node *model.Directory) {
	node.DirectFileCount = len(node.Files)
	node.AllFileCount = node.DirectFileCount
	node.AllDirCount = len(node.Dirs)
	for _, child := range node.Dirs {
		node.AllFileCount += child.AllFileCount
		node.AllDirCount += child.AllDirCount
	}
}

func isBinaryFS(fsys fs.FS, name string) (bool, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return false, err
	}
	defer file.Close()

	header := make([]byte, binaryProbeSize)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if n == 0 || hasUTF16BOM(header[:n]) {
		return false, nil
	}

	return bytes.IndexByte(header[:n], 0) >= 0, nil
}
