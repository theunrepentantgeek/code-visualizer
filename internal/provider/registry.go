package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/bevan/code-visualizer/internal/metric"
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
		})

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

// All returns all registered providers.
func All() []Interface { return globalRegistry.all() }

// Names returns the sorted names of all registered providers.
func Names() []metric.Name { return globalRegistry.names() }

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
