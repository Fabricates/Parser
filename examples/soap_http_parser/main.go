package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fabricates/parser"
)

func main() {
	// Create a memory-based template loader
	loader := parser.NewMemoryLoader()

	// Load the request parser template from file
	templateContent, err := os.ReadFile("request_parser.tmpl")
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	loader.AddTemplate("request_parser", string(templateContent))

	// Create custom function map with additional string functions
	funcMap := parser.DefaultFuncMap()
	funcMap["hasPrefix"] = func(s, prefix string) bool {
		return strings.HasPrefix(s, prefix)
	}
	funcMap["contains"] = func(s, substr string) bool {
		return strings.Contains(s, substr)
	}

	// Create parser configuration
	config := parser.Config{
		TemplateLoader: loader,
		MaxCacheSize:   10,
		WatchFiles:     false,
		FuncMap:        funcMap,
	}

	// Create parser
	p, err := parser.NewParser(config)
	if err != nil {
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Test cases demonstrating the template

	// Example 1: SOAP request with MESRecipeTurnOff
	fmt.Println("=== Example 1: SOAP MESRecipeTurnOff Request ===")
	soapBody1 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<MESRecipeTurnOff>
			<param>value</param>
		</MESRecipeTurnOff>
	</soap:Body>
</soap:Envelope>`
	req1, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapBody1))
	req1.Header.Set("Content-Type", "application/xml")

	var output1 bytes.Buffer
	err = p.Parse("request_parser", req1, &output1)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output1.String())

	// Example 2: SOAP request with Recommend_Request
	fmt.Println("\n=== Example 2: SOAP Recommend_Request ===")
	soapBody2 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<Recommend_Request>
			<objRequest>
				<CONTEXT_INFO>
					<ROUTEGROUP>RG123</ROUTEGROUP>
					<CONTROLJOBTYPE>CJ456</CONTROLJOBTYPE>
					<LAYER>L789</LAYER>
				</CONTEXT_INFO>
			</objRequest>
		</Recommend_Request>
	</soap:Body>
</soap:Envelope>`
	req2, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapBody2))
	req2.Header.Set("Content-Type", "text/xml")

	var output2 bytes.Buffer
	err = p.Parse("request_parser", req2, &output2)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output2.String())

	// Example 3: HTTP JSON request with ASML-LIS header
	fmt.Println("\n=== Example 3: HTTP JSON ASML-LIS Request ===")
	jsonBody3 := `{"opno": "OP123", "other": "data"}`
	req3, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonBody3))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("x-src-system", "ASML-LIS")

	var output3 bytes.Buffer
	err = p.Parse("request_parser", req3, &output3)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output3.String())

	// Example 4: HTTP JSON request with chamber unlock data
	fmt.Println("\n=== Example 4: HTTP JSON Chamber Unlock Request ===")
	jsonBody4 := `{
		"prodspecId": "PROD123",
		"routeId": "ROUTE456",
		"opeNo": "OP789",
		"lotId": "LOT001",
		"recipeId": "RCP123",
		"eqpId": "EQP456",
		"chamberId": "CH789"
	}`
	req4, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonBody4))
	req4.Header.Set("Content-Type", "application/json")

	var output4 bytes.Buffer
	err = p.Parse("request_parser", req4, &output4)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output4.String())

	// Example 5: Unknown request type (fallback)
	fmt.Println("\n=== Example 5: Unknown Request Type ===")
	req5, _ := http.NewRequest("GET", "http://example.com/unknown", nil)
	req5.Header.Set("Content-Type", "text/plain")

	var output5 bytes.Buffer
	err = p.Parse("request_parser", req5, &output5)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output5.String())

	// Example 6: SOAP request with GetUsedLotInfo and headers
	fmt.Println("\n=== Example 6: SOAP GetUsedLotInfo with Headers ===")
	soapBody6 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<GetUsedLotInfo>
			<result_xml>true</result_xml>
		</GetUsedLotInfo>
	</soap:Body>
</soap:Envelope>`
	req6, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapBody6))
	req6.Header.Set("Content-Type", "application/xml")
	req6.Header.Set("x-eap-tool", "TOOL123")
	req6.Header.Set("x-eap-product-group", "PG456")
	req6.Header.Set("x-eap-operation-no", "OP789")

	var output6 bytes.Buffer
	err = p.Parse("request_parser", req6, &output6)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Println(output6.String())

	fmt.Println("\n=== Parser Statistics ===")
	stats := p.GetCacheStats()
	fmt.Printf("Cache Size: %d/%d\n", stats.Size, stats.MaxSize)
	fmt.Printf("Total Cache Hits: %d\n", stats.HitCount)
}