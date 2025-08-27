# High-Performance HTTP Template Parser

A Go module that provides a high-performance parser based on text/template for HTTP requests. This module is designed to handle non-rewindable HTTP input streams efficiently while providing template caching and automatic recompilation on file changes.

## Features

- **Re-readable HTTP Requests**: Handles non-rewindable HTTP input streams by buffering them in memory
- **Template Caching**: High-performance template caching with LRU eviction
- **File Watching**: Automatic template recompilation when template files change
- **Flexible Template Loading**: Supports file system and memory-based template loaders
- **Rich Request Data**: Extracts headers, query parameters, form data, and body content
- **Custom Function Maps**: Support for custom template functions
- **Thread-Safe**: Concurrent access safe across goroutines
- **Benchmarked Performance**: Optimized for high-throughput scenarios

## Installation

```bash
go get github.com/fabricates/parser
```

## Quick Start

```go
package main

import (
    "bytes"
    "fmt"
    "net/http"
    "strings"
    
    "github.com/fabricates/parser"
)

func main() {
    // Simple configuration (uses MemoryLoader by default)
    config := parser.Config{
        MaxCacheSize: 100,
        FuncMap:      parser.DefaultFuncMap(),
    }
    
    // Create parser
    p, err := parser.NewParser(config)
    if err != nil {
        panic(err)
    }
    defer p.Close()
    
    // Add template dynamically
    err = p.UpdateTemplate("greeting", "Hello {{.Request.Method}} from {{.Request.URL.Path}}!", "v1")
    if err != nil {
        panic(err)
    }
    
    // Create HTTP request
    req, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
    
    // Parse template
    var output bytes.Buffer
    err = p.Parse("greeting", req, &output)
    if err != nil {
        panic(err)
    }
    
    fmt.Print(output.String()) // Output: Hello GET from /api/users!
}
```

### Alternative: Explicit MemoryLoader

```go
func main() {
    // Create a memory-based template loader explicitly
    loader := parser.NewMemoryLoader()
    loader.AddTemplate("greeting", "Hello {{.Request.Method}} from {{.Request.URL.Path}}!")
    
    // Create parser configuration
    config := parser.Config{
        TemplateLoader: loader,
        MaxCacheSize:   100,
        WatchFiles:     false,
        FuncMap:        parser.DefaultFuncMap(),
    }
    
    // Create parser
    p, err := parser.NewParser(config)
    if err != nil {
        panic(err)
    }
    defer p.Close()
    
    // Create HTTP request
    req, _ := http.NewRequest("GET", "http://example.com/api/users", nil)
    
    // Parse template
    var output bytes.Buffer
    err = p.Parse("greeting", req, &output)
    if err != nil {
        panic(err)
    }
    
    fmt.Print(output.String()) // Output: Hello GET from /api/users!
}
```

## Core Components

### Parser Interface

The main interface provides methods for template parsing and management:

```go
type Parser interface {
    Parse(templateName string, request *http.Request, output io.Writer) error
    ParseWith(templateName string, request *http.Request, data interface{}, output io.Writer) error
    UpdateTemplate(name string, content string, hash string) error
    GetCacheStats() CacheStats
    Close() error
}
```

### Template Loaders

#### File System Loader

Load templates from the file system with optional file watching:

```go
loader := parser.NewFileSystemLoader("/path/to/templates", ".tmpl", true)

config := parser.Config{
    TemplateLoader: loader,
    WatchFiles:     true, // Enable automatic reloading
    MaxCacheSize:   50,
}

p, err := parser.NewParser(config)
```

#### Memory Loader

For testing or when templates are embedded:

```go
loader := parser.NewMemoryLoader()
loader.AddTemplate("welcome", "Welcome {{.Custom.username}}!")

config := parser.Config{
    TemplateLoader: loader,
    MaxCacheSize:   10,
}

p, err := parser.NewParser(config)
```

### Request Data Structure

Templates have access to structured request data:

```go
type RequestData struct {
    Request *http.Request           // Original HTTP request
    Headers map[string][]string     // HTTP headers
    Query   map[string][]string     // Query parameters
    Form    map[string][]string     // Form data (for POST requests)
    Body    string                  // Request body as string
    Custom  interface{}             // Custom data passed to ParseWith
}
```

### Dynamic Template Updates

You can dynamically add or update templates at runtime using the `UpdateTemplate` method:

```go
// Add a new template
templateContent := "Hello {{.Request.Method}} from {{.Request.URL.Path}}!"
err := parser.UpdateTemplate("greeting", templateContent, "hash123")
if err != nil {
    log.Fatalf("Failed to update template: %v", err)
}

// Later, update the same template with new content
newContent := "Updated: {{.Request.Method}} {{.Request.URL.Path}}"
err = parser.UpdateTemplate("greeting", newContent, "hash456")
if err != nil {
    log.Fatalf("Failed to update template: %v", err)
}

// Use the updated template
var output bytes.Buffer
err = parser.Parse("greeting", request, &output)
```

## Template Examples

### Basic Request Information

```html
Method: {{.Request.Method}}
URL: {{.Request.URL.Path}}
User-Agent: {{index .Headers "User-Agent" 0}}
```

### Query Parameters

```html
{{if .Query.name}}
Hello {{index .Query "name" 0}}!
{{end}}
```

### Form Data

```html
{{if .Form.username}}
Username: {{index .Form "username" 0}}
{{end}}
```

### Request Body

```html
{{if .Body}}
Received: {{.Body}}
{{end}}
```

### Custom Data

```go
customData := map[string]interface{}{
    "user_id": 123,
    "role":    "admin",
}

err := parser.ParseWith("template", request, customData, output)
```

```html
User ID: {{.Custom.user_id}}
Role: {{.Custom.role}}
```

## Built-in Template Functions

The parser includes useful template functions:

- `upper`: Convert string to uppercase
- `lower`: Convert string to lowercase  
- `title`: Convert string to title case
- `trim`: Remove leading/trailing whitespace
- `default`: Provide default value for empty/nil values
- `header`: Get request header value
- `query`: Get query parameter value
- `form`: Get form field value

Example usage:

```html
Name: {{.Custom.name | upper | default "Anonymous"}}
Content-Type: {{header .Request "Content-Type"}}
```

## File Watching

When `WatchFiles` is enabled, the parser automatically detects template file changes and recompiles them:

```go
config := parser.Config{
    TemplateLoader: parser.NewFileSystemLoader("./templates", ".tmpl", true),
    WatchFiles:     true,
    MaxCacheSize:   100,
}

p, err := parser.NewParser(config)
// Templates will be automatically reloaded when files change
```

## Performance Optimizations

### Template Caching

Templates are cached after compilation with LRU eviction:

```go
config := parser.Config{
    TemplateLoader: loader,
    MaxCacheSize:   100, // Cache up to 100 templates
}

// Get cache statistics
stats := p.GetCacheStats()
fmt.Printf("Cache: %d/%d, Hits: %d\n", stats.Size, stats.MaxSize, stats.HitCount)
```

### Re-readable Requests

HTTP request bodies are automatically buffered to allow multiple reads:

```go
// The parser handles this automatically
rereadableReq, err := parser.NewRereadableRequest(originalRequest)
rereadableReq.Reset() // Reset body for re-reading
```

## Error Handling

The module defines specific error types:

```go
var (
    ErrTemplateNotFound = errors.New("template not found")
    ErrWatcherClosed    = errors.New("file watcher is closed")
    ErrInvalidConfig    = errors.New("invalid configuration")
    ErrParserClosed     = errors.New("parser is closed")
)
```

## Testing

Run the test suite:

```bash
go test -v
```

Run benchmarks:

```bash
go test -bench=.
```

Example benchmark results:
```
BenchmarkParserParse-2           193795    6008 ns/op
BenchmarkRequestExtraction-2     314965    3690 ns/op
```

## Examples

See the `/examples` directory for complete usage examples:

- `examples/basic/`: Basic usage with different request types
- More examples coming soon!

## Configuration

```go
type Config struct {
    TemplateLoader TemplateLoader    // How to load templates (defaults to MemoryLoader if nil)
    WatchFiles     bool              // Enable file watching (FileSystemLoader only)
    MaxCacheSize   int               // Template cache size (0 = unlimited)
    FuncMap        template.FuncMap  // Custom template functions
}
```

### Default Behavior

If no `TemplateLoader` is specified in the config, the parser will automatically use a `MemoryLoader` by default. This allows you to create a parser with minimal configuration:

```go
// Simple config with default MemoryLoader
config := parser.Config{
    MaxCacheSize: 100,
}

p, err := parser.NewParser(config)
if err != nil {
    panic(err)
}

// Add templates dynamically using UpdateTemplate
err = p.UpdateTemplate("greeting", "Hello {{.Request.Method}}!", "hash123")
```

## Thread Safety

All components are designed to be thread-safe and can be used concurrently across multiple goroutines.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.