package parser

import (
	"context"
	"sync"
	"time"
)

// TemplateLoader defines the interface for loading templates
type TemplateLoader interface {
	// Load returns the template content by name
	Load(name string) (content string, err error)

	// List returns all available template names
	List() ([]string, error)

	// Watch starts watching for template changes and calls the callback
	// when a template is modified. Returns a context cancel function.
	Watch(ctx context.Context, callback func(name string)) error

	// LastModified returns the last modification time of a template
	LastModified(name string) (time.Time, error)
}

// MemoryLoader loads templates from memory (useful for testing)
type MemoryLoader struct {
	templates map[string]string
	mu        sync.RWMutex
}

// NewMemoryLoader creates a new memory-based template loader
func NewMemoryLoader() *MemoryLoader {
	return &MemoryLoader{
		templates: make(map[string]string),
	}
}

// AddTemplate adds a template to memory
func (m *MemoryLoader) AddTemplate(name, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[name] = content
}

// Load implements TemplateLoader
func (m *MemoryLoader) Load(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content, exists := m.templates[name]
	if !exists {
		return "", ErrTemplateNotFound
	}

	return content, nil
}

// List implements TemplateLoader
func (m *MemoryLoader) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.templates))
	for name := range m.templates {
		names = append(names, name)
	}

	return names, nil
}

// Watch implements TemplateLoader (no-op for memory loader)
func (m *MemoryLoader) Watch(ctx context.Context, callback func(name string)) error {
	// Memory loader doesn't support watching
	return nil
}

// LastModified implements TemplateLoader (returns current time for memory loader)
func (m *MemoryLoader) LastModified(name string) (time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.templates[name]; !exists {
		return time.Time{}, ErrTemplateNotFound
	}

	return time.Now(), nil
}
