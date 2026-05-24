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
