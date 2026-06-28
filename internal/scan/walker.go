package scan

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

type walker struct {
	policy        filterPolicy
	builder       nodeBuilder
	progress      Progress
	includeBinary bool
}

func newWalker(rootPath string, rules []filter.Rule, progress Progress, includeBinary bool) walker {
	return walker{
		policy:        newFilterPolicy(rootPath, rules),
		builder:       newNodeBuilder(IsBinaryFile, includeBinary),
		progress:      progress,
		includeBinary: includeBinary,
	}
}

func (w walker) scanDir(dirPath string) (*model.Directory, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to read directory %s", dirPath)
	}

	node := &model.Directory{
		Path:  dirPath,
		Name:  filepath.Base(dirPath),
		Files: make([]*model.File, 0, len(entries)),
		Dirs:  make([]*model.Directory, 0, len(entries)),
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())

		if err := w.processEntry(node, entry, entryPath); err != nil {
			return nil, err
		}
	}

	if w.progress != nil {
		w.progress.OnDirectoryScanned(dirPath, len(node.Files))
	}

	return node, nil
}

func (w walker) processEntry(node *model.Directory, entry os.DirEntry, entryPath string) error {
	included, relPath, err := w.policy.includes(entryPath)
	if err != nil {
		return err
	}

	if !included {
		slog.Debug("excluding by filter rule", "path", relPath)

		return nil
	}

	if isSymlink(entry) {
		return w.processSymlink(node, entry, entryPath)
	}

	if entry.Type().IsDir() {
		return w.processDir(node, entry, entryPath)
	}

	if entry.Type().IsRegular() {
		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				slog.Warn("skipping file: permission denied", "path", entryPath)

				return nil
			}

			slog.Warn("skipping file", "path", entryPath, "error", err)

			return nil
		}

		w.builder.processFile(node, entry, info, entryPath)
	}

	return nil
}

func (w walker) processSymlink(node *model.Directory, entry os.DirEntry, entryPath string) error {
	info, err := os.Stat(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping file: permission denied", "path", entryPath)

			return nil
		}

		slog.Warn("skipping file", "path", entryPath, "error", err)

		return nil
	}

	if info.IsDir() {
		return w.processDir(node, entry, entryPath)
	}

	if info.Mode().IsRegular() {
		w.builder.processFile(node, entry, info, entryPath)
	}

	return nil
}

func (w walker) processDir(node *model.Directory, entry os.DirEntry, entryPath string) error {
	if isSymlink(entry) {
		slog.Debug("skipping directory symlink", "path", entryPath)

		return nil
	}

	child, err := w.scanDir(entryPath)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			slog.Warn("skipping directory: permission denied", "path", entryPath)

			return nil
		}

		return err
	}

	if len(child.Files) > 0 || len(child.Dirs) > 0 {
		node.Dirs = append(node.Dirs, child)
	}

	return nil
}

func isSymlink(entry os.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
}
