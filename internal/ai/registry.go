package ai

import "fmt"

// Registry maps provider names to their implementations.
// Register each Provider once at startup; the ChatService resolves
// the correct backend at request time based on the caller's choice.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry.
// Panics if a provider with the same name is registered twice.
func (r *Registry) Register(p Provider) {
	if _, exists := r.providers[p.Name()]; exists {
		panic(fmt.Sprintf("ai: provider %q already registered", p.Name()))
	}
	r.providers[p.Name()] = p
}

// Get returns the provider for the given name.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Names returns all registered provider names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
