package stages

import (
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// FilterChangedOnly limits the scanned tree to current files modified in the
// selected Git history range.
func FilterChangedOnly(c *CommonState) error {
	if c.Flags == nil || !c.Flags.ChangedOnly {
		return nil
	}

	historyRange := c.Flags.HistoryRange
	if strings.TrimSpace(historyRange.From) == "" && strings.TrimSpace(historyRange.Until) == "" {
		return eris.New("--changed-only requires --from or --until")
	}

	targetPath := c.TargetPath
	if targetPath == "" && c.Root != nil {
		targetPath = c.Root.Path
	}

	if err := CheckGitRepoHelper(targetPath); err != nil {
		return err
	}

	repoRoot, err := git.RepoRootFor(targetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve git root")
	}

	currentPaths := buildTrackedPathSet(c.Root, repoRoot)

	changedPaths, err := git.ChangedPathsInHistoryRange(repoRoot, currentPaths, historyRange)
	if err != nil {
		return eris.Wrap(err, "failed to filter files by git range")
	}

	pruneTreeToPaths(c.Root, repoRoot, changedPaths)
	if model.CountFiles(c.Root) == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterChangedOnlyMsg}
	}

	return nil
}

func pruneTreeToPaths(root *model.Directory, repoRoot string, included map[string]bool) {
	if root == nil {
		return
	}

	root.Files = keepIncludedFiles(root.Files, repoRoot, included)

	dirs := root.Dirs[:0]
	for _, child := range root.Dirs {
		pruneTreeToPaths(child, repoRoot, included)
		if child != nil && (len(child.Files) > 0 || len(child.Dirs) > 0) {
			dirs = append(dirs, child)
		}
	}

	root.Dirs = dirs
	root.DirectFileCount = len(root.Files)
	root.AllFileCount = root.DirectFileCount
	root.AllDirCount = len(root.Dirs)

	for _, child := range root.Dirs {
		root.AllFileCount += child.AllFileCount
		root.AllDirCount += child.AllDirCount
	}
}

func keepIncludedFiles(
	files []*model.File,
	repoRoot string,
	included map[string]bool,
) []*model.File {
	result := files[:0]

	for _, file := range files {
		rel, err := filepath.Rel(repoRoot, file.Path)
		if err == nil && included[filepath.ToSlash(rel)] {
			result = append(result, file)
		}
	}

	return result
}
