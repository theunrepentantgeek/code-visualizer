package provider

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
)

// registration pairs a MetricDescriptor with its Loader.
type registration struct {
	descriptor MetricDescriptor
	loader     Loader
}

// registry holds registered metric providers.
type registry struct {
	mu      sync.RWMutex
	entries map[metric.Name]registration
}

func newRegistry() *registry {
	return &registry{entries: make(map[metric.Name]registration)}
}

func (r *registry) register(desc MetricDescriptor, loader Loader) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[desc.Name]; exists {
		panic(fmt.Sprintf("provider %q already registered", desc.Name))
	}

	r.entries[desc.Name] = registration{descriptor: desc, loader: loader}
}

func (r *registry) get(name metric.Name) (MetricDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.entries[name]
	if !ok {
		return MetricDescriptor{}, false
	}

	return e.descriptor, true
}

func (r *registry) getLoader(name metric.Name) (Loader, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.entries[name]
	if !ok {
		return nil, false
	}

	return e.loader, true
}

func (r *registry) all() []MetricDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descs := make([]MetricDescriptor, 0, len(r.entries))
	for _, e := range r.entries {
		descs = append(descs, e.descriptor)
	}

	slices.SortFunc(descs, func(a, b MetricDescriptor) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return descs
}

func (r *registry) names() []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]metric.Name, 0, len(r.entries))
	for name := range r.entries {
		names = append(names, name)
	}

	slices.SortFunc(names, cmp.Compare)

	return names
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry. Panics on duplicate name.
func Register(desc MetricDescriptor, loader Loader) { globalRegistry.register(desc, loader) }

// Get retrieves a provider descriptor by name from the global registry.
func Get(name metric.Name) (MetricDescriptor, bool) { return globalRegistry.get(name) }

// All returns descriptors of all registered providers, sorted by name.
func All() []MetricDescriptor { return globalRegistry.all() }

// Names returns the sorted names of all registered providers.
func Names() []metric.Name { return globalRegistry.names() }

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
