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

	// ParseWith executes the named template with custom data along with HTTP request
	ParseWith(templateName string, request *http.Request, data interface{}, output io.Writer) error

	// UpdateTemplate loads or updates a template with the given content
	UpdateTemplate(name string, content string) error

	// GetCacheStats returns cache statistics
	GetCacheStats() CacheStats

	// Close cleanly shuts down the parser and releases resources
	Close() error
}

// GenericParser provides type-safe template parsing for HTTP requests
// T is the target type that the parsed template will be converted to
type GenericParser[T any] interface {
	// Parse executes the named template and returns the result as type T
	Parse(templateName string, request *http.Request) (T, error)

	// ParseWith executes the named template with custom data and returns the result as type T
	ParseWith(templateName string, request *http.Request, data interface{}) (T, error)

	// UpdateTemplate loads or updates a template with the given content
	UpdateTemplate(name string, content string) error

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

	// BodyJSON contains parsed JSON data when Content-Type is application/json
	BodyJSON map[string]interface{}

	// BodyXML contains parsed XML data when Content-Type is text/xml or application/xml
	BodyXML map[string]interface{}

	// Custom contains any additional custom data
	Custom interface{}
}
