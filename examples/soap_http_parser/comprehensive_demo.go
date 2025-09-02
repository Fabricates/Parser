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

// Test helper function to run a test case
func runTestCase(p parser.Parser, name, method, url, body, contentType string, headers map[string]string) {
	fmt.Printf("=== %s ===\n", name)
	
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	
	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	var output bytes.Buffer
	err := p.Parse("request_parser", req, &output)
	if err != nil {
		log.Fatalf("Failed to parse template for %s: %v", name, err)
	}
	fmt.Println(output.String())
	fmt.Println()
}

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

	// Test cases demonstrating various SOAP request types
	
	runTestCase(p, "SOAP MESRecipeTurnOff Request", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<MESRecipeTurnOff>
			<param>value</param>
		</MESRecipeTurnOff>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "SOAP Recommend_Request with Context", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
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
</soap:Envelope>`, "text/xml", nil)

	runTestCase(p, "SOAP Metrology_Request", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<Metrology_Request>
			<objRequest>
				<CONTEXT_INFO>
					<ROUTEGROUP>MET_RG</ROUTEGROUP>
					<CONTROLJOBTYPE>MET_CJ</CONTROLJOBTYPE>
					<LAYER>MET_L</LAYER>
				</CONTEXT_INFO>
			</objRequest>
		</Metrology_Request>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "SOAP Used_Request", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<Used_Request>
			<objRequest>
				<CONTEXT_INFO>
					<ROUTEGROUP>USED_RG</ROUTEGROUP>
					<CONTROLJOBTYPE>USED_CJ</CONTROLJOBTYPE>
					<LAYER>USED_L</LAYER>
				</CONTEXT_INFO>
			</objRequest>
		</Used_Request>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "SOAP R2R_Object_Metrology (DISPATCHING)", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<R2R_Object_Metrology>
			<data>content</data>
		</R2R_Object_Metrology>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "SOAP GetUsedLotInfo with result_xml", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<GetUsedLotInfo>
			<result_xml>true</result_xml>
		</GetUsedLotInfo>
	</soap:Body>
</soap:Envelope>`, "application/xml", map[string]string{
		"x-eap-tool": "TOOL123",
		"x-eap-product-group": "PG456", 
		"x-eap-operation-no": "OP789",
	})

	runTestCase(p, "SOAP GetUsedLotInfo with ProductID (EAP)", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<GetUsedLotInfo>
			<ProductID>PROD123</ProductID>
		</GetUsedLotInfo>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "SOAP AMAT_Platen_Control (CMP)", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<AMAT_Platen_Control>
			<control>data</control>
		</AMAT_Platen_Control>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	// HTTP JSON test cases

	runTestCase(p, "HTTP JSON ASML-LIS Request", "POST", "http://example.com/api",
		`{"opno": "OP123", "other": "data"}`, "application/json", map[string]string{
		"x-src-system": "ASML-LIS",
	})

	runTestCase(p, "HTTP JSON Chamber Unlock Request", "POST", "http://example.com/api",
		`{
		"prodspecId": "PROD123",
		"routeId": "ROUTE456",
		"opeNo": "OP789",
		"lotId": "LOT001",
		"recipeId": "RCP123",
		"eqpId": "EQP456",
		"chamberId": "CH789"
	}`, "application/json", nil)

	runTestCase(p, "HTTP JSON Incomplete Chamber Data", "POST", "http://example.com/api",
		`{
		"prodspecId": "PROD123",
		"routeId": "ROUTE456",
		"opeNo": "OP789"
	}`, "application/json", nil)

	// Edge cases

	runTestCase(p, "Unknown Content Type", "GET", "http://example.com/unknown",
		`some plain text content`, "text/plain", nil)

	runTestCase(p, "Unknown SOAP Request", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<UnknownRequest>
			<param>value</param>
		</UnknownRequest>
	</soap:Body>
</soap:Envelope>`, "application/xml", nil)

	runTestCase(p, "Case Sensitivity Test - XML", "POST", "http://example.com/soap",
		`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
	<soap:Body>
		<LotStartTime_For_Scanner>
			<param>value</param>
		</LotStartTime_For_Scanner>
	</soap:Body>
</soap:Envelope>`, "APPLICATION/XML", nil)

	fmt.Println("=== Parser Statistics ===")
	stats := p.GetCacheStats()
	fmt.Printf("Cache Size: %d/%d\n", stats.Size, stats.MaxSize)
	fmt.Printf("Total Cache Hits: %d\n", stats.HitCount)
}