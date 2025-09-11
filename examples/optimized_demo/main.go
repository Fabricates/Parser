package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/fabricates/parser"
)

func main() {
	fmt.Println("=== Optimized Extract with Provided Body Demo ===")

	// Create parser with default settings
	config := parser.Config{
		MaxCacheSize: 10,
		WatchFiles:   false,
	}

	p, err := parser.NewParser(config)
	if err != nil {
		log.Fatal("Failed to create parser:", err)
	}
	defer p.Close()

	// Create a request with empty body initially
	req, err := http.NewRequest("POST", "http://api.example.com/users", nil)
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Provide external body data
	externalBody := []byte(`{"name": "External Body", "provided": true, "optimized": "yes"}`)

	// Extract using provided body (optimized - no ReadAll called on request body)
	fmt.Println("1. Extract with provided body (optimized):")
	requestData, err := p.Extract(req, externalBody)
	if err != nil {
		log.Fatal("Failed to extract with provided body:", err)
	}

	fmt.Printf("  Body from provided data: %s\n", requestData.Body)
	fmt.Printf("  Parsed JSON name: %v\n", requestData.BodyJSON["name"])
	fmt.Printf("  Parsed JSON optimized: %v\n", requestData.BodyJSON["optimized"])

	// Extract without provided body (will read from request, which is empty)
	fmt.Println("\n2. Extract without provided body:")
	requestData2, err := p.Extract(req)
	if err != nil {
		log.Fatal("Failed to extract without provided body:", err)
	}

	fmt.Printf("  Body from request: '%s' (empty as expected)\n", requestData2.Body)
	if requestData2.BodyJSON == nil {
		fmt.Println("  BodyJSON: nil (no JSON to parse)")
	}

	fmt.Println("\n3. Comparison - body independence:")
	fmt.Printf("  First extraction body length: %d\n", len(requestData.Body))
	fmt.Printf("  Second extraction body length: %d\n", len(requestData2.Body))
	fmt.Println("  ✓ Provided body doesn't affect original request body")

	fmt.Println("\n=== Performance Benefits ===")
	fmt.Println("✓ When body is provided:")
	fmt.Println("  - No io.ReadAll() call on request body")
	fmt.Println("  - No resetBody() calls needed")
	fmt.Println("  - Direct use of provided data")
	fmt.Println("✓ When body is not provided:")
	fmt.Println("  - Normal ReadAll() and reset behavior")
	fmt.Println("  - Backwards compatible with existing code")
}
