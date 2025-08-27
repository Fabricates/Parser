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
go get github.com/Fabricates/Parser
```

## Quick Start

```go
package main

import (
    "bytes"
    "fmt"
    "net/http"
    "strings"
    
    "github.com/Fabricates/Parser"
)

func main() {
    // Create a memory-based template loader
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
    ParseWithData(templateName string, request *http.Request, data interface{}, output io.Writer) error
    LoadTemplate(name string) error
    ReloadTemplate(name string) error
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
    Custom  interface{}             // Custom data passed to ParseWithData
}
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

err := parser.ParseWithData("template", request, customData, output)
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
stats := parser.GetCacheStats()
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
    TemplateLoader TemplateLoader    // Required: how to load templates
    WatchFiles     bool              // Enable file watching (FileSystemLoader only)
    MaxCacheSize   int               // Template cache size (0 = unlimited)
    FuncMap        template.FuncMap  // Custom template functions
}
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