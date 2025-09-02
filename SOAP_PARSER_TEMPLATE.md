# SOAP Request Parser Template

This template provides an equivalent implementation of a JavaScript SOAP request parser using the Go template parser. It analyzes HTTP requests (both XML/SOAP and JSON) and returns structured metadata about the request type.

## Overview

The template replicates the behavior of a JavaScript function that:
- Parses SOAP/XML requests and extracts structured information
- Handles JSON/HTTP requests with specific routing logic
- Returns structured JSON metadata about the request type and module

## Features

- **XML/SOAP Request Processing**: Handles various SOAP envelope structures and extracts request types
- **JSON/HTTP Request Processing**: Processes JSON payloads with conditional logic
- **Module Classification**: Categorizes requests into different modules (PH, non-PH, EH, CMP)
- **Context Extraction**: Extracts contextual information like route groups, controllers, and layers
- **Header-based Routing**: Uses HTTP headers for conditional processing
- **JSON Output**: Returns results as structured JSON

## Template Functions Used

The template leverages several custom template functions:

- `header`: Extract HTTP header values
- `lower`: Convert strings to lowercase  
- `contains`: Check if string contains substring
- `dict`: Create dictionary/map structures
- `merge`: Merge dictionaries
- `index`: Access map/slice elements
- `hasXMLElement`: Check if XML element exists
- `toJson`: Convert data to JSON string

## Request Types Supported

### SOAP/XML Requests

#### PH Module Requests
- `MESRecipeTurnOff` → `{module: "*", type: "MESRecipeTurnOff"}`
- `Recommend_Request` → `{module: "PH", type: "REC", routegroup, controller, layer}`
- `Metrology_Request` → `{module: "PH", type: "MET", routegroup, controller, layer}`
- `Used_Request` → `{module: "PH", type: "USED", routegroup, controller, layer}`
- `LotStartTime_For_Scanner` → `{module: "PH", type: "LotStartTime_For_Scanner"}`
- `LotStartTime_For_YieldStarAndKLA` → `{module: "PH", type: "LotStartTime_For_YieldStarAndKLA"}`
- `GetUsedExpParameters` → `{module: "PH", type: "GetUsedExpParameters"}`
- `GetUsedLotInfoTest` → `{module: "PH", type: "GetUsedLotInfoTest"}`
- `ResendToolOffsets` → `{module: "PH", type: "ResendToolOffsets"}`
- `PackingLotEnd` → `{module: "PH", type: "PackingLotEnd"}`
- `WLC_EAP_REQUEST` → `{module: "PH", type: "WLC_EAP_REQUEST"}`

#### Special GetUsedLotInfo Handling
- With `result_xml` → `{module: "PH", type: "GetUsedLotInfo", tool, productGroup, operationNo}`
- With `ProductID` → `{module: "PH", type: "EAPGetUsedLotInfo"}`

#### Non-PH Dispatching Requests
- `R2R_Object_Metrology` → `{module: "non-PH", type: "DISPATCHING"}`
- `R2R_Object_Recommend` → `{module: "non-PH", type: "DISPATCHING"}`
- `R2R_Object_Used` → `{module: "non-PH", type: "DISPATCHING"}`

#### CMP Module Requests
- `AMAT_Platen_Control` → `{module: "CMP", type: "AMAT_Platen_Control"}`
- `HHQK_W2W_Control` → `{module: "CMP", type: "HHQK_W2W_Control"}`
- `Post_R2R_DC` → `{module: "CMP", type: "Post_R2R_DC"}`
- `Inline_Detection` → `{module: "CMP", type: "Inline_Detection"}`

### JSON/HTTP Requests

#### ASML-LIS System Requests
Header: `x-src-system: ASML-LIS`
→ `{module: "PH", type: "GetExposureInfoForLis", operationNo}`

#### EH Module Requests
JSON with all required fields:
- `prodspecId`
- `routeId`
- `opeNo`
- `lotId`
- `recipeId`
- `eqpId`
- `chamberId`

→ `{module: "EH", type: "unLockChamber"}`

### Default Behavior
Any unrecognized request → `{module: "non-PH", type: "*"}`

## Usage Example

```go
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
    // Load template content
    templateContent, err := os.ReadFile("templates/soap_request_parser.tmpl")
    if err != nil {
        log.Fatalf("Failed to read template: %v", err)
    }

    // Create memory loader and add template
    loader := parser.NewMemoryLoader()
    loader.AddTemplate("soap_parser", string(templateContent))

    // Create parser with custom functions
    config := parser.Config{
        TemplateLoader: loader,
        MaxCacheSize:   100,
        WatchFiles:     false,
        FuncMap:        parser.DefaultFuncMap(), // Includes required functions
    }

    p, err := parser.NewParser(config)
    if err != nil {
        log.Fatalf("Failed to create parser: %v", err)
    }
    defer p.Close()

    // Example: SOAP XML request
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
    err = p.Parse("soap_parser", req, &output)
    if err != nil {
        log.Fatalf("Failed to parse: %v", err)
    }

    fmt.Println(output.String())
    // Output: {"controller":"CTRL_JOB_1","layer":"Layer_A","module":"PH","routegroup":"RG001","type":"REC"}
}
```

## XML Structure Notes

The template works with the XML parser's flattened structure. For nested XML elements, it uses paths like:
- `Envelope/Body/Recommend_Request/objRequest/CONTEXT_INFO/ROUTEGROUP`

This allows direct access to deeply nested values without complex traversal logic.

## Testing

Comprehensive tests are provided in `soap_parser_test.go` that cover:
- All SOAP/XML request types
- JSON/HTTP request scenarios  
- Edge cases and error conditions
- Header-based conditional logic

Run tests with:
```bash
go test -v -run TestSOAPRequestParser
```

## Performance

The template is optimized for performance with:
- Template caching enabled by default
- Minimal memory allocations for result construction
- Efficient XML path lookups using flattened structure
- Lazy evaluation of conditional branches

## Error Handling

The template gracefully handles various error conditions:
- Missing XML elements
- Invalid content types
- Incomplete JSON payloads
- Missing HTTP headers

All error conditions default to `{module: "non-PH", type: "*"}` to ensure consistent output.