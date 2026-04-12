package model

import (
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// Directory represents a directory in the scanned tree.
type Directory struct {
	Path  string
	Name  string
	Files []*File
	Dirs  []*Directory

	mu              sync.RWMutex
	quantities      map[metric.Name]int
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int value for the named metric and whether it was set.
func (d *Directory) Quantity(name metric.Name) (int, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (d *Directory) Measure(name metric.Name) (float64, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (d *Directory) Classification(name metric.Name) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	v, ok := d.classifications[name]

	return v, ok
}

// SetQuantity stores an int metric value identified by name.
func (d *Directory) SetQuantity(name metric.Name, v int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.quantities == nil {
		d.quantities = make(map[metric.Name]int)
	}

	d.quantities[name] = v
}

// SetMeasure stores a float64 metric value identified by name.
func (d *Directory) SetMeasure(name metric.Name, v float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.measures == nil {
		d.measures = make(map[metric.Name]float64)
	}

	d.measures[name] = v
}

// SetClassification stores a string metric value identified by name.
func (d *Directory) SetClassification(name metric.Name, v string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.classifications == nil {
		d.classifications = make(map[metric.Name]string)
	}

	d.classifications[name] = v
}
