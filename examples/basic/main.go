package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Fabricates/Parser"
)

func main() {
	// Create a memory-based template loader for this example
	loader := parser.NewMemoryLoader()
	
	// Add some example templates
	loader.AddTemplate("greeting", `
Hello {{.Request.Method}} request!
URL: {{.Request.URL.Path}}
User-Agent: {{index .Headers "User-Agent" 0}}
{{if .Query.name}}Name: {{index .Query "name" 0}}{{end}}
{{if .Body}}Body: {{.Body}}{{end}}
{{if .Custom}}Custom data: {{.Custom}}{{end}}
`)
	
	loader.AddTemplate("simple", "Method: {{.Request.Method}}, Path: {{.Request.URL.Path}}")
	
	// Create parser configuration
	config := parser.Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false, // Not applicable for memory loader
		FuncMap:        parser.DefaultFuncMap(),
	}
	
	// Create parser
	p, err := parser.NewParser(config)
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()
	
	// Example 1: Simple GET request
	fmt.Println("=== Example 1: Simple GET request ===")
	req1, _ := http.NewRequest("GET", "http://example.com/api/users?name=John", nil)
	req1.Header.Set("User-Agent", "Parser-Example/1.0")
	
	var output1 bytes.Buffer
	err = p.Parse("greeting", req1, &output1)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output1.String())
	
	// Example 2: POST request with body
	fmt.Println("\n=== Example 2: POST request with body ===")
	req2, _ := http.NewRequest("POST", "http://example.com/api/users", strings.NewReader("username=jane&email=jane@example.com"))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("User-Agent", "Parser-Example/1.0")
	
	var output2 bytes.Buffer
	err = p.Parse("greeting", req2, &output2)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output2.String())
	
	// Example 3: Using custom data
	fmt.Println("\n=== Example 3: With custom data ===")
	req3, _ := http.NewRequest("GET", "http://example.com/dashboard", nil)
	req3.Header.Set("User-Agent", "Parser-Example/1.0")
	
	customData := map[string]interface{}{
		"user_id": 123,
		"role":    "admin",
	}
	
	var output3 bytes.Buffer
	err = p.ParseWithData("greeting", req3, customData, &output3)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output3.String())
	
	// Example 4: Simple template
	fmt.Println("\n=== Example 4: Simple template ===")
	req4, _ := http.NewRequest("PUT", "http://example.com/api/users/123", nil)
	
	var output4 bytes.Buffer
	err = p.Parse("simple", req4, &output4)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output4.String())
	
	fmt.Println("\n\n=== Parser Statistics ===")
	stats := p.GetCacheStats()
	fmt.Printf("Cache Size: %d/%d\n", stats.Size, stats.MaxSize)
	fmt.Printf("Total Cache Hits: %d\n", stats.HitCount)
}