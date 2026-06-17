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
//
// Git metrics have no silent fallback: if the repository cannot be opened, has
// no history, or contains none of the scanned files, walkGitFilesAll returns
// an error rather than producing an empty result that would cascade into
// confusing downstream failures.
func walkGitFilesAll(root *model.Directory) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrapf(err, "git loader requires a git repository")
	}

	pathSet := buildRelPathSet(s, root)
	if err := s.bulkPrewarm(pathSet); err != nil {
		return eris.Wrapf(err, "git loader requires readable git history at %s", s.RepoRoot())
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

	if !anyFileHasGitMetric(root) {
		return eris.Errorf(
			"git loader produced no metrics: none of the scanned files under %s have git history",
			s.RepoRoot(),
		)
	}

	return nil
}

// anyFileHasGitMetric reports whether at least one file in the tree has the
// FileAge metric set. FileAge is populated for every file with non-empty
// commit history, so it serves as a sentinel for "git produced data".
func anyFileHasGitMetric(root *model.Directory) bool {
	var found bool

	model.WalkFiles(root, func(f *model.File) {
		if found {
			return
		}

		if _, ok := f.Quantity(FileAge); ok {
			found = true
		}
	})

	return found
}
