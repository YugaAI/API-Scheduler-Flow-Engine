package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Action defines the interface that all built-in actions must implement.
type Action interface {
	Name() string
	Execute(ctx context.Context, config json.RawMessage) (string, error)
}

// ActionRegistry manages the available action handlers.
type ActionRegistry struct {
	mu      sync.RWMutex
	actions map[string]Action
}

// NewActionRegistry creates a new ActionRegistry.
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		actions: make(map[string]Action),
	}
}

// Register adds an action to the registry.
func (r *ActionRegistry) Register(action Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[action.Name()] = action
}

// Get retrieves an action by its name.
func (r *ActionRegistry) Get(name string) (Action, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	action, exists := r.actions[name]
	if !exists {
		return nil, fmt.Errorf("unknown action: %s", name)
	}
	return action, nil
}

// Validate checks if a given action name is registered.
func (r *ActionRegistry) Validate(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.actions[name]
	return exists
}
