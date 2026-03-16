package llm

import (
	"fmt"
)

// ProviderFactory builds a Provider from config values.
// Registered by each sub-package via init() or explicit call.
type ProviderFactory func(apiKey, baseURL, model string) Provider

var factories = map[string]ProviderFactory{}

// Register registers a provider factory under a name.
func Register(name string, f ProviderFactory) {
	factories[name] = f
}

// New returns a Provider for the given name.
func New(name, apiKey, baseURL, model string) (Provider, error) {
	f, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown LLM provider %q — supported: %v", name, Keys())
	}
	return f(apiKey, baseURL, model), nil
}

// Keys returns registered provider names.
func Keys() []string {
	ks := make([]string, 0, len(factories))
	for k := range factories {
		ks = append(ks, k)
	}
	return ks
}
