// Package scan provides recursive directory scanning with symlink handling.
package scan

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/filter"
	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/filesystem"
)

// Scan recursively scans the directory at path and returns a model.Directory tree.
// File symlinks are followed; directory symlinks are skipped.
// Permission-denied errors are logged and scanning continues.
// Returns an error if the directory contains no files.
func Scan(path string, rules []filter.Rule) (*model.Directory, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve absolute path")
	}

	root, err := scanDir(absPath, absPath, rules)
	if err != nil {
		return nil, err
	}

	if countFiles(root) == 0 {
		return nil, errors.New("no files found in directory")
	}

	return root, nil
}

func scanDir(dirPath, rootPath string, rules []filter.Rule) (*model.Directory, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", dirPath)
	}

	node := &model.Directory{
		Path: dirPath,
		Name: filepath.Base(dirPath),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		if err := processEntry(node, entry, entryPath, rootPath, rules); err != nil {
			return nil, err
		}
	}

	return node, nil
}

func processEntry(node *model.Directory, entry os.DirEntry, entryPath, rootPath string, rules []filter.Rule) error {
	info, err := os.Stat(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping file: permission denied", "path", entryPath)

			return nil
		}

		slog.Warn("skipping file", "path", entryPath, "error", err)

		return nil
	}

	relPath, err := filepath.Rel(rootPath, entryPath)
	if err != nil {
		return eris.Wrapf(err, "failed to compute relative path for %s", entryPath)
	}

	if !filter.IsIncluded(relPath, rules) {
		slog.Debug("excluding by filter rule", "path", relPath)

		return nil
	}

	if info.IsDir() {
		return processDir(node, entry, entryPath, rootPath, rules)
	}

	if info.Mode().IsRegular() || isSymlink(entry) {
		processFile(node, entry, info, entryPath)
	}

	return nil
}

func processDir(node *model.Directory, entry os.DirEntry, entryPath, rootPath string, rules []filter.Rule) error {
	if isSymlink(entry) {
		slog.Debug("skipping directory symlink", "path", entryPath)

		return nil
	}

	child, err := scanDir(entryPath, rootPath, rules)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping directory: permission denied", "path", entryPath)

			return nil
		}

		return err
	}

	// Prune empty directories
	if len(child.Files) > 0 || len(child.Dirs) > 0 {
		node.Dirs = append(node.Dirs, child)
	}

	return nil
}

func processFile(node *model.Directory, entry os.DirEntry, info os.FileInfo, entryPath string) {
	ext := strings.TrimPrefix(filepath.Ext(entry.Name()), ".")

	fileType := ext
	if fileType == "" {
		fileType = "no-extension"
	}

	f := &model.File{
		Path:      entryPath,
		Name:      entry.Name(),
		Extension: ext,
	}

	f.SetQuantity(filesystem.FileSize, info.Size())
	f.SetClassification(filesystem.FileType, fileType)

	node.Files = append(node.Files, f)
}

func isSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}

func countFiles(node *model.Directory) int {
	count := len(node.Files)
	for _, d := range node.Dirs {
		count += countFiles(d)
	}

	return count
}

// FilterBinaryFiles returns a copy of the directory tree with binary files removed.
// Directories that become empty after removal are also pruned.
func FilterBinaryFiles(node *model.Directory) *model.Directory {
	result := &model.Directory{
		Path: node.Path,
		Name: node.Name,
	}

	for _, f := range node.Files {
		if f.IsBinary {
			slog.Debug("excluding binary file", "path", f.Path)

			continue
		}

		result.Files = append(result.Files, f)
	}

	for _, d := range node.Dirs {
		filtered := FilterBinaryFiles(d)
		if len(filtered.Files) > 0 || len(filtered.Dirs) > 0 {
			result.Dirs = append(result.Dirs, filtered)
		}
	}

	return result
}
