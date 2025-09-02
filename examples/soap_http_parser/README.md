# SOAP/HTTP Request Parser Template

This example demonstrates a text/template implementation that mimics JavaScript request parser functionality for handling both SOAP XML and HTTP JSON requests to extract metadata.

## Overview

The template processes HTTP requests and extracts structured metadata based on the content type and request structure. It supports:

- **SOAP XML requests** with various operation types (MESRecipeTurnOff, Recommend_Request, Metrology_Request, etc.)
- **HTTP JSON requests** with header-based routing and payload validation
- **Fallback handling** for unknown request types

## Files

- `request_parser.tmpl` - The main template that implements the parsing logic
- `main.go` - Basic example with key test cases  
- `comprehensive_demo.go` - Comprehensive demonstration with all supported request types
- `test_functions.go` - Helper for testing template function availability
- `debug.go` - Debugging utility to inspect XML parsing structure

## Template Logic

The template follows this decision flow:

1. **Content-Type Detection**: Checks if the request is XML or JSON based on Content-Type header
2. **SOAP XML Processing**: 
   - Parses XML structure from `BodyXML`
   - Extracts SOAP envelope and body
   - Matches against known operation types
   - Extracts context information where available
3. **HTTP JSON Processing**:
   - Checks special headers like `x-src-system`
   - Validates required fields for specific operations
4. **Fallback**: Returns `{"module": "non-PH", "type": "*"}` for unknown requests

## Supported SOAP Operations

### PH Module Operations
- `MESRecipeTurnOff` → `{"module": "*", "type": "MESRecipeTurnOff"}`
- `Recommend_Request` → `{"module": "PH", "type": "REC", ...context}`
- `Metrology_Request` → `{"module": "PH", "type": "MET", ...context}`  
- `Used_Request` → `{"module": "PH", "type": "USED", ...context}`
- `LotStartTime_For_Scanner` → `{"module": "PH", "type": "LotStartTime_For_Scanner"}`
- `LotStartTime_For_YieldStarAndKLA` → `{"module": "PH", "type": "LotStartTime_For_YieldStarAndKLA"}`
- `GetUsedExpParameters` → `{"module": "PH", "type": "GetUsedExpParameters"}`
- `GetUsedLotInfoTest` → `{"module": "PH", "type": "GetUsedLotInfoTest"}`
- `ResendToolOffsets` → `{"module": "PH", "type": "ResendToolOffsets"}`
- `PackingLotEnd` → `{"module": "PH", "type": "PackingLotEnd"}`
- `GetUsedLotInfo` → Various types based on content and headers
- `WLC_EAP_REQUEST` → `{"module": "PH", "type": "WLC_EAP_REQUEST"}`

### Non-PH Module Operations  
- `R2R_Object_Metrology` → `{"module": "non-PH", "type": "DISPATCHING"}`
- `R2R_Object_Recommend` → `{"module": "non-PH", "type": "DISPATCHING"}`
- `R2R_Object_Used` → `{"module": "non-PH", "type": "DISPATCHING"}`

### CMP Module Operations
- `AMAT_Platen_Control` → `{"module": "CMP", "type": "AMAT_Platen_Control"}`
- `HHQK_W2W_Control` → `{"module": "CMP", "type": "HHQK_W2W_Control"}`
- `Post_R2R_DC` → `{"module": "CMP", "type": "Post_R2R_DC"}`
- `Inline_Detection` → `{"module": "CMP", "type": "Inline_Detection"}`

## Supported HTTP JSON Operations

- **ASML-LIS requests**: Detected by `x-src-system: ASML-LIS` header → `{"module": "PH", "type": "GetExposureInfoForLis", "operationNo": "..."}`
- **Chamber unlock requests**: Detected by presence of all required fields (prodspecId, routeId, opeNo, lotId, recipeId, eqpId, chamberId) → `{"module": "EH", "type": "unLockChamber"}`

## Usage

```bash
# Run basic example
go run main.go

# Run comprehensive demonstration
go run comprehensive_demo.go

# Debug XML structure
go run debug.go
```

## Extended Function Map

The template uses an extended function map with additional string functions:
- `hasPrefix(string, prefix)` - Check if string starts with prefix
- `contains(string, substring)` - Check if string contains substring

These functions are added to the default function map provided by the parser library.

## Original JavaScript Reference

This template implements the same logic as the provided JavaScript code that used:
- `parseFromSoapRequest(headers, payload)` for XML requests
- `parseFromHttpRequest(headers, payload)` for JSON requests
- Content-type based routing between XML and JSON parsers
- Header-based metadata extraction
- JSON response generation