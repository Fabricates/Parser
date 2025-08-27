package parser

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

// Test file system loader (requires temp directory)
func TestFileSystemLoader(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test template file
	templatePath := filepath.Join(tempDir, "test.tmpl")
	templateContent := "Template: {{.Body}}"
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Create loader
	loader := NewFileSystemLoader(tempDir, ".tmpl", false)

	// Test loading
	content, err := loader.Load("test")
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if content != templateContent {
		t.Errorf("Expected '%s', got '%s'", templateContent, content)
	}

	// Test listing
	names, err := loader.List()
	if err != nil {
		t.Fatalf("Failed to list templates: %v", err)
	}

	if len(names) != 1 || names[0] != "test" {
		t.Errorf("Expected ['test'], got %v", names)
	}

	// Test last modified
	modTime, err := loader.LastModified("test")
	if err != nil {
		t.Fatalf("Failed to get last modified: %v", err)
	}

	if modTime.IsZero() {
		t.Error("Expected non-zero modification time")
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
	err = parser.UpdateTemplate("test-template", templateContent, "hash123")
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
	err = parser.UpdateTemplate("test-template", newContent, "hash456")
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
	err = parser.UpdateTemplate("default-test", templateContent, "hash123")
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
	err = parser.UpdateTemplate("generic-default", templateContent, "hash456")
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
