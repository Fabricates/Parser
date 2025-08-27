package parser

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
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

// FileSystemLoader loads templates from the file system
type FileSystemLoader struct {
	// RootDir is the root directory for templates
	RootDir string

	// Extension is the file extension for templates (e.g., ".tmpl", ".tpl")
	Extension string

	// Recursive enables recursive directory scanning
	Recursive bool

	mu      sync.RWMutex
	watcher FileWatcher
}

// NewFileSystemLoader creates a new file system template loader
func NewFileSystemLoader(rootDir, extension string, recursive bool) *FileSystemLoader {
	return &FileSystemLoader{
		RootDir:   rootDir,
		Extension: extension,
		Recursive: recursive,
	}
}

// Load implements TemplateLoader
func (f *FileSystemLoader) Load(name string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path := filepath.Join(f.RootDir, name)
	if f.Extension != "" && filepath.Ext(path) != f.Extension {
		path += f.Extension
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// List implements TemplateLoader
func (f *FileSystemLoader) List() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var templates []string

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if !f.Recursive && path != f.RootDir {
				return fs.SkipDir
			}
			return nil
		}

		if f.Extension == "" || filepath.Ext(path) == f.Extension {
			relPath, err := filepath.Rel(f.RootDir, path)
			if err != nil {
				return err
			}

			// Remove extension for template name
			if f.Extension != "" {
				relPath = relPath[:len(relPath)-len(f.Extension)]
			}

			templates = append(templates, relPath)
		}

		return nil
	}

	err := filepath.WalkDir(f.RootDir, walkFn)
	return templates, err
}

// Watch implements TemplateLoader
func (f *FileSystemLoader) Watch(ctx context.Context, callback func(name string)) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.watcher == nil {
		var err error
		f.watcher, err = NewFileWatcher()
		if err != nil {
			return err
		}
	}

	return f.watcher.Watch(ctx, f.RootDir, f.Extension, f.Recursive, callback)
}

// LastModified implements TemplateLoader
func (f *FileSystemLoader) LastModified(name string) (time.Time, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path := filepath.Join(f.RootDir, name)
	if f.Extension != "" && filepath.Ext(path) != f.Extension {
		path += f.Extension
	}

	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}

	return info.ModTime(), nil
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
