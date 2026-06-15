package metric

// MetricLevel identifies where raw data lives in the model hierarchy.
type MetricLevel int

const (
	LevelFile        MetricLevel = iota // native to files (file-size, file-lines)
	LevelDeclaration                    // native to declarations (cyclomatic-complexity)
	LevelCommit                         // native to commits (commit-date)
	LevelDirectory                      // native to directories (computed aggregates)
)

// String returns the human-readable name of the level.
func (l MetricLevel) String() string {
	switch l {
	case LevelFile:
		return "file"
	case LevelDeclaration:
		return "declaration"
	case LevelCommit:
		return "commit"
	case LevelDirectory:
		return "directory"
	default:
		return "unknown"
	}
}
