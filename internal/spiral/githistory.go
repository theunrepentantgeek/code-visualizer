package spiral

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/git"
)

// CommitRecord represents a single commit touching a file.
type CommitRecord struct {
	FilePath  string
	Timestamp time.Time
	File      *model.File
}

// LoadCommitHistory walks the model tree and returns one CommitRecord per
// file-commit pair. It uses the git provider to fetch commit timestamps
// for every file in the tree.
func LoadCommitHistory(root *model.Directory) ([]CommitRecord, error) {
	repoRoot, err := git.RepoRootFor(root.Path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve git root")
	}

	var records []CommitRecord

	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := filepath.Rel(repoRoot, f.Path)
		if relErr != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", relErr)

			return
		}

		timestamps, tsErr := git.FileCommitTimestamps(root.Path, relPath)
		if tsErr != nil {
			slog.Debug("could not get commit timestamps", "path", relPath, "error", tsErr)

			return
		}

		for _, ts := range timestamps {
			records = append(records, CommitRecord{
				FilePath:  f.Path,
				Timestamp: ts,
				File:      f,
			})
		}
	})

	return records, nil
}
