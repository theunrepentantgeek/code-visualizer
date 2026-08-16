package git

import (
	"cmp"
	"math"
	"slices"
	"time"
)

// Sentinel classification values used by identity metrics.
const (
	// Unmaintained is the classification returned by CurrentMaintainer (and
	// other identity metrics) when no qualifying author is found.
	Unmaintained = "«unmaintained»"

	// OtherContributor is the classification assigned to contributors ranked
	// beyond identity-top-k in the global contribution ranking.
	OtherContributor = "«other»"
)

// AuthorshipParams holds the configurable thresholds for the nine authorship
// metrics.  All fields have spec-mandated defaults supplied by
// DefaultAuthorshipParams.
type AuthorshipParams struct {
	// ActivityWindowDays is the look-back window for orphan-risk: an author is
	// "still active" if they have committed anywhere in the repo within this
	// many days of HEAD. Default: 180.
	ActivityWindowDays int
	// RecentWindowDays is the look-back window for current-maintainer.
	// Default: 180.
	RecentWindowDays int
	// EarlyWindowFraction is the fraction of a node's calendar lifetime that
	// defines the "early window" used by initial-developer and
	// knowledge-handoff. Default: 0.25.
	EarlyWindowFraction float64
	// SignificantShareThreshold is the minimum Sₐ = Wₐ/W an author must hold
	// to count toward significant-contributor-count. Default: 0.10.
	SignificantShareThreshold float64
	// BusFactorThreshold is the combined-share target for bus-factor.
	// Default: 0.50.
	BusFactorThreshold float64
	// IdentityTopK is the number of top contributors (by global weight) that
	// receive distinct colours in identity-metric legends; contributors beyond
	// this rank are bucketed into OtherContributor. Default: 11.
	IdentityTopK int
	// HonorMailmap, when true, normalises author email and name through the
	// repository's .mailmap file before aggregating contributions. Default: true.
	HonorMailmap bool
}

// DefaultAuthorshipParams returns AuthorshipParams with the defaults from the
// issue #550 spec.
func DefaultAuthorshipParams() AuthorshipParams {
	return AuthorshipParams{
		ActivityWindowDays:        180,
		RecentWindowDays:          180,
		EarlyWindowFraction:       0.25,
		SignificantShareThreshold: 0.10,
		BusFactorThreshold:        0.50,
		IdentityTopK:              11,
		HonorMailmap:              true,
	}
}

// authorShare is a per-author precomputed weight/share tuple used for
// deterministic sorting: weight desc → earlier first contribution → lex email.
type authorShare struct {
	email  string
	weight int64
	share  float64
	first  time.Time // earliest contribution for tie-breaking
}

// sortedShares computes per-author lifetime shares and returns them in
// deterministic descending order.
func sortedShares(records []AuthorRecord) ([]authorShare, int64) {
	total := int64(0)
	for _, r := range records {
		total += r.Added + r.Removed
	}

	if total == 0 {
		return nil, 0
	}

	shares := make([]authorShare, len(records))
	for i, r := range records {
		w := r.Added + r.Removed
		shares[i] = authorShare{
			email:  r.Email,
			weight: w,
			share:  float64(w) / float64(total),
			first:  r.FirstSeen,
		}
	}

	sortAuthorShares(shares)

	return shares, total
}

// sortAuthorShares sorts in place: weight desc → earlier first → lex email.
func sortAuthorShares(shares []authorShare) {
	slices.SortStableFunc(shares, func(a, b authorShare) int {
		if c := cmp.Compare(b.weight, a.weight); c != 0 {
			return c
		}

		if !a.first.Equal(b.first) {
			if a.first.Before(b.first) {
				return -1
			}

			return 1
		}

		return cmp.Compare(a.email, b.email)
	})
}

// nodeSpan returns the oldest and newest commit dates across all records for a
// node (file or merged directory subtree).
func nodeSpan(records []AuthorRecord) (oldest, newest time.Time) {
	for _, r := range records {
		if oldest.IsZero() || r.FirstSeen.Before(oldest) {
			oldest = r.FirstSeen
		}

		if newest.IsZero() || r.LastSeen.After(newest) {
			newest = r.LastSeen
		}
	}

	return oldest, newest
}

// earlyWindowCutoff returns the timestamp that separates the early window from
// the rest of a node's lifetime.
func earlyWindowCutoff(oldest, newest time.Time, fraction float64) time.Time {
	if oldest.IsZero() {
		return time.Time{}
	}

	lifetime := newest.Sub(oldest)

	return oldest.Add(time.Duration(float64(lifetime) * fraction))
}

// windowedShares computes sorted authorShares from contributions that fall
// no earlier than from and no later than to. A zero from means "no lower
// bound"; a zero to means "no upper bound".
func windowedShares(records []AuthorRecord, from, to time.Time) []authorShare {
	totals := make(windowedAccums, len(records))

	for _, r := range records {
		for _, cp := range r.Contributions {
			if !contributionInWindow(cp, from, to) {
				continue
			}

			totals.add(r.Email, cp)
		}
	}

	total := int64(0)
	for _, a := range totals {
		total += a.weight
	}

	if total == 0 {
		return nil
	}

	shares := make([]authorShare, 0, len(totals))
	for email, a := range totals {
		shares = append(shares, authorShare{
			email:  email,
			weight: a.weight,
			share:  float64(a.weight) / float64(total),
			first:  a.first,
		})
	}

	sortAuthorShares(shares)

	return shares
}

type windowedAccum struct {
	weight int64
	first  time.Time
}

type windowedAccums map[string]*windowedAccum

func contributionInWindow(point ContributionPoint, from, to time.Time) bool {
	return (from.IsZero() || !point.When.Before(from)) &&
		(to.IsZero() || !point.When.After(to))
}

func (totals windowedAccums) add(email string, point ContributionPoint) {
	accumulator := totals[email]
	if accumulator == nil {
		accumulator = &windowedAccum{}
		totals[email] = accumulator
	}

	accumulator.weight += point.Added + point.Removed
	if accumulator.first.IsZero() || point.When.Before(accumulator.first) {
		accumulator.first = point.When
	}
}

// ─── Nine metric computations ────────────────────────────────────────────────

// codeOwner returns the greatest lifetime-weight author's email.
// Returns Unmaintained if all weights are zero.
func codeOwner(records []AuthorRecord) string {
	shares, _ := sortedShares(records)
	if len(shares) == 0 || shares[0].weight == 0 {
		return Unmaintained
	}

	return shares[0].email
}

// initialDeveloper returns the greatest-weight author within the early window
// (the first earlyFraction of the node's calendar lifetime).
func initialDeveloper(records []AuthorRecord, earlyFraction float64) string {
	oldest, newest := nodeSpan(records)
	cutoff := earlyWindowCutoff(oldest, newest, earlyFraction)

	// The early window starts at time-zero (no lower bound).
	shares := windowedShares(records, time.Time{}, cutoff)
	if len(shares) == 0 {
		return Unmaintained
	}

	return shares[0].email
}

// currentMaintainer returns the greatest-weight author within the recent window
// (the recentWindowDays days before headDate).
func currentMaintainer(records []AuthorRecord, headDate time.Time, recentWindowDays int) string {
	recentCutoff := headDate.AddDate(0, 0, -recentWindowDays)
	shares := windowedShares(records, recentCutoff, time.Time{})

	if len(shares) == 0 {
		return Unmaintained
	}

	return shares[0].email
}

// significantContributorCount returns the count of authors with share ≥ threshold.
func significantContributorCount(records []AuthorRecord, threshold float64) int64 {
	shares, _ := sortedShares(records)

	count := int64(0)

	for _, s := range shares {
		if s.share >= threshold {
			count++
		}
	}

	return count
}

// busFactor returns the smallest number of top authors whose combined share
// reaches or exceeds threshold.  Returns 0 if there are no contributions.
func busFactor(records []AuthorRecord, threshold float64) int64 {
	shares, _ := sortedShares(records)

	if len(shares) == 0 {
		return 0
	}

	cumulative := 0.0
	for i, s := range shares {
		cumulative += s.share
		if cumulative >= threshold {
			return int64(i + 1)
		}
	}

	return int64(len(shares))
}

// ownershipDominance returns the maximum share max(Sₐ).
func ownershipDominance(records []AuthorRecord) float64 {
	shares, _ := sortedShares(records)
	if len(shares) == 0 {
		return 0
	}

	return shares[0].share
}

// contributorEntropy returns the normalised Shannon entropy H/log(n) in [0,1].
// Returns 0 for a single author or no contributions.
func contributorEntropy(records []AuthorRecord) float64 {
	shares, _ := sortedShares(records)

	n := len(shares)
	if n <= 1 {
		return 0
	}

	h := 0.0

	for _, s := range shares {
		if s.share > 0 {
			h -= s.share * math.Log(s.share)
		}
	}

	return h / math.Log(float64(n))
}

// orphanRisk returns the summed share of authors whose most-recent repo-wide
// commit was more than activityWindowDays before headDate.
func orphanRisk(
	records []AuthorRecord,
	lastActive map[string]time.Time,
	headDate time.Time,
	activityWindowDays int,
) float64 {
	shares, total := sortedShares(records)
	if total == 0 {
		return 0
	}

	activeCutoff := headDate.AddDate(0, 0, -activityWindowDays)

	orphanWeight := int64(0)

	for _, s := range shares {
		last, seen := lastActive[s.email]
		if !seen || last.Before(activeCutoff) {
			orphanWeight += s.weight
		}
	}

	return float64(orphanWeight) / float64(total)
}

// knowledgeHandoff returns the share of recent-window contribution from authors
// absent in the early window.  Returns 0 if there is no recent contribution or
// no meaningful early/recent split.
func knowledgeHandoff(
	records []AuthorRecord,
	headDate time.Time,
	recentWindowDays int,
	earlyFraction float64,
) float64 {
	oldest, newest := nodeSpan(records)
	cutoff := earlyWindowCutoff(oldest, newest, earlyFraction)

	recentFrom := headDate.AddDate(0, 0, -recentWindowDays)

	// No meaningful split when the early window strictly overlaps the recent
	// window (cutoff > recentFrom). When they are exactly equal, the boundary
	// is shared and contributions at that instant fall into both windows, which
	// is a valid computable state — do not short-circuit.
	if cutoff.After(recentFrom) {
		return 0
	}

	earlyAuthors := earlyAuthors(records, cutoff)
	recentTotal, newWeight := recentContributionWeights(records, recentFrom, earlyAuthors)

	if recentTotal == 0 {
		return 0
	}

	return float64(newWeight) / float64(recentTotal)
}

func earlyAuthors(records []AuthorRecord, cutoff time.Time) map[string]bool {
	authors := make(map[string]bool, len(records))
	for _, record := range records {
		for _, point := range record.Contributions {
			if point.When.After(cutoff) {
				continue
			}
			if point.Added+point.Removed == 0 {
				continue
			}

			authors[record.Email] = true
			break
		}
	}

	return authors
}

func recentContributionWeights(
	records []AuthorRecord,
	recentFrom time.Time,
	earlyAuthors map[string]bool,
) (recentTotal, newWeight int64) {
	for _, record := range records {
		for _, point := range record.Contributions {
			if point.When.Before(recentFrom) {
				continue
			}

			weight := point.Added + point.Removed

			recentTotal += weight
			if !earlyAuthors[record.Email] {
				newWeight += weight
			}
		}
	}

	return recentTotal, newWeight
}
