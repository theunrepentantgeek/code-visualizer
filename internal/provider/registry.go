package provider

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// registry holds registered metric providers, grouped by target type.
type registry struct {
	mu        sync.RWMutex
	providers map[metric.Target]map[metric.Name]Interface
}

func newRegistry() *registry {
	return &registry{
		providers: make(map[metric.Target]map[metric.Name]Interface),
	}
}

func (r *registry) register(p Interface) {
	r.mu.Lock()
	defer r.mu.Unlock()

	target := p.Target()
	targetProviders := r.providers[target]

	if targetProviders == nil {
		targetProviders = make(map[metric.Name]Interface)
		r.providers[target] = targetProviders
	}

	if _, exists := targetProviders[p.Name()]; exists {
		panic(fmt.Sprintf("provider %q already registered for target %q", p.Name(), target))
	}

	targetProviders[p.Name()] = p
}

func (r *registry) get(name metric.Name, target metric.Target) (Interface, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil, false
	}

	p, ok := inner[name]
	if !ok || p == nil {
		return nil, false
	}

	return p, true
}

func (r *registry) all(target metric.Target) []Interface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil
	}

	result := slices.Collect(maps.Values(inner))
	slices.SortFunc(
		result,
		func(left Interface, right Interface) int {
			return cmp.Compare(left.Name(), right.Name())
		},
	)

	return result
}

func (r *registry) names(target metric.Target) []metric.Name {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inner := r.providers[target]
	if inner == nil {
		return nil
	}

	names := slices.Collect(maps.Keys(inner))
	slices.SortFunc(names, cmp.Compare)

	return names
}

// hasName reports whether any target has a provider with the given name.
func (r *registry) hasName(name metric.Name) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, inner := range r.providers {
		if _, ok := inner[name]; ok {
			return true
		}
	}

	return false
}

// targetsForName returns all targets that have a provider with the given name.
func (r *registry) targetsForName(name metric.Name) []metric.Target {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var targets []metric.Target

	for target, inner := range r.providers {
		if _, ok := inner[name]; ok {
			targets = append(targets, target)
		}
	}

	return targets
}

// globalRegistry is the process-wide provider registry.
var globalRegistry = newRegistry()

// Register adds a provider to the global registry.
// Panics on duplicate (name, target) pair.
func Register(p Interface) { globalRegistry.register(p) }

// Get retrieves a provider by name and target from the global registry.
func Get(name metric.Name, target metric.Target) (Interface, bool) {
	return globalRegistry.get(name, target)
}

// GetDescriptor retrieves only the metadata for a provider by name and target.
func GetDescriptor(name metric.Name, target metric.Target) (MetricDescriptor, bool) {
	p, ok := globalRegistry.get(name, target)
	if !ok {
		return MetricDescriptor{}, false
	}

	return Descriptor(p), true
}

// All returns all registered providers for the given target.
func All(target metric.Target) []Interface { return globalRegistry.all(target) }

// AllDescriptors returns metadata for all registered providers for the given target.
func AllDescriptors(target metric.Target) []MetricDescriptor {
	providers := globalRegistry.all(target)

	descriptors := make([]MetricDescriptor, len(providers))
	for i, p := range providers {
		descriptors[i] = Descriptor(p)
	}

	return descriptors
}

// Names returns the sorted names of all registered providers for the given target.
func Names(target metric.Target) []metric.Name { return globalRegistry.names(target) }

// FindWithHint looks up a provider by name and target. On failure, it checks
// whether the metric exists for a different target and includes that as a hint.
func FindWithHint(name metric.Name, target metric.Target) (Interface, error) {
	p, ok := globalRegistry.get(name, target)
	if ok {
		return p, nil
	}

	targets := globalRegistry.targetsForName(name)
	if len(targets) > 0 {
		return nil, fmt.Errorf(
			"unknown %s metric %q; metric %q exists for target %q",
			target, name, name, targets[0],
		)
	}

	return nil, fmt.Errorf("unknown %s metric %q", target, name)
}

// ResetRegistryForTesting clears the global registry. Test use only.
func ResetRegistryForTesting() {
	globalRegistry = newRegistry()
}
