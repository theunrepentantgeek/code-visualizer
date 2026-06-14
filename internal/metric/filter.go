package metric

// FilterName identifies a filter/qualifier applied to a base metric (e.g., "public", "stdlib").
type FilterName string

// IsZero reports whether the filter name is empty (no filter applied).
func (f FilterName) IsZero() bool {
	return f == ""
}
