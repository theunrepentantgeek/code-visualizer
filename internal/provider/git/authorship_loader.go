package git

import (
	"cmp"
	"log/slog"
	"slices"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// authorshipLoader is the metric loader for the nine authorship metrics.
// It calls BulkAuthorHistory once and then applies computed values to every
// file and directory node in the tree.
type authorshipLoader struct {
	params AuthorshipParams
}

// LoadAuthorshipMetrics applies authorship metrics using params. It is used by
// the pipeline so configuration can be passed without mutable global state.
func LoadAuthorshipMetrics(root *model.Directory, params AuthorshipParams) error {
	return (&authorshipLoader{params: params}).Load(root, authorshipMetricNames)
}

// Load satisfies provider.LoadFunc.  requested must be a subset of
// authorshipMetricNames; names not in that set are silently skipped.
func (al *authorshipLoader) Load(root *model.Directory, _ []metric.Name) error {
	s, err := getService(root.Path)
	if err != nil {
		return eris.Wrap(err, "authorship loader requires a git repository")
	}

	repoRoot := s.RepoRoot()
	pathSet := buildRelPathSet(s, root)

	result, err := BulkAuthorHistory(repoRoot, pathSet, al.params.HonorMailmap, nil)
	if err != nil {
		return eris.Wrap(err, "authorship loader failed to walk git history")
	}

	// Apply to every file.
	model.WalkFiles(root, func(f *model.File) {
		relPath, relErr := repoRelativePath(repoRoot, f.Path)
		if relErr != nil {
			slog.Warn("authorship loader: could not compute relative path",
				"path", f.Path, "error", relErr)

			return
		}

		records, ok := result.ByFile[relPath]
		if !ok {
			return
		}

		applyAuthorshipToNode(records, result, al.params, f)
	})

	// Apply to every directory: recompute from the flat union of subtree source
	// records (not from child metric values), as mandated by the issue spec.
	model.WalkDirectories(root, func(d *model.Directory) {
		records := collectSubtreeRecords(d, result.ByFile, repoRoot)
		if len(records) == 0 {
			return
		}

		applyAuthorshipToNode(records, result, al.params, d)
	})

	// Bucket identity metrics: replace contributors ranked beyond IdentityTopK
	// in the global weight ranking with the OtherContributor sentinel so that
	// colour legends have at most IdentityTopK+2 entries.
	if al.params.IdentityTopK > 0 {
		bucketIdentityMetrics(root, result.ByFile, al.params.IdentityTopK)
	}

	return nil
}

// metricNode is satisfied by both *model.File and *model.Directory because
// both embed model.MetricContainer.
type metricNode interface {
	SetQuantity(name metric.Name, value int64)
	SetMeasure(name metric.Name, value float64)
	SetClassification(name metric.Name, value string)
}

// applyAuthorshipToNode computes and stores all nine authorship metrics on node.
func applyAuthorshipToNode(
	records []AuthorRecord,
	result AuthorHistoryResult,
	params AuthorshipParams,
	node metricNode,
) {
	node.SetClassification(CodeOwnerMetric, codeOwner(records))
	node.SetClassification(InitialDeveloperMetric, initialDeveloper(records, params.EarlyWindowFraction))
	node.SetClassification(CurrentMaintainerMetric,
		currentMaintainer(records, result.HeadDate, params.RecentWindowDays))

	node.SetQuantity(SignificantContributorCountMetric, significantContributorCount(records, params.SignificantShareThreshold))
	node.SetQuantity(BusFactorMetric, busFactor(records, params.BusFactorThreshold))

	node.SetMeasure(OwnershipDominanceMetric, ownershipDominance(records))
	node.SetMeasure(ContributorEntropyMetric, contributorEntropy(records))
	node.SetMeasure(OrphanRiskMetric, orphanRisk(records, result.LastActive, result.HeadDate, params.ActivityWindowDays))
	node.SetMeasure(KnowledgeHandoffMetric, knowledgeHandoff(records, result.HeadDate, params.RecentWindowDays, params.EarlyWindowFraction))
}

// subtreeAccum accumulates per-author data across files in a directory subtree.
type subtreeAccum struct {
	name          string
	added         int64
	removed       int64
	firstSeen     time.Time
	lastSeen      time.Time
	contributions []ContributionPoint
}

func (a *subtreeAccum) merge(r AuthorRecord) {
	a.added += r.Added
	a.removed += r.Removed
	a.contributions = append(a.contributions, r.Contributions...)

	if a.firstSeen.IsZero() || r.FirstSeen.Before(a.firstSeen) {
		a.firstSeen = r.FirstSeen
	}

	if a.lastSeen.IsZero() || r.LastSeen.After(a.lastSeen) {
		a.lastSeen = r.LastSeen
	}
}

// collectSubtreeRecords merges AuthorRecords from every file in dir's subtree
// into a flat per-author slice.  This implements the "grounded in source data"
// directory computation mandated by the issue #550 spec: directory Wₐ equals
// the sum of an author's weights across the whole subtree, not an
// average-of-child-metric-values.
func collectSubtreeRecords(
	dir *model.Directory,
	byFile FileAuthorRecords,
	repoRoot string,
) []AuthorRecord {
	merged := make(map[string]*subtreeAccum)

	model.WalkFiles(dir, func(f *model.File) {
		relPath, err := repoRelativePath(repoRoot, f.Path)
		if err != nil {
			return
		}

		records, ok := byFile[relPath]
		if !ok {
			return
		}

		for _, r := range records {
			a, exists := merged[r.Email]
			if !exists {
				a = &subtreeAccum{name: r.Name}
				merged[r.Email] = a
			}

			a.merge(r)
		}
	})

	if len(merged) == 0 {
		return nil
	}

	result := make([]AuthorRecord, 0, len(merged))
	for email, a := range merged {
		result = append(result, AuthorRecord{
			Email:         email,
			Name:          a.name,
			Added:         a.added,
			Removed:       a.removed,
			FirstSeen:     a.firstSeen,
			LastSeen:      a.lastSeen,
			Contributions: a.contributions,
		})
	}

	return result
}

// globalEmailRanking returns a set of the top-K author emails ranked by total
// contribution weight (Added+Removed) across all files in byFile.
func globalEmailRanking(byFile FileAuthorRecords, topK int) map[string]bool {
	weights := make(map[string]int64)

	for _, records := range byFile {
		for _, r := range records {
			weights[r.Email] += r.Added + r.Removed
		}
	}

	emails := make([]string, 0, len(weights))
	for email := range weights {
		emails = append(emails, email)
	}

	slices.SortStableFunc(emails, func(a, b string) int {
		if c := cmp.Compare(weights[b], weights[a]); c != 0 {
			return c
		}

		return cmp.Compare(a, b)
	})

	top := make(map[string]bool, topK)
	for i, email := range emails {
		if i >= topK {
			break
		}

		top[email] = true
	}

	return top
}

// identityMetricNames lists the three Classification metrics that use author
// email as their value and need top-K bucketing.
var identityMetricNames = []metric.Name{
	CodeOwnerMetric,
	InitialDeveloperMetric,
	CurrentMaintainerMetric,
}

// bucketIdentityMetrics replaces classification values for identity metrics
// that are not in the global top-K email set with OtherContributor.  This caps
// the legend to at most topK+2 distinct colours (top-K + «other» +
// «unmaintained»).
func bucketIdentityMetrics(root *model.Directory, byFile FileAuthorRecords, topK int) {
	top := globalEmailRanking(byFile, topK)

	replace := func(val string) string {
		if val == Unmaintained || top[val] {
			return val
		}

		return OtherContributor
	}

	model.WalkFiles(root, func(f *model.File) {
		for _, m := range identityMetricNames {
			if val, ok := f.Classification(m); ok {
				f.SetClassification(m, replace(val))
			}
		}
	})

	model.WalkDirectories(root, func(d *model.Directory) {
		for _, m := range identityMetricNames {
			if val, ok := d.Classification(m); ok {
				d.SetClassification(m, replace(val))
			}
		}
	})
}
