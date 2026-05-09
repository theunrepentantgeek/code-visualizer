package canvas

// LegendPosition specifies where the legend is placed on the canvas.
type LegendPosition string

const (
	LegendPositionNone         LegendPosition = "none"
	LegendPositionTopLeft      LegendPosition = "top-left"
	LegendPositionTopCenter    LegendPosition = "top-center"
	LegendPositionTopRight     LegendPosition = "top-right"
	LegendPositionCenterRight  LegendPosition = "center-right"
	LegendPositionBottomRight  LegendPosition = "bottom-right"
	LegendPositionBottomCenter LegendPosition = "bottom-center"
	LegendPositionBottomLeft   LegendPosition = "bottom-left"
	LegendPositionCenterLeft   LegendPosition = "center-left"
)

// LegendOrientation controls whether swatches are stacked vertically
// or laid out horizontally.
type LegendOrientation string

const (
	LegendOrientationVertical   LegendOrientation = "vertical"
	LegendOrientationHorizontal LegendOrientation = "horizontal"
)

// LegendRole identifies what visual property a legend entry describes.
type LegendRole string

const (
	LegendRoleFill   LegendRole = "Fill"
	LegendRoleBorder LegendRole = "Border"
	LegendRoleSize   LegendRole = "Size"
)

// LegendEntry describes one metric shown in the legend.
type LegendEntry struct {
	Role       LegendRole
	MetricName string
	Ink        Ink
}

// LegendConfig holds everything needed to render a legend.
type LegendConfig struct {
	Position    LegendPosition
	Orientation LegendOrientation
	Entries     []LegendEntry
}
