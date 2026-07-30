package viz

// Grain controls the granularity of the nodes shown in a diagram.
type Grain string

const (
	// GrainFile shows both directories and files.
	GrainFile Grain = "file"
	// GrainDirectory shows directories only, omitting files.
	GrainDirectory Grain = "directory"
)
