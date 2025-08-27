package parser

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"
)

// templateParser implements the Parser interface
type templateParser struct {
	config Config
	cache  *TemplateCache
	watcher FileWatcher
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool
}

// NewParser creates a new template parser with the given configuration
func NewParser(config Config) (Parser, error) {
	if config.TemplateLoader == nil {
		return nil, ErrInvalidConfig
	}
	
	// Create context for file watching
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create template cache
	cache := NewTemplateCache(config.MaxCacheSize, config.FuncMap)
	
	parser := &templateParser{
		config: config,
		cache:  cache,
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Start file watching if enabled
	if config.WatchFiles {
		err := config.TemplateLoader.Watch(ctx, parser.onTemplateChanged)
		if err != nil {
			cancel()
			return nil, err
		}
	}
	
	return parser, nil
}

// Parse implements Parser
func (p *templateParser) Parse(templateName string, request *http.Request, output io.Writer) error {
	return p.ParseWithData(templateName, request, nil, output)
}

// ParseWithData implements Parser
func (p *templateParser) ParseWithData(templateName string, request *http.Request, data interface{}, output io.Writer) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrParserClosed
	}
	p.mu.RUnlock()
	
	// Create re-readable request
	rereadableReq, err := NewRereadableRequest(request)
	if err != nil {
		return err
	}
	
	// Extract request data
	requestData, err := ExtractRequestData(rereadableReq, data)
	if err != nil {
		return err
	}
	
	// Get template from cache
	tmpl, err := p.cache.Get(templateName, p.config.TemplateLoader)
	if err != nil {
		return err
	}
	
	// Execute template
	err = tmpl.Execute(output, requestData)
	
	// Reset request body for potential reuse
	rereadableReq.Reset()
	
	return err
}

// LoadTemplate implements Parser
func (p *templateParser) LoadTemplate(name string) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrParserClosed
	}
	p.mu.RUnlock()
	
	// Loading is handled automatically by the cache in Get method
	_, err := p.cache.Get(name, p.config.TemplateLoader)
	return err
}

// ReloadTemplate implements Parser
func (p *templateParser) ReloadTemplate(name string) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrParserClosed
	}
	p.mu.RUnlock()
	
	// Remove from cache to force reload
	p.cache.Remove(name)
	
	// Load the template again
	return p.LoadTemplate(name)
}

// Close implements Parser
func (p *templateParser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.closed {
		return nil
	}
	
	p.closed = true
	
	// Cancel file watching
	p.cancel()
	
	// Close file watcher if it exists
	if p.watcher != nil {
		p.watcher.Close()
	}
	
	// Clear cache
	p.cache.Clear()
	
	return nil
}

// onTemplateChanged handles template file changes
func (p *templateParser) onTemplateChanged(name string) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return
	}
	p.mu.RUnlock()
	
	// Remove from cache to force reload on next access
	p.cache.Remove(name)
}

// GetCacheStats returns cache statistics
func (p *templateParser) GetCacheStats() CacheStats {
	return p.cache.Stats()
}

// Helper function to create default function map with useful template functions
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		// String functions
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"title": func(s string) string {
			return strings.Title(s)
		},
		"trim": func(s string) string {
			return strings.TrimSpace(s)
		},
		
		// Utility functions
		"default": func(defaultValue, value interface{}) interface{} {
			if value == nil {
				return defaultValue
			}
			if s, ok := value.(string); ok && s == "" {
				return defaultValue
			}
			return value
		},
		
		// Request-specific functions
		"header": func(req *http.Request, name string) string {
			return req.Header.Get(name)
		},
		"query": func(req *http.Request, name string) string {
			return req.URL.Query().Get(name)
		},
		"form": func(req *http.Request, name string) string {
			return req.FormValue(name)
		},
	}
}