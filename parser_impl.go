package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

// templateParser implements the Parser interface
type templateParser struct {
	config Config
	cache  *TemplateCache
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool
}

// genericParser implements the GenericParser interface
type genericParser[T any] struct {
	*templateParser
}

// NewParser creates a new template parser with the given configuration
func NewParser(config Config) (Parser, error) {
	parser, err := newTemplateParser(config)
	if err != nil {
		return nil, err
	}
	return parser, nil
}

// NewGenericParser creates a new generic template parser with the given configuration
func NewGenericParser[T any](config Config) (GenericParser[T], error) {
	parser, err := newTemplateParser(config)
	if err != nil {
		return nil, err
	}
	return &genericParser[T]{templateParser: parser}, nil
}

// newTemplateParser creates the underlying template parser
func newTemplateParser(config Config) (*templateParser, error) {
	// If no TemplateLoader is specified, use MemoryLoader by default
	if config.TemplateLoader == nil {
		config.TemplateLoader = NewMemoryLoader()
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

// Parse implements GenericParser - executes template and returns result as type T
func (g *genericParser[T]) Parse(templateName string, request *http.Request) (T, error) {
	return g.ParseWith(templateName, request, nil)
}

// ParseWith implements GenericParser - executes template with custom data and returns result as type T
func (g *genericParser[T]) ParseWith(templateName string, request *http.Request, data interface{}) (T, error) {
	var zero T

	// Parse template to string buffer first
	var buf bytes.Buffer
	err := g.templateParser.ParseWith(templateName, request, data, &buf)
	if err != nil {
		return zero, err
	}

	// Convert string result to target type T
	result, err := convertToType[T](buf.String())
	if err != nil {
		return zero, err
	}

	return result, nil
}

// convertToType converts a string to the target type T
func convertToType[T any](s string) (T, error) {
	var zero T
	var result interface{}

	// Use type assertion to determine the target type
	switch any(zero).(type) {
	case string:
		result = s
	case []byte:
		result = []byte(s)
	case int:
		val, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return zero, fmt.Errorf("cannot convert '%s' to int: %w", s, err)
		}
		result = val
	case int64:
		val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return zero, fmt.Errorf("cannot convert '%s' to int64: %w", s, err)
		}
		result = val
	case float64:
		val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return zero, fmt.Errorf("cannot convert '%s' to float64: %w", s, err)
		}
		result = val
	case bool:
		val, err := strconv.ParseBool(strings.TrimSpace(s))
		if err != nil {
			return zero, fmt.Errorf("cannot convert '%s' to bool: %w", s, err)
		}
		result = val
	default:
		// For complex types, try JSON unmarshaling
		var target T
		err := json.Unmarshal([]byte(s), &target)
		if err != nil {
			return zero, fmt.Errorf("cannot unmarshal '%s' to type %T: %w", s, zero, err)
		}
		result = target
	}

	return result.(T), nil
}

// Parse implements Parser
func (p *templateParser) Parse(templateName string, request *http.Request, output io.Writer) error {
	return p.ParseWith(templateName, request, nil, output)
}

// ParseWith implements Parser
func (p *templateParser) ParseWith(templateName string, request *http.Request, data interface{}, output io.Writer) error {
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

// UpdateTemplate implements Parser
func (p *templateParser) UpdateTemplate(name string, content string, hash string) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrParserClosed
	}
	p.mu.RUnlock()

	// Check if template exists and has the same hash
	existingHash := p.cache.GetHash(name)
	if existingHash != "" && existingHash == hash {
		// Template exists and hasn't changed, no need to update
		return nil
	}

	// Parse the template content
	tmpl, err := template.New(name).Funcs(p.config.FuncMap).Parse(content)
	if err != nil {
		return err
	}

	// Update the cache directly with the parsed template
	p.cache.Set(name, tmpl, hash)
	return nil
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
	xmlHelper := XMLHelper{}

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

		// XML helper functions
		"xmlAttr":       xmlHelper.GetXMLAttribute,
		"xmlAttrArray":  xmlHelper.GetXMLAttributeArray,
		"xmlValue":      xmlHelper.GetXMLValue,
		"xmlValueArray": xmlHelper.GetXMLValueArray,
		"xmlText":       xmlHelper.GetXMLText,
		"xmlTextArray":  xmlHelper.GetXMLTextArray,
		"hasXMLAttr":    xmlHelper.HasXMLAttribute,
		"hasXMLElement": xmlHelper.HasXMLElement,
		"isXMLArray":    xmlHelper.IsXMLArray,
		"xmlArrayLen":   xmlHelper.XMLArrayLength,
		"xmlAttrs":      xmlHelper.ListXMLAttributes,
		"xmlElements":   xmlHelper.ListXMLElements,
	}
}
