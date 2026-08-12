package config

// AuthorshipConfig holds the configurable thresholds for the authorship
// metric family defined in issue #550. All fields are optional; when nil the
// metric computation uses the spec-mandated defaults (see
// internal/provider/git.DefaultAuthorshipParams).
type AuthorshipConfig struct {
	// ActivityWindowDays is the look-back window (in days from HEAD) used by
	// orphan-risk to decide whether an author is "still active". Default: 180.
	ActivityWindowDays *int `yaml:"activity-window-days,omitempty" json:"activity-window-days,omitempty"` //nolint:tagliatelle

	// RecentWindowDays is the look-back window (in days from HEAD) for
	// current-maintainer. Default: 180.
	RecentWindowDays *int `yaml:"recent-window-days,omitempty" json:"recent-window-days,omitempty"` //nolint:tagliatelle

	// EarlyWindowFraction is the fraction of a node's calendar lifetime that
	// defines the "early window" for initial-developer and knowledge-handoff.
	// Default: 0.25.
	EarlyWindowFraction *float64 `yaml:"early-window-fraction,omitempty" json:"early-window-fraction,omitempty"` //nolint:tagliatelle

	// SignificantShareThreshold is the minimum share (Sₐ = Wₐ/W) an author
	// must hold to count toward significant-contributor-count. Default: 0.10.
	SignificantShareThreshold *float64 `yaml:"significant-share-threshold,omitempty" json:"significant-share-threshold,omitempty"` //nolint:tagliatelle

	// BusFactorThreshold is the minimum combined share that bus-factor authors
	// must cover. Default: 0.50.
	BusFactorThreshold *float64 `yaml:"bus-factor-threshold,omitempty" json:"bus-factor-threshold,omitempty"` //nolint:tagliatelle

	// IdentityTopK is the number of top contributors assigned distinct colours
	// in identity-metric legends; contributors beyond this rank are bucketed
	// into «other». Default: 11.
	IdentityTopK *int `yaml:"identity-top-k,omitempty" json:"identity-top-k,omitempty"` //nolint:tagliatelle

	// HonorMailmap, when true, normalises author identity via .mailmap before
	// aggregating contributions. Default: true.
	HonorMailmap *bool `yaml:"honor-mailmap,omitempty" json:"honor-mailmap,omitempty"` //nolint:tagliatelle
}
