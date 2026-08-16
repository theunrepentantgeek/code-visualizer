package git

import (
	"path/filepath"
	"slices"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

const (
	FileAge           metric.Name = "file-age"
	FileFreshness     metric.Name = "file-freshness"
	AuthorCount       metric.Name = "author-count"
	CommitCount       metric.Name = "commit-count"
	TotalLinesAdded   metric.Name = "total-lines-added"
	TotalLinesRemoved metric.Name = "total-lines-removed"
	CommitDensity     metric.Name = "commit-density"

	// LinesAdded is a commit-level metric tracking lines added per commit.
	LinesAdded   metric.Name = "lines-added"
	LinesRemoved metric.Name = "lines-removed"
	LinesChanged metric.Name = "lines-changed"

	// Authorship metrics (issue #550).

	// InitialDeveloperMetric (Classification) is the greatest-weight author
	// within the early window (first earlyWindowFraction of the node's life).
	InitialDeveloperMetric metric.Name = "initial-developer"

	// CurrentMaintainerMetric (Classification) is the greatest-weight author
	// within recentWindowDays of HEAD; Unmaintained if none.
	CurrentMaintainerMetric metric.Name = "current-maintainer"

	// CodeOwnerMetric (Classification) is the greatest lifetime-weight author.
	CodeOwnerMetric metric.Name = "code-owner"

	// SignificantContributorCountMetric (Quantity) counts authors with
	// share ≥ significantShareThreshold.
	SignificantContributorCountMetric metric.Name = "significant-contributor-count"

	// BusFactorMetric (Quantity) is the smallest number of top authors whose
	// combined share reaches busFactorThreshold.
	BusFactorMetric metric.Name = "bus-factor"

	// OwnershipDominanceMetric (Measure 0–1) is the maximum per-author share.
	OwnershipDominanceMetric metric.Name = "ownership-dominance"

	// ContributorEntropyMetric (Measure 0–1) is normalised Shannon entropy of
	// per-author shares; 0 = single owner, →1 = evenly shared.
	ContributorEntropyMetric metric.Name = "contributor-entropy"

	// OrphanRiskMetric (Measure 0–1) is the summed share of authors not active
	// repo-wide within activityWindowDays of HEAD.
	OrphanRiskMetric metric.Name = "orphan-risk"

	// KnowledgeHandoffMetric (Measure 0–1) is the share of recent-window
	// contribution from authors absent in the early window.
	KnowledgeHandoffMetric metric.Name = "knowledge-handoff"
)

var fileMetricNames = []metric.Name{
	FileAge,
	FileFreshness,
	AuthorCount,
	CommitCount,
	TotalLinesAdded,
	TotalLinesRemoved,
	CommitDensity,
}

// authorshipMetricNames lists the nine authorship metrics in the order they are
// registered and loaded.  This slice is kept separate from fileMetricNames
// because the two families use different loading paths.
var authorshipMetricNames = []metric.Name{
	InitialDeveloperMetric,
	CurrentMaintainerMetric,
	CodeOwnerMetric,
	SignificantContributorCountMetric,
	BusFactorMetric,
	OwnershipDominanceMetric,
	ContributorEntropyMetric,
	OrphanRiskMetric,
	KnowledgeHandoffMetric,
}

// IsAuthorshipMetric reports whether name belongs to the authorship metric
// family, whose configured loader is run by the shared pipeline stage.
func IsAuthorshipMetric(name metric.Name) bool {
	return slices.Contains(authorshipMetricNames, name)
}

// IsGitMetric reports whether name is a metric that requires a git repository.
func IsGitMetric(name metric.Name) bool {
	return slices.Contains(fileMetricNames, name) || IsAuthorshipMetric(name)
}

// buildRelPathSet returns the set of relative paths (relative to the git
// worktree root) for all files under root.
func buildRelPathSet(s *repoService, root *model.Directory) map[string]bool {
	paths := make(map[string]bool)

	model.WalkFiles(root, func(f *model.File) {
		relPath, err := repoRelativePath(s.RepoRoot(), f.Path)
		if err == nil {
			paths[relPath] = true
		}
	})

	return paths
}

func repoRelativePath(repoRoot, path string) (string, error) {
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "", eris.Wrap(err, "failed to compute repository-relative path")
	}

	return filepath.ToSlash(relPath), nil
}
