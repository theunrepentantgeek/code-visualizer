package canvas //nolint:revive // max-public-structs: transient alias file removed in Task 6

// This file is a transient compatibility shim introduced when the ink
// machinery moved to the inks package. It will be deleted in Task 6 of the
// inks-encapsulation plan. The shim re-exports ink types/functions from the
// inks package so callers keep compiling while migration is in progress.

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
)

// Type aliases re-export ink types from inks for backward compatibility.
type (
	Ink         = inks.Ink
	InkInfo     = inks.Info
	InkKind     = inks.Kind
	InkOption   = inks.Option
	MetricValue = inks.MetricValue

	RadialGradientInk = inks.RadialGradientInk
)

// Kind constants.
const (
	InkFixed       = inks.KindFixed
	InkNumeric     = inks.KindNumeric
	InkCategorical = inks.KindCategorical
)

// Function re-exports.
//
//nolint:gochecknoglobals // transient aliases removed in Task 6
var (
	FixedInk             = inks.FixedInk
	NumericInk           = inks.NumericInk
	CategoricalInk       = inks.CategoricalInk
	NewRadialGradientInk = inks.NewRadialGradientInk
	WithOpacity          = inks.WithOpacity
	MeasureValue         = inks.MeasureValue
	QuantityValue        = inks.QuantityValue
	CategoryValue        = inks.CategoryValue
)
