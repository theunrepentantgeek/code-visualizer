package config

// defaultAuthorshipActivityWindowDays is the default look-back window for
// orphan-risk (days from HEAD an author must have committed to be "active").
const defaultAuthorshipActivityWindowDays = 180

// defaultAuthorshipRecentWindowDays is the default look-back window for
// current-maintainer (days from HEAD).
const defaultAuthorshipRecentWindowDays = 180

// defaultAuthorshipEarlyWindowFraction is the default fraction of a node's
// calendar lifetime that defines the "early" contribution window.
const defaultAuthorshipEarlyWindowFraction = 0.25

// defaultAuthorshipSignificantShareThreshold is the minimum author share
// required to count toward significant-contributor-count.
const defaultAuthorshipSignificantShareThreshold = 0.10

// defaultAuthorshipBusFactorThreshold is the cumulative-share target for
// bus-factor calculation.
const defaultAuthorshipBusFactorThreshold = 0.50

// defaultAuthorshipIdentityTopK is the number of top contributors that receive
// distinct legend colours; contributors beyond this rank become «other».
const defaultAuthorshipIdentityTopK = 11

// defaultAuthorshipHonorMailmap controls whether .mailmap normalisation is
// applied to author identities before aggregating contributions.
const defaultAuthorshipHonorMailmap = true

// DefaultAuthorshipConfig returns an *AuthorshipConfig with every field set to
// the spec-mandated default value.  config.New() embeds this so that
// --export-config always emits an authorship section, making all thresholds
// discoverable without any git://issue#550 documentation lookup.
func DefaultAuthorshipConfig() *AuthorshipConfig {
	activityWindow := defaultAuthorshipActivityWindowDays
	recentWindow := defaultAuthorshipRecentWindowDays
	earlyFraction := defaultAuthorshipEarlyWindowFraction
	sigThreshold := defaultAuthorshipSignificantShareThreshold
	busThreshold := defaultAuthorshipBusFactorThreshold
	topK := defaultAuthorshipIdentityTopK
	honorMailmap := defaultAuthorshipHonorMailmap

	return &AuthorshipConfig{
		ActivityWindowDays:        &activityWindow,
		RecentWindowDays:          &recentWindow,
		EarlyWindowFraction:       &earlyFraction,
		SignificantShareThreshold: &sigThreshold,
		BusFactorThreshold:        &busThreshold,
		IdentityTopK:              &topK,
		HonorMailmap:              &honorMailmap,
	}
}

// AuthorshipConfig holds the configurable thresholds for the authorship
// metric family defined in issue #550. All fields are optional; when nil the
// metric computation uses the spec-mandated defaults (see
// internal/provider/git.DefaultAuthorshipParams).
type AuthorshipConfig struct {
	// ActivityWindowDays is the look-back window (in days from HEAD) used by
	// orphan-risk to decide whether an author is "still active". Default: 180.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	ActivityWindowDays *int `yaml:"activity-window-days,omitempty" json:"activity-window-days,omitempty"`

	// RecentWindowDays is the look-back window (in days from HEAD) for
	// current-maintainer. Default: 180.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	RecentWindowDays *int `yaml:"recent-window-days,omitempty" json:"recent-window-days,omitempty"`

	// EarlyWindowFraction is the fraction of a node's calendar lifetime that
	// defines the "early window" for initial-developer and knowledge-handoff.
	// Default: 0.25.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	EarlyWindowFraction *float64 `yaml:"early-window-fraction,omitempty" json:"early-window-fraction,omitempty"`

	// SignificantShareThreshold is the minimum share (Sₐ = Wₐ/W) an author
	// must hold to count toward significant-contributor-count. Default: 0.10.
	//nolint:tagliatelle,revive // External config keys use hyphens; schema field exceeds the line limit.
	SignificantShareThreshold *float64 `yaml:"significant-share-threshold,omitempty" json:"significant-share-threshold,omitempty"`

	// BusFactorThreshold is the minimum combined share that bus-factor authors
	// must cover. Default: 0.50.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	BusFactorThreshold *float64 `yaml:"bus-factor-threshold,omitempty" json:"bus-factor-threshold,omitempty"`

	// IdentityTopK is the number of top contributors assigned distinct colours
	// in identity-metric legends; contributors beyond this rank are bucketed
	// into «other». Default: 11.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	IdentityTopK *int `yaml:"identity-top-k,omitempty" json:"identity-top-k,omitempty"`

	// HonorMailmap, when true, normalises author identity via .mailmap before
	// aggregating contributions. Default: true.
	//nolint:tagliatelle,lll // External config keys use hyphens.
	HonorMailmap *bool `yaml:"honor-mailmap,omitempty" json:"honor-mailmap,omitempty"`
}
