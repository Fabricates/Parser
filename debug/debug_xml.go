package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/fabricates/parser"
)

func main() {
	// Create a memory-based template loader
	loader := parser.NewMemoryLoader()
	
	// Simple debug template to see XML structure for Recommend_Request
	debugTemplate := `{{- if .BodyXML -}}
Full BodyXML: {{ toJson .BodyXML }}

{{- $envelope := index .BodyXML "Envelope" -}}
{{- if $envelope -}}
Envelope found: {{ toJson $envelope }}

{{- $body := index $envelope "Body" -}}
{{- if $body -}}
Body found: {{ toJson $body }}

{{- if hasXMLElement $body "Recommend_Request" -}}
Has Recommend_Request: true
{{- $req := index $body "Recommend_Request" -}}
Recommend_Request content: {{ toJson $req }}

{{- if $req -}}
{{- $objRequest := index $req "objRequest" -}}
{{- if $objRequest -}}
objRequest found: {{ toJson $objRequest }}

{{- $contextInfo := index $objRequest "CONTEXT_INFO" -}}
{{- if $contextInfo -}}
CONTEXT_INFO found: {{ toJson $contextInfo }}

ROUTEGROUP: {{ xmlText $contextInfo "ROUTEGROUP" }}
CONTROLJOBTYPE: {{ xmlText $contextInfo "CONTROLJOBTYPE" }}
LAYER: {{ xmlText $contextInfo "LAYER" }}
{{- else -}}
CONTEXT_INFO not found
{{- end -}}
{{- else -}}
objRequest not found
{{- end -}}
{{- end -}}
{{- else -}}
Has Recommend_Request: false
{{- end -}}

{{- else -}}
Body not found
{{- end -}}
{{- else -}}
Envelope not found
{{- end -}}
{{- else -}}
BodyXML not found
{{- end -}}`

	loader.AddTemplate("debug_recommend", debugTemplate)

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

	// Test XML parsing with Recommend_Request
	fmt.Println("=== Debug: Recommend_Request XML Structure ===")
	soapXML := `<?xml version="1.0" encoding="UTF-8"?>
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

	req, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML))
	req.Header.Set("Content-Type", "text/xml")

	var output bytes.Buffer
	err = p.Parse("debug_recommend", req, &output)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}
	fmt.Print(output.String())
}