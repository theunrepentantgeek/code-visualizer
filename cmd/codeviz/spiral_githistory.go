package main

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/rotisserie/eris"

	"github.com/bevan/code-visualizer/internal/model"
	"github.com/bevan/code-visualizer/internal/provider/git"
)

// commitRecord represents a single commit touching a file.
type commitRecord struct {
	FilePath  string
	Timestamp time.Time
	File      *model.File
}

// loadCommitHistory walks the entire commit graph once and returns one
// commitRecord per file-commit pair. It uses a bulk tree-diff approach that is
// dramatically faster than per-file log queries.
// The optional onCommitProcessed callback is invoked after each commit is examined.
func loadCommitHistory(root *model.Directory, onCommitProcessed func()) ([]commitRecord, error) {
	repoRoot, err := git.RepoRootFor(root.Path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to resolve git root")
	}

	// Build index: slash-separated relative path -> *model.File
	filesByPath := make(map[string]*model.File)
	pathSet := make(map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := filepath.Rel(repoRoot, f.Path)
		if relErr != nil {
			slog.Warn("could not compute relative path", "path", f.Path, "error", relErr)

			return
		}

		key := filepath.ToSlash(relPath)
		filesByPath[key] = f
		pathSet[key] = true
	})

	// Single-pass bulk extraction
	history, err := git.BulkFileHistory(root.Path, pathSet, onCommitProcessed)
	if err != nil {
		return nil, eris.Wrap(err, "failed to load bulk commit history")
	}

	// Convert to records
	var records []commitRecord

	for path, timestamps := range history {
		f, ok := filesByPath[path]
		if !ok {
			continue
		}

		for _, ts := range timestamps {
			records = append(records, commitRecord{
				FilePath:  f.Path,
				Timestamp: ts,
				File:      f,
			})
		}
	}

	return records, nil
}
