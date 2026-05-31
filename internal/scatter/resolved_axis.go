package scatter

// NumericAxis holds the numeric range and ticks for one axis.
type NumericAxis struct {
	Min   float64
	Max   float64
	Ticks []AxisTick
}

// CategoricalAxis holds the categorical swimlanes for one axis.
type CategoricalAxis struct {
	Bands []AxisBand
}

// ResolvedAxis is the layout-ready representation of a scatter axis.
type ResolvedAxis struct {
	Spec        AxisSpec
	Title       string
	Numeric     *NumericAxis
	Categorical *CategoricalAxis
}

func (a ResolvedAxis) NumericTicks() []AxisTick {
	if a.Numeric == nil {
		return nil
	}

	return a.Numeric.Ticks
}

func (a ResolvedAxis) CategoricalBands() []AxisBand {
	if a.Categorical == nil {
		return nil
	}

	return a.Categorical.Bands
}

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
}
