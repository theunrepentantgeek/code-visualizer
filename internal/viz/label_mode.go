// Package viz provides shared types used across visualization layout packages.
package viz

// LabelMode controls which node labels are shown in a diagram.
type LabelMode string

const (
	// LabelAll shows labels for all nodes.
	LabelAll LabelMode = "all"
	// LabelFoldersOnly shows labels for directory nodes only.
	LabelFoldersOnly LabelMode = "folders"
	// LabelLaps shows labels only at lap boundaries (e.g. midnight, week start).
	LabelLaps LabelMode = "laps"
	// LabelNone hides all labels.
	LabelNone LabelMode = "none"
)
