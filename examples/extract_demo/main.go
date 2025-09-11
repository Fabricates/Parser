package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/fabricates/parser"
)

func main() {
	fmt.Println("=== Extract Method Demo ===")

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

	// Create a POST request with JSON body
	jsonBody := `{"name": "John Doe", "email": "john@example.com", "age": 30}`
	req, err := http.NewRequest("POST", "http://api.example.com/users?source=web&ref=signup", strings.NewReader(jsonBody))
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "MyApp/1.0")

	// Custom data
	customData := map[string]interface{}{
		"user_id":    "12345",
		"session_id": "abcdef",
		"timestamp":  1694675400,
	}

	// Extract RequestData without parsing any template
	requestData, err := p.Extract(req, customData)
	if err != nil {
		log.Fatal("Failed to extract request data:", err)
	}

	// Display extracted data
	fmt.Printf("Method: %s\n", requestData.Request.Method)
	fmt.Printf("URL: %s\n", requestData.Request.URL.String())
	fmt.Printf("Body: %s\n", requestData.Body)
	fmt.Printf("Content-Type: %s\n", requestData.Headers["Content-Type"][0])
	fmt.Printf("Authorization: %s\n", requestData.Headers["Authorization"][0])

	// Show parsed JSON
	if requestData.BodyJSON != nil {
		fmt.Println("\nParsed JSON:")
		for key, value := range requestData.BodyJSON {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Show query parameters
	fmt.Println("\nQuery Parameters:")
	for key, values := range requestData.Query {
		fmt.Printf("  %s: %v\n", key, values)
	}

	// Show custom data
	fmt.Println("\nCustom Data:")
	if customMap, ok := requestData.Custom.(map[string]interface{}); ok {
		for key, value := range customMap {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	fmt.Println("\n=== Generic Parser Extract Demo ===")

	// Demo with generic parser
	genericParser, err := parser.NewGenericParser[string](config)
	if err != nil {
		log.Fatal("Failed to create generic parser:", err)
	}
	defer genericParser.Close()

	// Extract using generic parser (same method, same result)
	requestData2, err := genericParser.Extract(req, customData)
	if err != nil {
		log.Fatal("Failed to extract request data with generic parser:", err)
	}

	fmt.Printf("Generic parser extracted method: %s\n", requestData2.Request.Method)
	fmt.Printf("Generic parser extracted URL: %s\n", requestData2.Request.URL.String())
}
