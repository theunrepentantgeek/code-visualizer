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

const maxSymlinkDepth = 40

func newFSWalker(tree source.Tree, rules []filter.Rule, progress Progress, includeBinary bool) fsWalker {
	return fsWalker{tree: tree, rules: rules, progress: progress, includeBinary: includeBinary}
}

func (w fsWalker) scanDir(name string) (*model.Directory, error) {
	entries, err := fs.ReadDir(w.tree.FS, name)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", w.displayPath(name))
	}

	node := &model.Directory{
		Path:          w.displayPath(name),
		RepoPath:      w.tree.RepoPath(name),
		Name:          w.directoryName(name),
		Source:        w.tree.FS,
		ReferenceTime: w.tree.Clock,
		Files:         make([]*model.File, 0, len(entries)),
		Dirs:          make([]*model.Directory, 0, len(entries)),
	}

	for _, entry := range entries {
		if err := w.processEntry(node, name, entry); err != nil {
			return nil, err
		}
	}

	populateFSCounts(node)

	if w.progress != nil {
		w.progress.OnDirectoryScanned(node.Path, len(node.Files))
	}

	return node, nil
}

func (w fsWalker) processEntry(node *model.Directory, dirName string, entry fs.DirEntry) error {
	entryName := path.Join(dirName, entry.Name())
	if dirName == "." {
		entryName = entry.Name()
	}

	if !filter.IsIncluded(filepath.FromSlash(entryName), w.rules) {
		return nil
	}

	if entry.Type()&fs.ModeSymlink != 0 {
		return w.processSymlink(node, entryName, entry)
	}

	if entry.IsDir() {
		return w.processDir(node, entryName)
	}

	if entry.Type().IsRegular() {
		return w.processFile(node, entryName, entry.Name())
	}

	return nil
}

func (w fsWalker) processDir(node *model.Directory, name string) error {
	child, err := w.scanDir(name)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("permission denied, skipping directory", "path", w.displayPath(name), "error", err)

			return nil
		}

		return err
	}

	if hasFiles(child) {
		node.Dirs = append(node.Dirs, child)
	}

	return nil
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
		RepoSource: w.tree.RepoFS,
	}
	file.SetQuantity(filesystem.FileSize, info.Size())
	file.SetClassification(filesystem.FileType, fileType)
	node.Files = append(node.Files, file)

	return nil
}

func (w fsWalker) processSymlink(node *model.Directory, sourcePath string, entry fs.DirEntry) error {
	resolved, ok := w.resolveFileSymlink(sourcePath)
	if !ok {
		return nil
	}

	if err := w.processFile(node, resolved, entry.Name()); err != nil {
		return err
	}

	if len(node.Files) > 0 {
		file := node.Files[len(node.Files)-1]
		file.Path = w.displayPath(sourcePath)
		file.RepoPath = w.tree.RepoPath(sourcePath)
	}

	return nil
}

func (w fsWalker) resolveFileSymlink(sourcePath string) (string, bool) {
	return w.resolveSymlinkPath(sourcePath, sourcePath, make(map[string]struct{}), 0)
}

func (w fsWalker) resolveSymlinkPath(
	sourcePath,
	name string,
	seen map[string]struct{},
	depth int,
) (string, bool) {
	components := strings.Split(name, "/")
	for i := range components {
		current := path.Join(components[:i+1]...)

		info, err := fs.Lstat(w.tree.FS, current)
		if err != nil || info == nil {
			slog.Debug("skipping broken symlink", "path", w.displayPath(sourcePath), "error", err)

			return "", false
		}

		if info.Mode()&fs.ModeSymlink != 0 {
			remaining := strings.Join(components[i+1:], "/")

			return w.followSymlink(sourcePath, current, remaining, seen, depth)
		}

		if i < len(components)-1 && !info.IsDir() {
			return "", false
		}
	}

	info, err := fs.Lstat(w.tree.FS, name)

	return name, err == nil && info != nil && info.Mode().IsRegular()
}

func (w fsWalker) followSymlink(
	sourcePath,
	current,
	remaining string,
	seen map[string]struct{},
	depth int,
) (string, bool) {
	if _, duplicate := seen[current]; duplicate {
		slog.Debug("skipping cyclic symlink", "path", w.displayPath(sourcePath))

		return "", false
	}

	if depth >= maxSymlinkDepth {
		slog.Debug("skipping deeply nested symlink", "path", w.displayPath(sourcePath))

		return "", false
	}

	seen[current] = struct{}{}

	target, err := fs.ReadLink(w.tree.FS, current)
	if err != nil {
		slog.Warn("skipping file", "path", w.displayPath(sourcePath), "error", err)

		return "", false
	}

	var resolvedBase string

	if filepath.IsAbs(target) {
		relative, relativeErr := filepath.Rel(w.tree.RootPath, target)
		if relativeErr != nil {
			slog.Warn("skipping symlink outside source", "path", w.displayPath(sourcePath))

			return "", false
		}

		resolvedBase = filepath.ToSlash(relative)
	} else {
		resolvedBase = path.Join(path.Dir(current), target)
	}

	resolved := path.Clean(path.Join(resolvedBase, remaining))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		slog.Warn("skipping symlink outside source", "path", w.displayPath(sourcePath))

		return "", false
	}

	return w.resolveSymlinkPath(sourcePath, resolved, seen, depth+1)
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

func populateFSCounts(node *model.Directory) {
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
		return false, eris.Wrap(err, "opening file for binary probe")
	}
	defer file.Close()

	header := make([]byte, binaryProbeSize)

	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, eris.Wrap(err, "reading file for binary probe")
	}

	if n == 0 || hasUTF16BOM(header[:n]) {
		return false, nil
	}

	return bytes.IndexByte(header[:n], 0) >= 0, nil
}
