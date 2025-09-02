package parser

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Test data structure for expected results
type ParseResult struct {
	Module        string `json:"module"`
	Type          string `json:"type"`
	RouteGroup    string `json:"routegroup,omitempty"`
	Controller    string `json:"controller,omitempty"`
	Layer         string `json:"layer,omitempty"`
	Tool          string `json:"tool,omitempty"`
	ProductGroup  string `json:"productGroup,omitempty"`
	OperationNo   string `json:"operationNo,omitempty"`
}

func TestSOAPRequestParser(t *testing.T) {
	// Load the template
	loader := NewMemoryLoader()
	templateContent := `{{- /* Equivalent template for the JavaScript SOAP request parser */ -}}
{{- $contentType := header .Request "Content-Type" -}}
{{- $contentTypeLower := lower $contentType -}}
{{- $result := dict -}}

{{- /* Determine request type based on content type */ -}}
{{- if or (contains $contentTypeLower "application/xml") (contains $contentTypeLower "text/xml") -}}
  {{- /* XML/SOAP Request Processing */ -}}
  {{- if .BodyXML -}}
    {{- $envelope := index .BodyXML "Envelope" -}}
    {{- if $envelope -}}
      {{- $body := index $envelope "Body" -}}
      {{- if $body -}}
        {{- /* Check for MESRecipeTurnOff */ -}}
        {{- if hasXMLElement $body "MESRecipeTurnOff" -}}
          {{- $result = dict "module" "*" "type" "MESRecipeTurnOff" -}}
        {{- /* Check for Recommend_Request */ -}}
        {{- else if hasXMLElement $body "Recommend_Request" -}}
          {{- $result = dict "module" "PH" "type" "REC" -}}
          {{- $routegroup := index .BodyXML "Envelope/Body/Recommend_Request/objRequest/CONTEXT_INFO/ROUTEGROUP" -}}
          {{- if $routegroup -}}
            {{- $result = merge $result (dict "routegroup" $routegroup) -}}
          {{- end -}}
          {{- $controller := index .BodyXML "Envelope/Body/Recommend_Request/objRequest/CONTEXT_INFO/CONTROLJOBTYPE" -}}
          {{- if $controller -}}
            {{- $result = merge $result (dict "controller" $controller) -}}
          {{- end -}}
          {{- $layer := index .BodyXML "Envelope/Body/Recommend_Request/objRequest/CONTEXT_INFO/LAYER" -}}
          {{- if $layer -}}
            {{- $result = merge $result (dict "layer" $layer) -}}
          {{- end -}}
        {{- /* Check for Metrology_Request */ -}}
        {{- else if hasXMLElement $body "Metrology_Request" -}}
          {{- $result = dict "module" "PH" "type" "MET" -}}
          {{- $routegroup := index .BodyXML "Envelope/Body/Metrology_Request/objRequest/CONTEXT_INFO/ROUTEGROUP" -}}
          {{- if $routegroup -}}
            {{- $result = merge $result (dict "routegroup" $routegroup) -}}
          {{- end -}}
          {{- $controller := index .BodyXML "Envelope/Body/Metrology_Request/objRequest/CONTEXT_INFO/CONTROLJOBTYPE" -}}
          {{- if $controller -}}
            {{- $result = merge $result (dict "controller" $controller) -}}
          {{- end -}}
          {{- $layer := index .BodyXML "Envelope/Body/Metrology_Request/objRequest/CONTEXT_INFO/LAYER" -}}
          {{- if $layer -}}
            {{- $result = merge $result (dict "layer" $layer) -}}
          {{- end -}}
        {{- /* Check for Used_Request */ -}}
        {{- else if hasXMLElement $body "Used_Request" -}}
          {{- $result = dict "module" "PH" "type" "USED" -}}
          {{- $routegroup := index .BodyXML "Envelope/Body/Used_Request/objRequest/CONTEXT_INFO/ROUTEGROUP" -}}
          {{- if $routegroup -}}
            {{- $result = merge $result (dict "routegroup" $routegroup) -}}
          {{- end -}}
          {{- $controller := index .BodyXML "Envelope/Body/Used_Request/objRequest/CONTEXT_INFO/CONTROLJOBTYPE" -}}
          {{- if $controller -}}
            {{- $result = merge $result (dict "controller" $controller) -}}
          {{- end -}}
          {{- $layer := index .BodyXML "Envelope/Body/Used_Request/objRequest/CONTEXT_INFO/LAYER" -}}
          {{- if $layer -}}
            {{- $result = merge $result (dict "layer" $layer) -}}
          {{- end -}}
        {{- /* Check for other PH module requests */ -}}
        {{- else if hasXMLElement $body "LotStartTime_For_Scanner" -}}
          {{- $result = dict "module" "PH" "type" "LotStartTime_For_Scanner" -}}
        {{- else if hasXMLElement $body "LotStartTime_For_YieldStarAndKLA" -}}
          {{- $result = dict "module" "PH" "type" "LotStartTime_For_YieldStarAndKLA" -}}
        {{- else if hasXMLElement $body "GetUsedExpParameters" -}}
          {{- $result = dict "module" "PH" "type" "GetUsedExpParameters" -}}
        {{- else if hasXMLElement $body "GetUsedLotInfoTest" -}}
          {{- $result = dict "module" "PH" "type" "GetUsedLotInfoTest" -}}
        {{- else if hasXMLElement $body "ResendToolOffsets" -}}
          {{- $result = dict "module" "PH" "type" "ResendToolOffsets" -}}
        {{- else if hasXMLElement $body "PackingLotEnd" -}}
          {{- $result = dict "module" "PH" "type" "PackingLotEnd" -}}
        {{- /* Check for Non-PH dispatching requests */ -}}
        {{- else if hasXMLElement $body "R2R_Object_Metrology" -}}
          {{- $result = dict "module" "non-PH" "type" "DISPATCHING" -}}
        {{- else if hasXMLElement $body "R2R_Object_Recommend" -}}
          {{- $result = dict "module" "non-PH" "type" "DISPATCHING" -}}
        {{- else if hasXMLElement $body "R2R_Object_Used" -}}
          {{- $result = dict "module" "non-PH" "type" "DISPATCHING" -}}
        {{- /* Check for GetUsedLotInfo with conditional logic */ -}}
        {{- else if hasXMLElement $body "GetUsedLotInfo" -}}
          {{- $req := index $body "GetUsedLotInfo" -}}
          {{- if $req -}}
            {{- if hasXMLElement $req "result_xml" -}}
              {{- $result = dict "module" "PH" "type" "GetUsedLotInfo" -}}
              {{- $tool := header .Request "x-eap-tool" -}}
              {{- if $tool -}}
                {{- $result = merge $result (dict "tool" $tool) -}}
              {{- end -}}
              {{- $productGroup := header .Request "x-eap-product-group" -}}
              {{- if $productGroup -}}
                {{- $result = merge $result (dict "productGroup" $productGroup) -}}
              {{- end -}}
              {{- $operationNo := header .Request "x-eap-operation-no" -}}
              {{- if $operationNo -}}
                {{- $result = merge $result (dict "operationNo" $operationNo) -}}
              {{- end -}}
            {{- else if hasXMLElement $req "ProductID" -}}
              {{- $result = dict "module" "PH" "type" "EAPGetUsedLotInfo" -}}
            {{- end -}}
          {{- end -}}
        {{- else if hasXMLElement $body "WLC_EAP_REQUEST" -}}
          {{- $result = dict "module" "PH" "type" "WLC_EAP_REQUEST" -}}
        {{- /* Check for CMP module requests */ -}}
        {{- else if hasXMLElement $body "AMAT_Platen_Control" -}}
          {{- $result = dict "module" "CMP" "type" "AMAT_Platen_Control" -}}
        {{- else if hasXMLElement $body "HHQK_W2W_Control" -}}
          {{- $result = dict "module" "CMP" "type" "HHQK_W2W_Control" -}}
        {{- else if hasXMLElement $body "Post_R2R_DC" -}}
          {{- $result = dict "module" "CMP" "type" "Post_R2R_DC" -}}
        {{- else if hasXMLElement $body "Inline_Detection" -}}
          {{- $result = dict "module" "CMP" "type" "Inline_Detection" -}}
        {{- else -}}
          {{- $result = dict "module" "non-PH" "type" "*" -}}
        {{- end -}}
      {{- else -}}
        {{- $result = dict "module" "non-PH" "type" "*" -}}
      {{- end -}}
    {{- else -}}
      {{- $result = dict "module" "non-PH" "type" "*" -}}
    {{- end -}}
  {{- else -}}
    {{- $result = dict "module" "non-PH" "type" "*" -}}
  {{- end -}}
{{- else if contains $contentTypeLower "application/json" -}}
  {{- /* JSON/HTTP Request Processing */ -}}
  {{- if .BodyJSON -}}
    {{- $srcSystem := header .Request "x-src-system" -}}
    {{- if eq $srcSystem "ASML-LIS" -}}
      {{- $result = dict "module" "PH" "type" "GetExposureInfoForLis" -}}
      {{- $opno := index .BodyJSON "opno" -}}
      {{- if $opno -}}
        {{- $result = merge $result (dict "operationNo" $opno) -}}
      {{- end -}}
    {{- else -}}
      {{- /* Check for EH module unLockChamber request */ -}}
      {{- $prodspecId := index .BodyJSON "prodspecId" -}}
      {{- $routeId := index .BodyJSON "routeId" -}}
      {{- $opeNo := index .BodyJSON "opeNo" -}}
      {{- $lotId := index .BodyJSON "lotId" -}}
      {{- $recipeId := index .BodyJSON "recipeId" -}}
      {{- $eqpId := index .BodyJSON "eqpId" -}}
      {{- $chamberId := index .BodyJSON "chamberId" -}}
      {{- if and $prodspecId $routeId $opeNo $lotId $recipeId $eqpId $chamberId -}}
        {{- $result = dict "module" "EH" "type" "unLockChamber" -}}
      {{- else -}}
        {{- $result = dict "module" "non-PH" "type" "*" -}}
      {{- end -}}
    {{- end -}}
  {{- else -}}
    {{- $result = dict "module" "non-PH" "type" "*" -}}
  {{- end -}}
{{- else -}}
  {{- $result = dict "module" "non-PH" "type" "*" -}}
{{- end -}}

{{- /* Output the result as JSON */ -}}
{{ toJson $result }}`

	loader.AddTemplate("soap_parser", templateContent)

	// Create parser
	config := Config{
		TemplateLoader: loader,
		MaxCacheSize:   100,
		WatchFiles:     false,
		FuncMap:        DefaultFuncMap(),
	}

	p, err := NewParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	defer p.Close()

	// Helper function to test parsing
	testParse := func(t *testing.T, name string, req *http.Request, expected ParseResult) {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			err := p.Parse("soap_parser", req, &output)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			var result ParseResult
			err = json.Unmarshal(output.Bytes(), &result)
			if err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if result.Module != expected.Module {
				t.Errorf("Expected module %s, got %s", expected.Module, result.Module)
			}
			if result.Type != expected.Type {
				t.Errorf("Expected type %s, got %s", expected.Type, result.Type)
			}
			if result.RouteGroup != expected.RouteGroup {
				t.Errorf("Expected routegroup %s, got %s", expected.RouteGroup, result.RouteGroup)
			}
			if result.Controller != expected.Controller {
				t.Errorf("Expected controller %s, got %s", expected.Controller, result.Controller)
			}
			if result.Layer != expected.Layer {
				t.Errorf("Expected layer %s, got %s", expected.Layer, result.Layer)
			}
			if result.Tool != expected.Tool {
				t.Errorf("Expected tool %s, got %s", expected.Tool, result.Tool)
			}
			if result.ProductGroup != expected.ProductGroup {
				t.Errorf("Expected productGroup %s, got %s", expected.ProductGroup, result.ProductGroup)
			}
			if result.OperationNo != expected.OperationNo {
				t.Errorf("Expected operationNo %s, got %s", expected.OperationNo, result.OperationNo)
			}
		})
	}

	// Test SOAP XML requests
	t.Run("SOAP_XML_Tests", func(t *testing.T) {
		// Test MESRecipeTurnOff
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
		testParse(t, "MESRecipeTurnOff", req1, ParseResult{Module: "*", Type: "MESRecipeTurnOff"})

		// Test Recommend_Request
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
		testParse(t, "Recommend_Request", req2, ParseResult{
			Module:     "PH",
			Type:       "REC",
			RouteGroup: "RG001",
			Controller: "CTRL_JOB_1",
			Layer:      "Layer_A",
		})

		// Test Metrology_Request
		soapXML3 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Metrology_Request>
      <objRequest>
        <CONTEXT_INFO>
          <ROUTEGROUP>RG002</ROUTEGROUP>
          <CONTROLJOBTYPE>CTRL_JOB_2</CONTROLJOBTYPE>
          <LAYER>Layer_B</LAYER>
        </CONTEXT_INFO>
      </objRequest>
    </Metrology_Request>
  </soap:Body>
</soap:Envelope>`
		req3, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML3))
		req3.Header.Set("Content-Type", "text/xml")
		testParse(t, "Metrology_Request", req3, ParseResult{
			Module:     "PH",
			Type:       "MET",
			RouteGroup: "RG002",
			Controller: "CTRL_JOB_2",
			Layer:      "Layer_B",
		})

		// Test LotStartTime_For_Scanner
		soapXML4 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LotStartTime_For_Scanner>
      <param>value</param>
    </LotStartTime_For_Scanner>
  </soap:Body>
</soap:Envelope>`
		req4, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML4))
		req4.Header.Set("Content-Type", "text/xml")
		testParse(t, "LotStartTime_For_Scanner", req4, ParseResult{Module: "PH", Type: "LotStartTime_For_Scanner"})

		// Test R2R_Object_Metrology (non-PH dispatching)
		soapXML5 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <R2R_Object_Metrology>
      <param>value</param>
    </R2R_Object_Metrology>
  </soap:Body>
</soap:Envelope>`
		req5, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML5))
		req5.Header.Set("Content-Type", "text/xml")
		testParse(t, "R2R_Object_Metrology", req5, ParseResult{Module: "non-PH", Type: "DISPATCHING"})

		// Test AMAT_Platen_Control (CMP module)
		soapXML6 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <AMAT_Platen_Control>
      <param>value</param>
    </AMAT_Platen_Control>
  </soap:Body>
</soap:Envelope>`
		req6, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML6))
		req6.Header.Set("Content-Type", "text/xml")
		testParse(t, "AMAT_Platen_Control", req6, ParseResult{Module: "CMP", Type: "AMAT_Platen_Control"})

		// Test GetUsedLotInfo with result_xml and headers
		soapXML7 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetUsedLotInfo>
      <result_xml>some_xml_data</result_xml>
    </GetUsedLotInfo>
  </soap:Body>
</soap:Envelope>`
		req7, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML7))
		req7.Header.Set("Content-Type", "text/xml")
		req7.Header.Set("x-eap-tool", "Tool123")
		req7.Header.Set("x-eap-product-group", "PG456")
		req7.Header.Set("x-eap-operation-no", "OP789")
		testParse(t, "GetUsedLotInfo_with_result_xml", req7, ParseResult{
			Module:       "PH",
			Type:         "GetUsedLotInfo",
			Tool:         "Tool123",
			ProductGroup: "PG456",
			OperationNo:  "OP789",
		})

		// Test GetUsedLotInfo with ProductID
		soapXML8 := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetUsedLotInfo>
      <ProductID>PROD123</ProductID>
    </GetUsedLotInfo>
  </soap:Body>
</soap:Envelope>`
		req8, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML8))
		req8.Header.Set("Content-Type", "text/xml")
		testParse(t, "GetUsedLotInfo_with_ProductID", req8, ParseResult{Module: "PH", Type: "EAPGetUsedLotInfo"})
	})

	// Test JSON HTTP requests
	t.Run("JSON_HTTP_Tests", func(t *testing.T) {
		// Test ASML-LIS request
		jsonPayload1 := `{"opno": "OP123", "lotId": "LOT456"}`
		req1, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload1))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("x-src-system", "ASML-LIS")
		testParse(t, "ASML_LIS_Request", req1, ParseResult{
			Module:      "PH",
			Type:        "GetExposureInfoForLis",
			OperationNo: "OP123",
		})

		// Test EH unLockChamber request
		jsonPayload2 := `{
			"prodspecId": "PROD001",
			"routeId": "ROUTE001", 
			"opeNo": "OP001",
			"lotId": "LOT001",
			"recipeId": "RECIPE001",
			"eqpId": "EQP001",
			"chamberId": "CHAMBER001"
		}`
		req2, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload2))
		req2.Header.Set("Content-Type", "application/json")
		testParse(t, "EH_unLockChamber", req2, ParseResult{Module: "EH", Type: "unLockChamber"})

		// Test incomplete EH request (should default to non-PH)
		jsonPayload3 := `{
			"prodspecId": "PROD001",
			"routeId": "ROUTE001"
		}`
		req3, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload3))
		req3.Header.Set("Content-Type", "application/json")
		testParse(t, "Incomplete_EH_Request", req3, ParseResult{Module: "non-PH", Type: "*"})
	})

	// Test edge cases
	t.Run("Edge_Cases", func(t *testing.T) {
		// Test unknown content type
		req1, _ := http.NewRequest("GET", "http://example.com/unknown", nil)
		req1.Header.Set("Content-Type", "text/plain")
		testParse(t, "Unknown_Content_Type", req1, ParseResult{Module: "non-PH", Type: "*"})

		// Test unknown XML request
		soapXML := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <UnknownRequest>
      <param>value</param>
    </UnknownRequest>
  </soap:Body>
</soap:Envelope>`
		req2, _ := http.NewRequest("POST", "http://example.com/soap", strings.NewReader(soapXML))
		req2.Header.Set("Content-Type", "text/xml")
		testParse(t, "Unknown_XML_Request", req2, ParseResult{Module: "non-PH", Type: "*"})

		// Test JSON without ASML-LIS header or EH fields
		jsonPayload := `{"someField": "someValue"}`
		req3, _ := http.NewRequest("POST", "http://example.com/api", strings.NewReader(jsonPayload))
		req3.Header.Set("Content-Type", "application/json")
		testParse(t, "Unknown_JSON_Request", req3, ParseResult{Module: "non-PH", Type: "*"})
	})
}