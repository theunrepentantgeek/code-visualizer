package model

import "time"

// Commit represents a single git commit that touched a file.
type Commit struct {
	MetricContainer
	Hash   string
	Author string
	Date   time.Time
}
