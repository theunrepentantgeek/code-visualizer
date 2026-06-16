package git

import (
	"log/slog"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// loadAllFileMetrics runs the git analysis once and populates all 7 file-level
// git metrics in a single pass. This replaces 7 separate legacy providers that
// each independently walked git history.
func loadAllFileMetrics(root *model.Directory) error {
	return walkGitFilesAll(root)
}

// walkGitFilesAll opens the repo service, walks all files, and invokes every
// providerDef's process function for each file. This populates all git metrics
// in a single walk rather than one walk per metric.
func walkGitFilesAll(root *model.Directory) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrapf(err, "git loader requires a git repository")
	}

	pathSet := buildRelPathSet(s, root)
	if prewarmErr := s.bulkPrewarm(pathSet); prewarmErr != nil {
		slog.Warn("git bulk prewarm failed, falling back to per-file lookups",
			"metric", "git-all", "error", prewarmErr)
	}

	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := filepath.Rel(s.RepoRoot(), f.Path)
		if relErr != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", relErr)

			return
		}

		for _, def := range providerDefs {
			def.process(s, f, relPath)
		}
	})

	return nil
}
