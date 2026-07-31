package scatter

// ScaleType controls how numeric values are mapped to axis positions.
type ScaleType int

const (
	// Linear maps values to axis positions proportionally.
	Linear ScaleType = iota
	// Log maps values to axis positions on a base-10 logarithmic scale.
	Log
)

// NumericAxis holds the numeric range and ticks for one axis.
type NumericAxis struct {
	Min   float64
	Max   float64
	Scale ScaleType
	Ticks []AxisTick
}

// CategoricalAxis holds the categorical swimlanes for one axis.
// Centers maps each label to its band centre position for O(1) lookup.
// Centers is nil for manually-constructed axes (e.g. in tests); positionForValue
// falls back to a linear scan over Bands in that case.
type CategoricalAxis struct {
	Bands   []AxisBand
	Centers map[string]float64
}

// ResolvedAxis is the layout-ready representation of a scatter axis.
type ResolvedAxis struct {
	Spec        AxisSpec
	Title       string
	Numeric     *NumericAxis
	Categorical *CategoricalAxis
}

// NumericTicks returns the tick marks for a numeric axis, or nil when the
// receiver is nil or the axis has no numeric data.
func (a *ResolvedAxis) NumericTicks() []AxisTick {
	if a == nil || a.Numeric == nil {
		return nil
	}

	return a.Numeric.Ticks
}

// CategoricalBands returns the swimlanes for a categorical axis, or nil when
// the receiver is nil or the axis has no categorical data.
func (a *ResolvedAxis) CategoricalBands() []AxisBand {
	if a == nil || a.Categorical == nil {
		return nil
	}

	return a.Categorical.Bands
}

// Offset shifts all absolute positions in the axis by delta, adjusting tick
// positions (numeric) and band start/end/center (categorical). A nil receiver
// is a no-op.
func (a *ResolvedAxis) Offset(delta float64) {
	if a == nil {
		return
	}

	for i := range a.NumericTicks() {
		a.Numeric.Ticks[i].Position += delta
	}

	for i := range a.CategoricalBands() {
		a.Categorical.Bands[i].Start += delta
		a.Categorical.Bands[i].End += delta
		a.Categorical.Bands[i].Center += delta
	}

	if a.Categorical != nil {
		for label := range a.Categorical.Centers {
			a.Categorical.Centers[label] += delta
		}
	}
}
