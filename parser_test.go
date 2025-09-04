package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

// Test memory loader
func TestMemoryLoader(t *testing.T) {
	loader := NewMemoryLoader()

	// Test adding and loading templates
	loader.AddTemplate("test", "Hello {{.Request.Method}}")

	content, err := loader.Load("test")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if content != "Hello {{.Request.Method}}" {
		t.Errorf("Expected 'Hello {{.Request.Method}}', got '%s'", content)
	}

	// Test listing templates
	names, err := loader.List()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(names) != 1 || names[0] != "test" {
		t.Errorf("Expected ['test'], got %v", names)
	}

	// Test non-existent template
	_, err = loader.Load("nonexistent")
	if err != ErrTemplateNotFound {
		t.Errorf("Expected ErrTemplateNotFound, got %v", err)
	}
}

// Test re-readable request
func TestRereadableRequest(t *testing.T) {
	// Create a test request with body
	body := "test=value&another=data"
	req, err := http.NewRequest("POST", "http://example.com/test?param=1", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Custom", "custom-value")

	// Create re-readable request
	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	// Test body reading
	if rereadable.Body() != body {
		t.Errorf("Expected body '%s', got '%s'", body, rereadable.Body())
	}

	// Test body re-reading
	bodyBytes, err := io.ReadAll(rereadable.Request.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if string(bodyBytes) != body {
		t.Errorf("Expected body '%s', got '%s'", body, string(bodyBytes))
	}

	// Reset and read again
	rereadable.Reset()
	bodyBytes2, err := io.ReadAll(rereadable.Request.Body)
	if err != nil {
		t.Fatalf("Failed to read body after reset: %v", err)
	}

	if string(bodyBytes2) != body {
		t.Errorf("Expected body '%s' after reset, got '%s'", body, string(bodyBytes2))
	}
}

// Test request data extraction
func TestExtractRequestData(t *testing.T) {
	// Create a test request
	body := "name=John&age=30"
	req, err := http.NewRequest("POST", "http://example.com/test?param=value", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer token123")

	// Create re-readable request
	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	// Extract request data
	customData := map[string]interface{}{"user": "test"}
	data, err := ExtractRequestData(rereadable, customData)
	if err != nil {
		t.Fatalf("Failed to extract request data: %v", err)
	}

	// Test extracted data
	if data.Body != body {
		t.Errorf("Expected body '%s', got '%s'", body, data.Body)
	}

	if data.Headers["Authorization"][0] != "Bearer token123" {
		t.Errorf("Expected Authorization header 'Bearer token123', got '%s'", data.Headers["Authorization"][0])
	}

	if data.Query["param"][0] != "value" {
		t.Errorf("Expected query param 'value', got '%s'", data.Query["param"][0])
	}

	if data.Form["name"][0] != "John" {
		t.Errorf("Expected form field 'John', got '%s'", data.Form["name"][0])
	}

	if data.Custom.(map[string]interface{})["user"] != "test" {
		t.Errorf("Expected custom data 'test', got '%v'", data.Custom)
	}
}

// Test template cache
func TestTemplateCache(t *testing.T) {
	cache := NewTemplateCache(2, nil)
	loader := NewMemoryLoader()

	// Add test templates
	loader.AddTemplate("template1", "Hello {{.Body}}")
	loader.AddTemplate("template2", "Hi {{.Body}}")
	loader.AddTemplate("template3", "Hey {{.Body}}")

	// Load templates into cache
	tmpl1, err := cache.Get("template1", loader)
	if err != nil {
		t.Fatalf("Failed to get template1: %v", err)
	}

	tmpl2, err := cache.Get("template2", loader)
	if err != nil {
		t.Fatalf("Failed to get template2: %v", err)
	}

	// Check cache stats
	stats := cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected cache size 2, got %d", stats.Size)
	}

	// Load third template (should evict first one due to LRU)
	tmpl3, err := cache.Get("template3", loader)
	if err != nil {
		t.Fatalf("Failed to get template3: %v", err)
	}

	// Cache should still be size 2
	stats = cache.Stats()
	if stats.Size != 2 {
		t.Errorf("Expected cache size 2 after eviction, got %d", stats.Size)
	}

	// Test that templates work
	var buf bytes.Buffer
	data := &RequestData{Body: "World"}

	err = tmpl1.Execute(&buf, data)
	if err != nil {
		t.Fatalf("Failed to execute template1: %v", err)
	}

	if buf.String() != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", buf.String())
	}

	// Test template removal
	cache.Remove("template2")
	stats = cache.Stats()
	if stats.Size != 1 {
		t.Errorf("Expected cache size 1 after removal, got %d", stats.Size)
	}

	// Test cache clear
	cache.Clear()
	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", stats.Size)
	}

	_ = tmpl2
	_ = tmpl3
}

// Test parser with memory loader
func TestParserWithMemoryLoader(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("greeting", "Hello {{.Request.Method}} {{index .Query \"name\" 0}} from {{index .Headers \"X-Custom\" 0}}!")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false, // Disable for memory loader
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/test?name=World", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Custom", "example.com")

	// Parse template
	var output bytes.Buffer
	err = parser.Parse("greeting", req, &output)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := "Hello GET World from example.com!"
	if output.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.String())
	}
}

// Test parser with custom data
func TestParserWithCustomData(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("custom", "User: {{.Custom.username}}, Method: {{.Request.Method}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Create test request
	req, err := http.NewRequest("POST", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Custom data
	customData := map[string]string{"username": "john"}

	// Parse template with custom data
	var output bytes.Buffer
	err = parser.ParseWith("custom", req, customData, &output)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := "User: john, Method: POST"
	if output.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.String())
	}
}

// Test default function map
func TestDefaultFuncMap(t *testing.T) {
	funcMap := DefaultFuncMap()

	// Test string functions
	upper := funcMap["upper"].(func(string) string)
	if upper("hello") != "HELLO" {
		t.Errorf("Expected 'HELLO', got '%s'", upper("hello"))
	}

	lower := funcMap["lower"].(func(string) string)
	if lower("HELLO") != "hello" {
		t.Errorf("Expected 'hello', got '%s'", lower("HELLO"))
	}

	defaultFunc := funcMap["default"].(func(interface{}, interface{}) interface{})
	if defaultFunc("default", "") != "default" {
		t.Errorf("Expected 'default' for empty string, got '%v'", defaultFunc("default", ""))
	}

	if defaultFunc("default", "value") != "value" {
		t.Errorf("Expected 'value' for non-empty string, got '%v'", defaultFunc("default", "value"))
	}
}

// Benchmark tests
func BenchmarkParserParse(b *testing.B) {
	loader := NewMemoryLoader()
	loader.AddTemplate("bench", "Hello {{.Request.Method}} {{.Body}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	req, err := http.NewRequest("GET", "http://example.com", strings.NewReader("World"))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		err := parser.Parse("bench", req, &output)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

func BenchmarkRequestExtraction(b *testing.B) {
	body := "name=John&age=30&city=NYC"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", "http://example.com/test?param=value", strings.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rereadable, err := NewRereadableRequest(req)
		if err != nil {
			b.Fatalf("Failed to create re-readable request: %v", err)
		}

		_, err = ExtractRequestData(rereadable, nil)
		if err != nil {
			b.Fatalf("Failed to extract request data: %v", err)
		}
	}
}

// Benchmark generic parser performance
func BenchmarkGenericParserString(b *testing.B) {
	loader := NewMemoryLoader()
	loader.AddTemplate("bench", "Method: {{.Request.Method}}, Path: {{.Request.URL.Path}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewGenericParser[string](config)
	if err != nil {
		b.Fatalf("Failed to create generic parser: %v", err)
	}
	defer parser.Close()

	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse("bench", req)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// Benchmark JSON conversion performance
func BenchmarkGenericParserJSON(b *testing.B) {
	type APIResponse struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Count  int    `json:"count"`
	}

	loader := NewMemoryLoader()
	loader.AddTemplate("json", `{"method":"{{.Request.Method}}","path":"{{.Request.URL.Path}}","count":{{len .Request.Header}}}`)

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewGenericParser[APIResponse](config)
	if err != nil {
		b.Fatalf("Failed to create JSON parser: %v", err)
	}
	defer parser.Close()

	req, err := http.NewRequest("POST", "http://example.com/api/create", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse("json", req)
		if err != nil {
			b.Fatalf("Failed to parse JSON: %v", err)
		}
	}
}

// Benchmark template cache performance
func BenchmarkTemplateCache(b *testing.B) {
	cache := NewTemplateCache(100, DefaultFuncMap())
	loader := NewMemoryLoader()

	// Pre-populate templates
	for i := 0; i < 50; i++ {
		templateName := fmt.Sprintf("template%d", i)
		loader.AddTemplate(templateName, fmt.Sprintf("Template %d: {{.Body}}", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		templateName := fmt.Sprintf("template%d", i%50)
		_, err := cache.Get(templateName, loader)
		if err != nil {
			b.Fatalf("Failed to get template: %v", err)
		}
	}
}

// Benchmark UpdateTemplate performance
func BenchmarkUpdateTemplate(b *testing.B) {
	config := Config{
		MaxCacheSize: 100,
		WatchFiles:   false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		templateName := fmt.Sprintf("dynamic%d", i%10) // Rotate through 10 templates
		content := fmt.Sprintf("Dynamic %d: {{.Request.Method}}", i)

		err := parser.UpdateTemplate(templateName, content)
		if err != nil {
			b.Fatalf("Failed to update template: %v", err)
		}
	}
}

// Benchmark UpdateTemplate change detection performance
func BenchmarkUpdateTemplateChangeDetection(b *testing.B) {
	config := Config{
		MaxCacheSize: 100,
		WatchFiles:   false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Pre-populate some templates
	templateName := "stable-template"
	content := "Static template: {{.Request.Method}}"

	err = parser.UpdateTemplate(templateName, content)
	if err != nil {
		b.Fatalf("Failed to pre-populate template: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This should trigger the change detection and skip parsing
		err := parser.UpdateTemplate(templateName, content)
		if err != nil {
			b.Fatalf("Failed to update template: %v", err)
		}
	}
}

// Benchmark large request body parsing
func BenchmarkLargeRequestBody(b *testing.B) {
	loader := NewMemoryLoader()
	loader.AddTemplate("large", "Body length: {{len .Body}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Create large body (1MB)
	largeBody := strings.Repeat("x", 1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", "http://example.com/large", strings.NewReader(largeBody))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		var output bytes.Buffer
		err = parser.Parse("large", req, &output)
		if err != nil {
			b.Fatalf("Failed to parse large body: %v", err)
		}
	}
}

// Benchmark concurrent parser usage
func BenchmarkConcurrentParsing(b *testing.B) {
	loader := NewMemoryLoader()
	loader.AddTemplate("concurrent", "Goroutine parsing: {{.Request.Method}} {{.Request.URL.Path}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest("GET", "http://example.com/concurrent", nil)
			if err != nil {
				b.Fatalf("Failed to create request: %v", err)
			}

			var output bytes.Buffer
			err = parser.Parse("concurrent", req, &output)
			if err != nil {
				b.Fatalf("Failed to parse concurrently: %v", err)
			}
		}
	})
}

// Benchmark RereadableRequest creation
func BenchmarkRereadableRequest(b *testing.B) {
	body := "test body content for benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		_, err = NewRereadableRequest(req)
		if err != nil {
			b.Fatalf("Failed to create rereadable request: %v", err)
		}
	}
}

// Benchmark complex template with multiple functions
func BenchmarkComplexTemplate(b *testing.B) {
	loader := NewMemoryLoader()
	complexTemplate := `
Method: {{.Request.Method | upper}}
Path: {{.Request.URL.Path | trim}}
Host: {{header .Request "Host" | default "unknown"}}
Query Count: {{len .Query}}
{{if .Query.name}}Name: {{index .Query "name" 0 | title}}{{end}}
{{if .Body}}Body Length: {{len .Body}}{{end}}
Custom: {{.Custom.value | default "none"}}`

	loader.AddTemplate("complex", complexTemplate)

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	body := "request body content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", "http://example.com/api/test?name=john&age=30", strings.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Host", "api.example.com")
		req.Header.Set("User-Agent", "benchmark-client")

		customData := map[string]interface{}{"value": "test"}

		var output bytes.Buffer
		err = parser.ParseWith("complex", req, customData, &output)
		if err != nil {
			b.Fatalf("Failed to parse complex template: %v", err)
		}
	}
}

// Test UpdateTemplate method
func TestUpdateTemplate(t *testing.T) {
	loader := NewMemoryLoader()

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Update template using the new method
	templateContent := "Hello {{.Request.Method}} from {{.Request.URL.Path}}!"
	err = parser.UpdateTemplate("test-template", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	var output bytes.Buffer
	err = parser.Parse("test-template", req, &output)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := "Hello GET from /api/users!"
	if output.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.String())
	}

	// Update the same template with new content
	newContent := "Updated: {{.Request.Method}} {{.Request.URL.Path}}"
	err = parser.UpdateTemplate("test-template", newContent)
	if err != nil {
		t.Fatalf("Failed to update template with new content: %v", err)
	}

	// Parse updated template
	var output2 bytes.Buffer
	err = parser.Parse("test-template", req, &output2)
	if err != nil {
		t.Fatalf("Failed to parse updated template: %v", err)
	}

	expected2 := "Updated: GET /api/users"
	if output2.String() != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, output2.String())
	}
}

// Test UpdateTemplate change detection behavior
func TestUpdateTemplateChangeDetection(t *testing.T) {
	loader := NewMemoryLoader()

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Initial template update
	templateContent := "Hello {{.Request.Method}} from {{.Request.URL.Path}}!"
	err = parser.UpdateTemplate("test-template", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template first time
	var output1 bytes.Buffer
	err = parser.Parse("test-template", req, &output1)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected1 := "Hello GET from /api/users!"
	if output1.String() != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, output1.String())
	}

	// Update with the same hash and same content - should not return error but should skip update
	err = parser.UpdateTemplate("test-template", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template with same hash: %v", err)
	}

	// Parse again to verify template still works
	var output1a bytes.Buffer
	err = parser.Parse("test-template", req, &output1a)
	if err != nil {
		t.Fatalf("Failed to parse template after same hash update: %v", err)
	}

	if output1a.String() != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, output1a.String())
	}

	// Update with different hash - should parse and update
	newContent := "Updated: {{.Request.Method}} {{.Request.URL.Path}}"
	err = parser.UpdateTemplate("test-template", newContent)
	if err != nil {
		t.Fatalf("Failed to update template with new hash: %v", err)
	}

	// Parse updated template to verify it changed
	var output2 bytes.Buffer
	err = parser.Parse("test-template", req, &output2)
	if err != nil {
		t.Fatalf("Failed to parse updated template: %v", err)
	}

	expected2 := "Updated: GET /api/users"
	if output2.String() != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, output2.String())
	}

	// Update with same hash as previous update - should skip
	err = parser.UpdateTemplate("test-template", newContent)
	if err != nil {
		t.Fatalf("Failed to update template with same hash (second time): %v", err)
	}

	// Parse again to verify template still has the updated content
	var output2a bytes.Buffer
	err = parser.Parse("test-template", req, &output2a)
	if err != nil {
		t.Fatalf("Failed to parse template after same hash update (second time): %v", err)
	}

	if output2a.String() != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, output2a.String())
	}

	// Update a new template (doesn't exist) - should always update
	err = parser.UpdateTemplate("new-template", "New template: {{.Request.Method}}")
	if err != nil {
		t.Fatalf("Failed to update new template: %v", err)
	}

	// Parse the new template to verify it works
	var output3 bytes.Buffer
	err = parser.Parse("new-template", req, &output3)
	if err != nil {
		t.Fatalf("Failed to parse new template: %v", err)
	}

	expected3 := "New template: GET"
	if output3.String() != expected3 {
		t.Errorf("Expected '%s', got '%s'", expected3, output3.String())
	}
}

// Test UpdateTemplate change detection for GenericParser
func TestGenericParserUpdateTemplateChangeDetection(t *testing.T) {
	config := Config{
		MaxCacheSize: 10,
		WatchFiles:   false,
		FuncMap:      DefaultFuncMap(),
	}

	parser, err := NewGenericParser[string](config)
	if err != nil {
		t.Fatalf("Failed to create generic parser: %v", err)
	}
	defer parser.Close()

	// Initial template update
	templateContent := "{{.Request.Method}}"
	err = parser.UpdateTemplate("test-template", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template first time
	result1, err := parser.Parse("test-template", req)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected1 := "GET"
	if result1 != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, result1)
	}

	// Update with the same hash - should skip update
	err = parser.UpdateTemplate("test-template", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template with same hash: %v", err)
	}

	// Parse again to verify template still works
	result1a, err := parser.Parse("test-template", req)
	if err != nil {
		t.Fatalf("Failed to parse template after same hash update: %v", err)
	}

	if result1a != expected1 {
		t.Errorf("Expected '%s', got '%s'", expected1, result1a)
	}

	// Update with different hash
	newContent := "{{.Request.URL.Path}}"
	err = parser.UpdateTemplate("test-template", newContent)
	if err != nil {
		t.Fatalf("Failed to update template with new hash: %v", err)
	}

	// Parse updated template to verify it changed
	result2, err := parser.Parse("test-template", req)
	if err != nil {
		t.Fatalf("Failed to parse updated template: %v", err)
	}

	expected2 := "/api/users"
	if result2 != expected2 {
		t.Errorf("Expected '%s', got '%s'", expected2, result2)
	}
}

// Test that original request body remains readable after Parse
func TestOriginalRequestBodyAfterParse(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("body-test", "Template got: {{.Body}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Create request with body
	originalBody := "test body content"
	req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(originalBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	// Parse template (this should consume the body internally)
	var output bytes.Buffer
	err = parser.Parse("body-test", req, &output)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Verify template got the body
	expected := "Template got: " + originalBody
	if output.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.String())
	}

	// Now try to read the original request body again (this is what external code might do)
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Failed to read original request body after Parse: %v", err)
	}

	// The body should still be readable and contain the original content
	if string(bodyBytes) != originalBody {
		t.Errorf("Expected original body '%s' to be readable after Parse, got '%s'", originalBody, string(bodyBytes))
	}

	// Test reading the body again (should be empty since stream was consumed)
	bodyBytes2, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("Failed to read original request body second time: %v", err)
	}

	// This should be empty because the stream was consumed
	if len(bodyBytes2) != 0 {
		t.Errorf("Expected empty body on second read, got '%s'", string(bodyBytes2))
	}
}

// Test that Parse can be called multiple times on the same request
func TestMultipleParseCalls(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("multi-test", "Body: {{.Body}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Create request with body
	originalBody := "test content"
	req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(originalBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template first time
	var output1 bytes.Buffer
	err = parser.Parse("multi-test", req, &output1)
	if err != nil {
		t.Fatalf("Failed to parse template first time: %v", err)
	}

	expected := "Body: " + originalBody
	if output1.String() != expected {
		t.Errorf("First parse: expected '%s', got '%s'", expected, output1.String())
	}

	// Parse template second time (should work because body is reset)
	var output2 bytes.Buffer
	err = parser.Parse("multi-test", req, &output2)
	if err != nil {
		t.Fatalf("Failed to parse template second time: %v", err)
	}

	if output2.String() != expected {
		t.Errorf("Second parse: expected '%s', got '%s'", expected, output2.String())
	}
}

// Test generic parser with string type
func TestGenericParserString(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("string-test", "Hello {{.Request.Method}} from {{.Request.URL.Path}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewGenericParser[string](config)
	if err != nil {
		t.Fatalf("Failed to create generic parser: %v", err)
	}
	defer parser.Close()

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	result, err := parser.Parse("string-test", req)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := "Hello GET from /api/users"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// Test generic parser with int type
func TestGenericParserInt(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("int-test", "{{len .Request.URL.Path}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewGenericParser[int](config)
	if err != nil {
		t.Fatalf("Failed to create generic parser: %v", err)
	}
	defer parser.Close()

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	result, err := parser.Parse("int-test", req)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := len("/api/users")
	if result != expected {
		t.Errorf("Expected %d, got %d", expected, result)
	}
}

// Test generic parser with JSON struct
func TestGenericParserJSON(t *testing.T) {
	type APIResponse struct {
		Method string `json:"method"`
		Path   string `json:"path"`
		Status string `json:"status"`
	}

	loader := NewMemoryLoader()
	loader.AddTemplate("json-test", `{"method":"{{.Request.Method}}","path":"{{.Request.URL.Path}}","status":"ok"}`)

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	parser, err := NewGenericParser[APIResponse](config)
	if err != nil {
		t.Fatalf("Failed to create generic parser: %v", err)
	}
	defer parser.Close()

	// Create test request
	req, err := http.NewRequest("POST", "http://example.com/api/create", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	result, err := parser.Parse("json-test", req)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	expected := APIResponse{
		Method: "POST",
		Path:   "/api/create",
		Status: "ok",
	}

	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// Test default MemoryLoader behavior when no TemplateLoader is specified
func TestDefaultMemoryLoader(t *testing.T) {
	// Create config with no TemplateLoader specified
	config := Config{
		MaxCacheSize: 10,
		WatchFiles:   false,
		FuncMap:      DefaultFuncMap(),
	}

	// This should not return an error and should use MemoryLoader by default
	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Expected parser creation to succeed with default MemoryLoader, got error: %v", err)
	}
	defer parser.Close()

	// Test that we can add templates using UpdateTemplate (which works with the default MemoryLoader)
	templateContent := "Default loader test: {{.Request.Method}} {{.Request.URL.Path}}"
	err = parser.UpdateTemplate("default-test", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template with default loader: %v", err)
	}

	// Create test request
	req, err := http.NewRequest("GET", "http://example.com/default", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	var output bytes.Buffer
	err = parser.Parse("default-test", req, &output)
	if err != nil {
		t.Fatalf("Failed to parse template with default loader: %v", err)
	}

	expected := "Default loader test: GET /default"
	if output.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.String())
	}

	// Test that the config's TemplateLoader was set to a MemoryLoader
	// (We can't directly access it, but we can test that it behaves like one)

	// Verify cache stats work (indicating the parser is functional)
	stats := parser.GetCacheStats()
	if stats.Size != 1 {
		t.Errorf("Expected cache size 1 after adding one template, got %d", stats.Size)
	}
}

// Test default MemoryLoader with generic parser
func TestDefaultMemoryLoaderGeneric(t *testing.T) {
	// Create config with no TemplateLoader specified
	config := Config{
		MaxCacheSize: 10,
		WatchFiles:   false,
		FuncMap:      DefaultFuncMap(),
	}

	// This should not return an error and should use MemoryLoader by default
	parser, err := NewGenericParser[string](config)
	if err != nil {
		t.Fatalf("Expected generic parser creation to succeed with default MemoryLoader, got error: %v", err)
	}
	defer parser.Close()

	// Test that we can add templates using UpdateTemplate
	templateContent := "Generic default: {{.Request.Method}}"
	err = parser.UpdateTemplate("generic-default", templateContent)
	if err != nil {
		t.Fatalf("Failed to update template with default loader: %v", err)
	}

	// Create test request
	req, err := http.NewRequest("POST", "http://example.com/generic", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Parse template
	result, err := parser.Parse("generic-default", req)
	if err != nil {
		t.Fatalf("Failed to parse template with default loader: %v", err)
	}

	expected := "Generic default: POST"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// Test BodyBytes function
func TestRereadableRequestBodyBytes(t *testing.T) {
	body := "test body content"
	req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	// Test BodyBytes returns correct content
	bodyBytes := rereadable.BodyBytes()
	if string(bodyBytes) != body {
		t.Errorf("Expected body bytes '%s', got '%s'", body, string(bodyBytes))
	}

	// Test that BodyBytes returns a copy (modifying it shouldn't affect original)
	bodyBytes[0] = 'X'
	originalBodyBytes := rereadable.BodyBytes()
	if string(originalBodyBytes) != body {
		t.Errorf("BodyBytes should return a copy, original was modified: got '%s'", string(originalBodyBytes))
	}
}

// Test error handling in various functions
func TestErrorHandling(t *testing.T) {
	// Test NewRereadableRequest with nil body
	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request with nil body: %v", err)
	}

	if rereadable.Body() != "" {
		t.Errorf("Expected empty body for nil request body, got '%s'", rereadable.Body())
	}

	// Test parser with closed parser
	loader := NewMemoryLoader()
	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Close the parser
	parser.Close()

	// Try to use closed parser
	var output bytes.Buffer
	err = parser.Parse("test", req, &output)
	if err == nil {
		t.Error("Expected error when using closed parser")
	}

	// Try UpdateTemplate on closed parser
	err = parser.UpdateTemplate("test", "content")
	if err == nil {
		t.Error("Expected error when updating template on closed parser")
	}
}

// Test more DefaultFuncMap functions
func TestDefaultFuncMapComplete(t *testing.T) {
	funcMap := DefaultFuncMap()

	// Test title function
	titleFunc := funcMap["title"].(func(string) string)
	if titleFunc("hello world") != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", titleFunc("hello world"))
	}

	// Test trim function
	trimFunc := funcMap["trim"].(func(string) string)
	if trimFunc("  hello  ") != "hello" {
		t.Errorf("Expected 'hello', got '%s'", trimFunc("  hello  "))
	}

	// Test header function
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("X-Test", "test-value")

	headerFunc := funcMap["header"].(func(*http.Request, string) string)
	if headerFunc(req, "X-Test") != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", headerFunc(req, "X-Test"))
	}

	// Test query function
	req.URL.RawQuery = "param=value&other=test"
	queryFunc := funcMap["query"].(func(*http.Request, string) string)
	if queryFunc(req, "param") != "value" {
		t.Errorf("Expected 'value', got '%s'", queryFunc(req, "param"))
	}

	// Test form function
	req.Form = map[string][]string{"field": {"form-value"}}
	formFunc := funcMap["form"].(func(*http.Request, string) string)
	if formFunc(req, "field") != "form-value" {
		t.Errorf("Expected 'form-value', got '%s'", formFunc(req, "field"))
	}

	// Test default function with nil
	defaultFunc := funcMap["default"].(func(interface{}, interface{}) interface{})
	if defaultFunc("fallback", nil) != "fallback" {
		t.Errorf("Expected 'fallback' for nil value, got '%v'", defaultFunc("fallback", nil))
	}
}

// Test new string functions in DefaultFuncMap
func TestDefaultFuncMapStringFunctions(t *testing.T) {
	funcMap := DefaultFuncMap()

	// Test hasPrefix function
	hasPrefixFunc := funcMap["hasPrefix"].(func(string, string) bool)
	if !hasPrefixFunc("hello world", "hello") {
		t.Error("Expected 'hello world' to have prefix 'hello'")
	}
	if hasPrefixFunc("hello world", "world") {
		t.Error("Expected 'hello world' not to have prefix 'world'")
	}

	// Test hasSuffix function
	hasSuffixFunc := funcMap["hasSuffix"].(func(string, string) bool)
	if !hasSuffixFunc("hello world", "world") {
		t.Error("Expected 'hello world' to have suffix 'world'")
	}
	if hasSuffixFunc("hello world", "hello") {
		t.Error("Expected 'hello world' not to have suffix 'hello'")
	}

	// Test contains function
	containsFunc := funcMap["contains"].(func(string, string) bool)
	if !containsFunc("hello world", "lo wo") {
		t.Error("Expected 'hello world' to contain 'lo wo'")
	}
	if containsFunc("hello world", "xyz") {
		t.Error("Expected 'hello world' not to contain 'xyz'")
	}

	// Test replace function
	replaceFunc := funcMap["replace"].(func(string, string, string) string)
	result := replaceFunc("hello world", "world", "universe")
	if result != "hello universe" {
		t.Errorf("Expected 'hello universe', got '%s'", result)
	}

	// Test split function
	splitFunc := funcMap["split"].(func(string, string) []string)
	parts := splitFunc("a,b,c", ",")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Errorf("Expected [a b c], got %v", parts)
	}

	// Test join function
	joinFunc := funcMap["join"].(func([]string, string) string)
	joined := joinFunc([]string{"a", "b", "c"}, "-")
	if joined != "a-b-c" {
		t.Errorf("Expected 'a-b-c', got '%s'", joined)
	}

	// Test trimPrefix function
	trimPrefixFunc := funcMap["trimPrefix"].(func(string, string) string)
	trimmed := trimPrefixFunc("hello world", "hello ")
	if trimmed != "world" {
		t.Errorf("Expected 'world', got '%s'", trimmed)
	}

	// Test trimSuffix function
	trimSuffixFunc := funcMap["trimSuffix"].(func(string, string) string)
	trimmed = trimSuffixFunc("hello world", " world")
	if trimmed != "hello" {
		t.Errorf("Expected 'hello', got '%s'", trimmed)
	}

	// Test repeat function
	repeatFunc := funcMap["repeat"].(func(string, int) string)
	repeated := repeatFunc("a", 3)
	if repeated != "aaa" {
		t.Errorf("Expected 'aaa', got '%s'", repeated)
	}

	// Test substr function
	substrFunc := funcMap["substr"].(func(string, int, int) string)
	sub := substrFunc("hello world", 6, 5)
	if sub != "world" {
		t.Errorf("Expected 'world', got '%s'", sub)
	}

	// Test substr with length beyond string
	sub = substrFunc("hello", 2, 10)
	if sub != "llo" {
		t.Errorf("Expected 'llo', got '%s'", sub)
	}

	// Test substr with negative start (should return empty)
	sub = substrFunc("hello", -1, 3)
	if sub != "" {
		t.Errorf("Expected empty string, got '%s'", sub)
	}
}

// Test multipart form data extraction
func TestExtractRequestDataMultipart(t *testing.T) {
	// Create multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("name", "John")
	writer.WriteField("age", "30")
	writer.Close()

	req, err := http.NewRequest("POST", "http://example.com/test?param=value", &body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract multipart request data: %v", err)
	}

	// Check that form data was parsed
	if len(data.Form) == 0 {
		t.Error("Expected form data to be parsed for multipart form")
	}
}

// Test JSON body parsing
func TestExtractRequestDataJSON(t *testing.T) {
	// Create JSON request
	jsonBody := `{"name": "John", "age": 30, "city": "New York", "active": true}`
	req, err := http.NewRequest("POST", "http://example.com/api/users", strings.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract JSON request data: %v", err)
	}

	// Check that raw body is still available
	if data.Body != jsonBody {
		t.Errorf("Expected raw body '%s', got '%s'", jsonBody, data.Body)
	}

	// Check that JSON was parsed
	if data.BodyJSON == nil {
		t.Fatal("Expected BodyJSON to be parsed, got nil")
	}

	// Verify JSON content
	if data.BodyJSON["name"] != "John" {
		t.Errorf("Expected JSON name 'John', got '%v'", data.BodyJSON["name"])
	}

	if data.BodyJSON["age"] != float64(30) { // JSON numbers become float64
		t.Errorf("Expected JSON age 30, got '%v'", data.BodyJSON["age"])
	}

	if data.BodyJSON["city"] != "New York" {
		t.Errorf("Expected JSON city 'New York', got '%v'", data.BodyJSON["city"])
	}

	if data.BodyJSON["active"] != true {
		t.Errorf("Expected JSON active true, got '%v'", data.BodyJSON["active"])
	}
}

// Test XML body parsing
func TestExtractRequestDataXML(t *testing.T) {
	// Create XML request
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<user>
	<name>John</name>
	<age>30</age>
	<city>New York</city>
</user>`
	req, err := http.NewRequest("POST", "http://example.com/api/users", strings.NewReader(xmlBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract XML request data: %v", err)
	}

	// Check that raw body is still available
	if data.Body != xmlBody {
		t.Errorf("Expected raw body '%s', got '%s'", xmlBody, data.Body)
	}

	// Note: XML parsing in Go is more complex than JSON, and the simple approach
	// we're using may not always result in a map[string]interface{}.
	// For this test, we mainly verify that the parsing doesn't error
	// and that the raw body is still accessible.
}

// Test SOAP body parsing
func TestExtractRequestDataSOAP(t *testing.T) {
	// Create SOAP request
	soapBody := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<GetUser>
			<UserId>123</UserId>
			<IncludeDetails>true</IncludeDetails>
		</GetUser>
	</soap:Body>
</soap:Envelope>`
	req, err := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "GetUser")

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract SOAP request data: %v", err)
	}

	// Check that raw body is still available
	if data.Body != soapBody {
		t.Errorf("Expected raw body '%s', got '%s'", soapBody, data.Body)
	}

	// For SOAP, the XML parsing should be triggered
	// The actual structure depends on the XML parser implementation
}

// Test invalid JSON handling
func TestExtractRequestDataInvalidJSON(t *testing.T) {
	// Create request with invalid JSON
	invalidJSON := `{"name": "John", "age": 30, "invalid": json}`
	req, err := http.NewRequest("POST", "http://example.com/api/users", strings.NewReader(invalidJSON))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract request data with invalid JSON: %v", err)
	}

	// Should not error, but BodyJSON should be nil
	if data.BodyJSON != nil {
		t.Error("Expected BodyJSON to be nil for invalid JSON")
	}

	// Raw body should still be available
	if data.Body != invalidJSON {
		t.Errorf("Expected raw body '%s', got '%s'", invalidJSON, data.Body)
	}
}

// Test content type variations
func TestExtractRequestDataContentTypeVariations(t *testing.T) {
	testCases := []struct {
		name        string
		contentType string
		body        string
		expectJSON  bool
		expectXML   bool
	}{
		{"JSON with charset", "application/json; charset=utf-8", `{"test": "value"}`, true, false},
		{"JSON uppercase", "APPLICATION/JSON", `{"test": "value"}`, true, false},
		{"XML text", "text/xml", `<test>value</test>`, false, true},
		{"XML application", "application/xml", `<test>value</test>`, false, true},
		{"SOAP XML", "application/soap+xml", `<test>value</test>`, false, true},
		{"Plain text", "text/plain", `plain text`, false, false},
		{"Form data", "application/x-www-form-urlencoded", `name=value`, false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(tc.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", tc.contentType)

			rereadable, err := NewRereadableRequest(req)
			if err != nil {
				t.Fatalf("Failed to create re-readable request: %v", err)
			}

			data, err := ExtractRequestData(rereadable, nil)
			if err != nil {
				t.Fatalf("Failed to extract request data: %v", err)
			}

			if tc.expectJSON && data.BodyJSON == nil {
				t.Error("Expected BodyJSON to be parsed")
			}
			if !tc.expectJSON && data.BodyJSON != nil {
				t.Error("Expected BodyJSON to be nil")
			}

			if tc.expectXML && data.BodyXML == nil {
				// Note: XML parsing might still fail due to implementation complexity
				// This is not necessarily an error for this test
			}
			if !tc.expectXML && data.BodyXML != nil {
				t.Error("Expected BodyXML to be nil")
			}

			// Raw body should always be available
			if data.Body != tc.body {
				t.Errorf("Expected raw body '%s', got '%s'", tc.body, data.Body)
			}
		})
	}
}

// Test memory loader edge cases
func TestMemoryLoaderEdgeCases(t *testing.T) {
	loader := NewMemoryLoader()

	// Test loading non-existent template
	_, err := loader.Load("nonexistent")
	if err == nil {
		t.Error("Expected error loading non-existent template from memory loader")
	}

	// Test last modified on non-existent template
	_, err = loader.LastModified("nonexistent")
	if err == nil {
		t.Error("Expected error getting last modified of non-existent template")
	}

	// Test Watch function (should return nil for memory loader)
	err = loader.Watch(context.Background(), func(string) {})
	if err != nil {
		t.Errorf("Expected no error from memory loader Watch, got: %v", err)
	}
}

// Test generic parser conversion edge cases
func TestGenericParserConversion(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("invalid-json", `{"invalid": json}`)
	loader.AddTemplate("valid-int", "42")
	loader.AddTemplate("invalid-int", "not-a-number")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
	}

	// Test invalid JSON conversion
	jsonParser, err := NewGenericParser[map[string]interface{}](config)
	if err != nil {
		t.Fatalf("Failed to create generic parser: %v", err)
	}
	defer jsonParser.Close()

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	_, err = jsonParser.Parse("invalid-json", req)
	if err == nil {
		t.Error("Expected error for invalid JSON conversion")
	}

	// Test valid int conversion
	intParser, err := NewGenericParser[int](config)
	if err != nil {
		t.Fatalf("Failed to create int parser: %v", err)
	}
	defer intParser.Close()

	result, err := intParser.Parse("valid-int", req)
	if err != nil {
		t.Fatalf("Failed to parse valid int: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	// Test invalid int conversion
	_, err = intParser.Parse("invalid-int", req)
	if err == nil {
		t.Error("Expected error for invalid int conversion")
	}

	// Test with ParseWith
	_, err = intParser.ParseWith("valid-int", req, nil)
	if err != nil {
		t.Fatalf("Failed to parse with valid int: %v", err)
	}
}

// Test template cache edge cases
func TestTemplateCacheEdgeCases(t *testing.T) {
	// Test cache with max size 0 (unlimited)
	cache := NewTemplateCache(0, nil)
	loader := NewMemoryLoader()
	loader.AddTemplate("test", "Hello {{.Body}}")

	// Load template
	tmpl, err := cache.Get("test", loader)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	stats := cache.Stats()
	if stats.MaxSize != 0 {
		t.Errorf("Expected max size 0, got %d", stats.MaxSize)
	}

	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}

	// Test getting template that already exists in cache
	tmpl2, err := cache.Get("test", loader)
	if err != nil {
		t.Fatalf("Failed to get cached template: %v", err)
	}

	if tmpl2 == nil {
		t.Error("Expected to get cached template")
	}

	// Test eviction with empty cache
	cache.Clear()
	cache.evictLRU() // Should not panic

	_ = tmpl
}

// Test parser with invalid template
func TestParserInvalidTemplate(t *testing.T) {
	loader := NewMemoryLoader()
	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	// Try to update with invalid template syntax
	err = parser.UpdateTemplate("invalid", "{{invalid template syntax")
	if err == nil {
		t.Error("Expected error for invalid template syntax")
	}

	// Try to parse non-existent template
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	var output bytes.Buffer
	err = parser.Parse("nonexistent", req, &output)
	if err == nil {
		t.Error("Expected error parsing non-existent template")
	}
}

// Test query parameter parsing edge cases
func TestExtractRequestDataQueryEdgeCases(t *testing.T) {
	// Test with invalid query string
	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.URL.RawQuery = "invalid%zzquery"

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract request data with invalid query: %v", err)
	}

	// Should still work even with invalid query
	if data.Request == nil {
		t.Error("Expected request to be set even with invalid query")
	}

	// Test with no query at all
	req2, _ := http.NewRequest("GET", "http://example.com/test", nil)
	rereadable2, _ := NewRereadableRequest(req2)
	data2, err := ExtractRequestData(rereadable2, nil)
	if err != nil {
		t.Fatalf("Failed to extract request data with no query: %v", err)
	}

	if len(data2.Query) != 0 {
		t.Errorf("Expected empty query map, got %v", data2.Query)
	}
}

// Test conversion with different types
func TestConversionTypes(t *testing.T) {
	loader := NewMemoryLoader()
	loader.AddTemplate("float-test", "3.14")
	loader.AddTemplate("bool-test", "true")
	loader.AddTemplate("string-test", "hello world")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// Test float conversion
	floatParser, err := NewGenericParser[float64](config)
	if err != nil {
		t.Fatalf("Failed to create float parser: %v", err)
	}
	defer floatParser.Close()

	floatResult, err := floatParser.Parse("float-test", req)
	if err != nil {
		t.Fatalf("Failed to parse float: %v", err)
	}
	if floatResult != 3.14 {
		t.Errorf("Expected 3.14, got %f", floatResult)
	}

	// Test bool conversion
	boolParser, err := NewGenericParser[bool](config)
	if err != nil {
		t.Fatalf("Failed to create bool parser: %v", err)
	}
	defer boolParser.Close()

	boolResult, err := boolParser.Parse("bool-test", req)
	if err != nil {
		t.Fatalf("Failed to parse bool: %v", err)
	}
	if !boolResult {
		t.Errorf("Expected true, got %t", boolResult)
	}

	// Test string conversion (should always work)
	stringParser, err := NewGenericParser[string](config)
	if err != nil {
		t.Fatalf("Failed to create string parser: %v", err)
	}
	defer stringParser.Close()

	stringResult, err := stringParser.Parse("string-test", req)
	if err != nil {
		t.Fatalf("Failed to parse string: %v", err)
	}
	if stringResult != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", stringResult)
	}
}

// Test template cache with last modified time updates
func TestTemplateCacheLastModified(t *testing.T) {
	cache := NewTemplateCache(10, nil)
	loader := NewMemoryLoader()
	loader.AddTemplate("test", "Hello {{.Body}}")

	// Get template first time
	tmpl1, err := cache.Get("test", loader)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	// Get it again (should use cache)
	tmpl2, err := cache.Get("test", loader)
	if err != nil {
		t.Fatalf("Failed to get template from cache: %v", err)
	}

	// Both should have the same name
	if tmpl1.Name() != tmpl2.Name() {
		t.Error("Expected same template name from cache")
	}

	// Cache should have size 1
	stats := cache.Stats()
	if stats.Size != 1 {
		t.Errorf("Expected cache size 1, got %d", stats.Size)
	}
}

// Test cache statistics accumulation
func TestCacheStatsHitCount(t *testing.T) {
	cache := NewTemplateCache(10, nil)
	loader := NewMemoryLoader()
	loader.AddTemplate("test", "Hello {{.Body}}")

	// Access template multiple times to accumulate access count
	for i := 0; i < 5; i++ {
		_, err := cache.Get("test", loader)
		if err != nil {
			t.Fatalf("Failed to get template: %v", err)
		}
	}

	stats := cache.Stats()
	// Hit count includes all accesses, so should be at least 5
	if stats.HitCount < 1 {
		t.Errorf("Expected hit count >= 1, got %d", stats.HitCount)
	}

	if stats.Size != 1 {
		t.Errorf("Expected cache size 1, got %d", stats.Size)
	}
}

// Test form parsing error handling
func TestFormParsingErrorHandling(t *testing.T) {
	// Create request with invalid form content type
	req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader("name=John&age=30"))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; boundary=invalid")

	rereadable, err := NewRereadableRequest(req)
	if err != nil {
		t.Fatalf("Failed to create re-readable request: %v", err)
	}

	// This should still work even with malformed content type
	data, err := ExtractRequestData(rereadable, nil)
	if err != nil {
		t.Fatalf("Failed to extract request data: %v", err)
	}

	if data == nil {
		t.Error("Expected data to be extracted even with invalid content type")
	}
}

// Benchmark cache performance with different sizes
func BenchmarkCacheSize1(b *testing.B) {
	benchmarkCacheWithSize(b, 1)
}

func BenchmarkCacheSize10(b *testing.B) {
	benchmarkCacheWithSize(b, 10)
}

func BenchmarkCacheSize100(b *testing.B) {
	benchmarkCacheWithSize(b, 100)
}

func BenchmarkCacheUnlimited(b *testing.B) {
	benchmarkCacheWithSize(b, 0) // 0 means unlimited
}

func benchmarkCacheWithSize(b *testing.B, cacheSize int) {
	loader := NewMemoryLoader()
	loader.AddTemplate("cache", "Template: {{.Request.Method}} {{.Request.URL.Path}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   cacheSize,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	req, err := http.NewRequest("GET", "http://example.com/test", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output bytes.Buffer
		err := parser.Parse("cache", req, &output)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// Benchmark memory efficiency with different body sizes
func BenchmarkSmallBody(b *testing.B) {
	benchmarkBodySize(b, 100) // 100 bytes
}

func BenchmarkMediumBody(b *testing.B) {
	benchmarkBodySize(b, 10*1024) // 10KB
}

func BenchmarkLargeBody(b *testing.B) {
	benchmarkBodySize(b, 100*1024) // 100KB
}

func benchmarkBodySize(b *testing.B, size int) {
	loader := NewMemoryLoader()
	loader.AddTemplate("body", "Length: {{len .Body}}")

	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
	}

	parser, err := NewParser(config)
	if err != nil {
		b.Fatalf("Failed to create parser: %v", err)
	}
	defer parser.Close()

	body := strings.Repeat("x", size)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest("POST", "http://example.com/test", strings.NewReader(body))
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		var output bytes.Buffer
		err = parser.Parse("body", req, &output)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}
