package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// registry holds registered metric providers.
type registry struct {
	mu        sync.RWMutex
	providers map[metric.Name]Interface
}

func newRegistry() *registry {
	return &registry{providers: make(map[metric.Name]Interface)}
}

func (r *registry) register(p Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[p.Name()]; exists {
		panic(fmt.Sprintf("provider %q already registered", p.Name()))
	}

	r.providers[p.Name()] = p
}

func (r *registry) get(name metric.Name) (Interface, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok || p == nil {
		return nil, false
	}

	return p, ok
}

func (r *registry) all() []Interface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := slices.Collect(maps.Values(r.providers))
	slices.SortFunc(
		result,
		func(left Interface, right Interface) int {
			return cmp.Compare(left.Name(), right.Name())
		},
	)

	return result
}

func (r *registry) names() []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := slices.Collect(maps.Keys(r.providers))

	slices.SortFunc(names, cmp.Compare)

	return names
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry. Panics on duplicate name.
func Register(p Interface) { globalRegistry.register(p) }

// Get retrieves a provider by name from the global registry.
func Get(name metric.Name) (Interface, bool) { return globalRegistry.get(name) }

// GetDescriptor retrieves only the metadata for a provider by name.
// Use this instead of Get() when you only need provider metadata.
func GetDescriptor(name metric.Name) (MetricDescriptor, bool) {
	p, ok := globalRegistry.get(name)
	if !ok {
		return MetricDescriptor{}, false
	}

	return Descriptor(p), true
}

// All returns all registered providers.
func All() []Interface { return globalRegistry.all() }

// AllDescriptors returns metadata for all registered providers.
// Use this instead of All() when you only need provider metadata.
func AllDescriptors() []MetricDescriptor {
	providers := globalRegistry.all()

	descriptors := make([]MetricDescriptor, len(providers))
	for i, p := range providers {
		descriptors[i] = Descriptor(p)
	}

	return descriptors
}

// Names returns the sorted names of all registered providers.
func Names() []metric.Name { return globalRegistry.names() }

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
