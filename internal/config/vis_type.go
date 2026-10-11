package config

// VisType identifies a supported visualization.
type VisType string

const (
	// VisTypeNone selects only common configuration defaults.
	VisTypeNone       VisType = ""
	VisTypeTreeMap    VisType = "tree-map"
	VisTypeRadialTree VisType = "radial-tree"
	VisTypeDonutTree  VisType = "donut-tree"
	VisTypeBubbleTree VisType = "bubble-tree"
	VisTypeSpiral     VisType = "spiral"
	VisTypeScatter    VisType = "scatter"
	VisTypeAlluvial   VisType = "alluvial"
)
