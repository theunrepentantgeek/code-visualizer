// Package scan provides recursive directory scanning with symlink handling.
package scan

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/source"
)

// Progress receives notifications as directories are scanned.
type Progress interface {
	// OnDirectoryScanned is called after each directory is fully processed.
	// fileCount is the number of direct (non-recursive) files in that directory.
	OnDirectoryScanned(path string, fileCount int)
}

// ScanTree scans a read-only content source.
func ScanTree(
	ctx context.Context,
	tree source.Tree,
	rules []filter.Rule,
	progress Progress,
	includeBinary bool,
) (*model.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, eris.Wrap(err, "source scan cancelled")
	}

	root, err := newFSWalker(ctx, tree, rules, progress, includeBinary).scanDir(".")
	if err != nil {
		return nil, err
	}

	if !hasFiles(root) {
		return nil, errors.New("no files found in directory")
	}

	return root, nil
}

// Scan recursively scans the directory at path and returns a model.Directory tree.
// File symlinks are followed; directory symlinks are skipped.
// Permission-denied errors are logged and scanning continues.
// When includeBinary is false, binary files are excluded during the scan rather
// than being added to the tree and filtered later.
// Returns an error if the directory contains no files.
func Scan(
	ctx context.Context,
	path string,
	rules []filter.Rule,
	progress Progress,
	includeBinary bool,
) (*model.Directory, error) {
	if err := ctx.Err(); err != nil {
		return nil, eris.Wrap(err, "directory scan cancelled")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve absolute path")
	}

	root, err := newWalker(ctx, absPath, rules, progress, includeBinary).scanDir(absPath)
	if err != nil {
		return nil, err
	}

	if !hasFiles(root) {
		return nil, errors.New("no files found in directory")
	}

	return root, nil
}
