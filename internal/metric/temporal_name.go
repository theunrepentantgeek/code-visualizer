package metric

// TemporalName identifies a comparison across ordered snapshots.
type TemporalName string

const (
	TemporalDelta     TemporalName = "delta"
	TemporalStepDelta TemporalName = "stepdelta"
)

var knownTemporal = map[TemporalName]struct{}{
	TemporalDelta:     {},
	TemporalStepDelta: {},
}

// IsZero reports whether no temporal comparison was requested.
func (t TemporalName) IsZero() bool {
	return t == ""
}

// IsKnown reports whether the temporal comparison is recognized.
func (t TemporalName) IsKnown() bool {
	_, ok := knownTemporal[t]

	return ok
}
