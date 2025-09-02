package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabricates/parser"
)

func main() {
	// Get the template file path
	templatePath := filepath.Join("..", "..", "templates", "soap_request_parser.tmpl")
	
	// Read the template content
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	// Create a memory-based template loader
	loader := parser.NewMemoryLoader()
	loader.AddTemplate("soap_parser", string(templateContent))

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
		log.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Example 1: SOAP XML request with MESRecipeTurnOff
	fmt.Println("=== Example 1: SOAP XML - MESRecipeTurnOff ===")
	soapXML1 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <MESRecipeTurnOff>
      <param>value</param>
    </MESRecipeTurnOff>
  </soap:Body>
</soap:Envelope>`

	req1, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML1))
	req1.Header.Set("Content-Type", "text/xml")

	var output1 bytes.Buffer
	err = p.Parse("soap_parser", req1, &output1)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output1.String())

	// Example 2: SOAP XML request with Recommend_Request
	fmt.Println("\n\n=== Example 2: SOAP XML - Recommend_Request ===")
	soapXML2 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Recommend_Request>
      <objRequest>
        <CONTEXT_INFO>
          <ROUTEGROUP>RG001</ROUTEGROUP>
          <CONTROLJOBTYPE>CTRL_JOB_1</CONTROLJOBTYPE>
          <LAYER>Layer_A</LAYER>
        </CONTEXT_INFO>
      </objRequest>
    </Recommend_Request>
  </soap:Body>
</soap:Envelope>`

	req2, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML2))
	req2.Header.Set("Content-Type", "application/xml")

	var output2 bytes.Buffer
	err = p.Parse("soap_parser", req2, &output2)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output2.String())

	// Example 3: JSON request from ASML-LIS
	fmt.Println("\n\n=== Example 3: JSON - ASML-LIS Request ===")
	jsonPayload := `{"opno": "OP123", "lotId": "LOT456"}`

	req3, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("x-src-system", "ASML-LIS")

	var output3 bytes.Buffer
	err = p.Parse("soap_parser", req3, &output3)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output3.String())

	// Example 4: JSON request for EH unLockChamber
	fmt.Println("\n\n=== Example 4: JSON - EH unLockChamber ===")
	jsonPayload4 := `{
		"prodspecId": "PROD001",
		"routeId": "ROUTE001", 
		"opeNo": "OP001",
		"lotId": "LOT001",
		"recipeId": "RECIPE001",
		"eqpId": "EQP001",
		"chamberId": "CHAMBER001"
	}`

	req4, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload4))
	req4.Header.Set("Content-Type", "application/json")

	var output4 bytes.Buffer
	err = p.Parse("soap_parser", req4, &output4)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output4.String())

	// Example 5: Unknown request type
	fmt.Println("\n\n=== Example 5: Unknown Request Type ===")
	req5, _ := http.NewRequest("GET", "http://example.com/unknown", nil)
	req5.Header.Set("Content-Type", "text/plain")

	var output5 bytes.Buffer
	err = p.Parse("soap_parser", req5, &output5)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output5.String())

	fmt.Println("\n\n=== Parser Statistics ===")
	stats := p.GetCacheStats()
	fmt.Printf("Cache Size: %d/%d\n", stats.Size, stats.MaxSize)
	fmt.Printf("Total Cache Hits: %d\n", stats.HitCount)
}