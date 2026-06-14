package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// baseRegistry holds registered base metric descriptors.
type baseRegistry struct {
	mu          sync.RWMutex
	descriptors map[metric.Name]BaseMetricDescriptor
	providers   map[metric.Name]ProviderDescriptor
}

func newBaseRegistry() *baseRegistry {
	return &baseRegistry{
		descriptors: make(map[metric.Name]BaseMetricDescriptor),
		providers:   make(map[metric.Name]ProviderDescriptor),
	}
}

func (r *baseRegistry) register(desc BaseMetricDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[desc.Name]; exists {
		panic(fmt.Sprintf("base metric %q already registered", desc.Name))
	}

	r.descriptors[desc.Name] = desc
}

func (r *baseRegistry) registerWithProvider(desc BaseMetricDescriptor, pd ProviderDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.descriptors[desc.Name]; exists {
		panic(fmt.Sprintf("base metric %q already registered", desc.Name))
	}

	r.descriptors[desc.Name] = desc
	r.providers[desc.Name] = pd
}

func (r *baseRegistry) get(name metric.Name) (BaseMetricDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.descriptors[name]

	return d, ok
}

func (r *baseRegistry) providerFor(name metric.Name) (ProviderDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pd, ok := r.providers[name]

	return pd, ok
}

func (r *baseRegistry) all() []BaseMetricDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := slices.Collect(maps.Values(r.descriptors))
	slices.SortFunc(result, func(a, b BaseMetricDescriptor) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

func (r *baseRegistry) allForLevel(level metric.MetricLevel) []BaseMetricDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []BaseMetricDescriptor
	for _, d := range r.descriptors {
		if d.Level == level {
			result = append(result, d)
		}
	}

	slices.SortFunc(result, func(a, b BaseMetricDescriptor) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return result
}

func (r *baseRegistry) names() []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := slices.Collect(maps.Keys(r.descriptors))
	slices.SortFunc(names, cmp.Compare)

	return names
}

// globalBaseRegistry is the process-wide base metric registry.
var globalBaseRegistry = newBaseRegistry()

// RegisterBase adds a base metric descriptor to the global registry.
// Panics on duplicate name.
func RegisterBase(desc BaseMetricDescriptor) {
	globalBaseRegistry.register(desc)
}

// RegisterBaseWithProvider adds a base metric descriptor and associates it
// with the given provider descriptor.
func RegisterBaseWithProvider(desc BaseMetricDescriptor, pd ProviderDescriptor) {
	globalBaseRegistry.registerWithProvider(desc, pd)
}

// GetBase retrieves a base metric descriptor by name.
func GetBase(name metric.Name) (BaseMetricDescriptor, bool) {
	return globalBaseRegistry.get(name)
}

// GetBaseProvider retrieves the provider descriptor for a base metric.
func GetBaseProvider(name metric.Name) (ProviderDescriptor, bool) {
	return globalBaseRegistry.providerFor(name)
}

// AllBase returns all registered base metric descriptors, sorted by name.
func AllBase() []BaseMetricDescriptor {
	return globalBaseRegistry.all()
}

// AllBaseForLevel returns base metrics at the given native level.
func AllBaseForLevel(level metric.MetricLevel) []BaseMetricDescriptor {
	return globalBaseRegistry.allForLevel(level)
}

// BaseNames returns the sorted names of all registered base metrics.
func BaseNames() []metric.Name {
	return globalBaseRegistry.names()
}

// ResetBaseRegistryForTesting clears the global base registry. Test use only.
func ResetBaseRegistryForTesting() {
	globalBaseRegistry = newBaseRegistry()
}
