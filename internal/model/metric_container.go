package model

import (
	"maps"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// MetricContainer is a thread-safe container for named metric values.
// File and Directory embed it to share a single implementation.
type MetricContainer struct {
	mu              sync.RWMutex
	quantities      map[metric.Name]int64
	measures        map[metric.Name]float64
	classifications map[metric.Name]string
}

// Quantity returns the int64 value for the named metric and whether it was set.
func (mc *MetricContainer) Quantity(name metric.Name) (int64, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.quantities == nil {
		return 0, false
	}

	v, ok := mc.quantities[name]

	return v, ok
}

// Measure returns the float64 value for the named metric and whether it was set.
func (mc *MetricContainer) Measure(name metric.Name) (float64, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.measures == nil {
		return 0, false
	}

	v, ok := mc.measures[name]

	return v, ok
}

// Classification returns the string value for the named metric and whether it was set.
func (mc *MetricContainer) Classification(name metric.Name) (string, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	v, ok := mc.classifications[name]

	return v, ok
}

// Clone returns a deep copy of the container's metric maps.
func (mc *MetricContainer) Clone() *MetricContainer {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	clone := &MetricContainer{}
	if mc.quantities != nil {
		clone.quantities = make(map[metric.Name]int64, len(mc.quantities))
		maps.Copy(clone.quantities, mc.quantities)
	}

	if mc.measures != nil {
		clone.measures = make(map[metric.Name]float64, len(mc.measures))
		maps.Copy(clone.measures, mc.measures)
	}

	if mc.classifications != nil {
		clone.classifications = make(map[metric.Name]string, len(mc.classifications))
		maps.Copy(clone.classifications, mc.classifications)
	}

	return clone
}

func (mc *MetricContainer) cloneInto(target *MetricContainer) {
	clone := mc.Clone()
	target.quantities = clone.quantities
	target.measures = clone.measures
	target.classifications = clone.classifications
}

// SetQuantity stores an int64 metric value identified by name.
func (mc *MetricContainer) SetQuantity(name metric.Name, v int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.quantities == nil {
		mc.quantities = make(map[metric.Name]int64)
	}

	mc.quantities[name] = v
}

// SetMeasure stores a float64 metric value identified by name.
func (mc *MetricContainer) SetMeasure(name metric.Name, v float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.measures == nil {
		mc.measures = make(map[metric.Name]float64)
	}

	mc.measures[name] = v
}

// SetClassification stores a string metric value identified by name.
func (mc *MetricContainer) SetClassification(name metric.Name, v string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.classifications == nil {
		mc.classifications = make(map[metric.Name]string)
	}

	mc.classifications[name] = v
}
