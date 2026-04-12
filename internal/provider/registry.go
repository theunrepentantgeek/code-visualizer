package provider

import (
	"fmt"
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

	result := make([]Interface, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}

	return result
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry. Panics on duplicate name.
func Register(p Interface) { globalRegistry.register(p) }

// Get retrieves a provider by name from the global registry.
func Get(name metric.Name) (Interface, bool) { return globalRegistry.get(name) }

// All returns all registered providers.
func All() []Interface { return globalRegistry.all() }

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
