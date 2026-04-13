// Package model defines the tree data structure used by the metric framework.
package model

import (
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// File represents a single file in the scanned tree.
type File struct {
	Path      string
	Name      string
	Extension string
	IsBinary  bool

	mu              sync.RWMutex
	quantities      map[metric.Name]int64
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int64 value for the named metric and whether it was set.
func (f *File) Quantity(name metric.Name) (int64, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (f *File) Measure(name metric.Name) (float64, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (f *File) Classification(name metric.Name) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	v, ok := f.classifications[name]

	return v, ok
}

// SetQuantity stores an int64 metric value identified by name.
func (f *File) SetQuantity(name metric.Name, v int64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.quantities == nil {
		f.quantities = make(map[metric.Name]int64)
	}

	f.quantities[name] = v
}

// SetMeasure stores a float64 metric value identified by name.
func (f *File) SetMeasure(name metric.Name, v float64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.measures == nil {
		f.measures = make(map[metric.Name]float64)
	}

	f.measures[name] = v
}

// SetClassification stores a string metric value identified by name.
func (f *File) SetClassification(name metric.Name, v string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.classifications == nil {
		f.classifications = make(map[metric.Name]string)
	}

	f.classifications[name] = v
}
