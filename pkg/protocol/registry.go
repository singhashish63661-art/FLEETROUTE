package protocol

import (
	"fmt"
	"sync"
)

// Registry is a thread-safe store of registered protocol handlers.
// Handlers are matched in registration order during auto-detection.
type Registry struct {
	mu       sync.RWMutex
	handlers []Handler
	byName   map[string]Handler
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Handler)}
}

// Register adds a handler to the registry.
func (r *Registry) Register(h Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.byName[h.Name()]; exists {
		return fmt.Errorf("protocol %q already registered", h.Name())
	}
	r.handlers = append(r.handlers, h)
	r.byName[h.Name()] = h
	return nil
}

// Detect returns the first handler whose Detect() returns true for the given header,
// or nil if no handler matches.
func (r *Registry) Detect(header []byte) Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, h := range r.handlers {
		if h.Detect(header) {
			return h
		}
	}
	return nil
}

// Get returns a handler by name.
func (r *Registry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.byName[name]
	return h, ok
}
