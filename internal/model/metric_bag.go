package model

import (
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// MetricBag is a thread-safe container for named metric values.
// File and Directory embed it to share a single implementation.
type MetricBag struct {
	mu              sync.RWMutex
	quantities      map[metric.Name]int64
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int64 value for the named metric and whether it was set.
func (b *MetricBag) Quantity(name metric.Name) (int64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	v, ok := b.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (b *MetricBag) Measure(name metric.Name) (float64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	v, ok := b.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (b *MetricBag) Classification(name metric.Name) (string, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	v, ok := b.classifications[name]

	return v, ok
}

// SetQuantity stores an int64 metric value identified by name.
func (b *MetricBag) SetQuantity(name metric.Name, v int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.quantities == nil {
		b.quantities = make(map[metric.Name]int64)
	}

	b.quantities[name] = v
}

// SetMeasure stores a float64 metric value identified by name.
func (b *MetricBag) SetMeasure(name metric.Name, v float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.measures == nil {
		b.measures = make(map[metric.Name]float64)
	}

	b.measures[name] = v
}

// SetClassification stores a string metric value identified by name.
func (b *MetricBag) SetClassification(name metric.Name, v string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.classifications == nil {
		b.classifications = make(map[metric.Name]string)
	}

	b.classifications[name] = v
}
