package stages

import (
	"fmt"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// GitRequiredError reports that a requested metric needs a git repository
// but the target path is not inside one.
type GitRequiredError struct {
	Metric metric.Name
	Target string
}

func (e *GitRequiredError) Error() string {
	return fmt.Sprintf("metric %q requires a git repository, but %q is not a git repository", e.Metric, e.Target)
}

// TargetPathError reports a problem with the target directory argument.
type TargetPathError struct {
	Msg string
}

func (e *TargetPathError) Error() string { return e.Msg }

// OutputPathError reports a problem with the output file path.
type OutputPathError struct {
	Msg string
}

func (e *OutputPathError) Error() string { return e.Msg }

// NoFilesAfterFilterMsg is the message used when binary filtering empties the tree.
const NoFilesAfterFilterMsg = "no files available for visualization after excluding binary files"

// NoFilesAfterFilterError reports that no files remain after filtering.
type NoFilesAfterFilterError struct {
	Msg string
}

func (e *NoFilesAfterFilterError) Error() string { return e.Msg }
