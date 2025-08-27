package parser

import (
	"io"
	"net/http"
	"text/template"
)

// Parser provides high-performance template parsing for HTTP requests
type Parser interface {
	// Parse executes the named template with the given HTTP request data
	Parse(templateName string, request *http.Request, output io.Writer) error
	
	// ParseWithData executes the named template with custom data along with HTTP request
	ParseWithData(templateName string, request *http.Request, data interface{}, output io.Writer) error
	
	// LoadTemplate loads a template from the configured loader
	LoadTemplate(name string) error
	
	// ReloadTemplate forces a reload of the specified template
	ReloadTemplate(name string) error
	
	// GetCacheStats returns cache statistics
	GetCacheStats() CacheStats
	
	// Close cleanly shuts down the parser and releases resources
	Close() error
}

// Config holds configuration for the parser
type Config struct {
	// TemplateLoader specifies how to load templates
	TemplateLoader TemplateLoader
	
	// WatchFiles enables automatic template reloading on file changes
	WatchFiles bool
	
	// MaxCacheSize limits the number of cached templates (0 = unlimited)
	MaxCacheSize int
	
	// FuncMap provides custom template functions
	FuncMap template.FuncMap
}

// RequestData represents the data structure available to templates
type RequestData struct {
	// Request is the original HTTP request
	Request *http.Request
	
	// Headers contains all HTTP headers
	Headers map[string][]string
	
	// Query contains query parameters
	Query map[string][]string
	
	// Form contains form data (for POST requests)
	Form map[string][]string
	
	// Body contains the request body as string
	Body string
	
	// Custom contains any additional custom data
	Custom interface{}
}