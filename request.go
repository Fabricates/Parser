package parser

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var xmlContentTypes = []string{
	"text/xml",
	"application/xml",
	"application/soap+xml",
}

// RereadableRequest wraps an HTTP request to make it re-readable
type RereadableRequest struct {
	*http.Request
	body         []byte
	providedBody bool // true if body was provided externally, false if read from request
}

// NewRereadableRequest creates a new re-readable HTTP request
// If body is provided, it will be used instead of reading from the request's body stream
func NewRereadableRequest(r *http.Request, body ...[]byte) (*RereadableRequest, error) {
	var requestBody []byte
	var err error

	// Use provided body if available, otherwise read from request
	var providedBody bool
	if len(body) > 0 && body[0] != nil {
		requestBody = body[0]
		providedBody = true
	} else if r.Body != nil {
		providedBody = false
		if rr, ok := r.Body.(Reader); ok {
			requestBody, err = rr.ReadAll()
			if err != nil {
				return nil, err
			}
			rr.Reset()
		} else {
			if r.Body, requestBody, err = NewRepeatableReadCloser(r.Body); err != nil {
				return nil, err
			}
			r.Body.Close()
		}
	}

	// Create wrapper that uses the original request but makes body re-readable
	req := &RereadableRequest{
		Request:      r, // Use the original request, don't create a copy
		body:         requestBody,
		providedBody: providedBody,
	}

	// Reset the original request's body to be re-readable (only if body wasn't provided externally)
	if !providedBody {
		req.resetBody()
	}

	return req, nil
}

// resetBody resets the body reader to the beginning
func (r *RereadableRequest) resetBody() {
	// If body was provided externally, no need to reset the reader
	if r.providedBody {
		return
	}

	// Only reset if body was read from the original request
	if r.Request.Body != nil {
		r.Request.Body.(Reader).Reset()
	}
}

// Reset resets the request body to the beginning for re-reading
func (r *RereadableRequest) Reset() {
	r.resetBody()
}

// Body returns the request body as a string
func (r *RereadableRequest) Body() string {
	return string(r.body)
}

// BodyBytes returns the request body as bytes
func (r *RereadableRequest) BodyBytes() []byte {
	// Return a copy to prevent modification
	result := make([]byte, len(r.body))
	copy(result, r.body)
	return result
}

// Extract extracts structured data from the HTTP request for template use
func (r *RereadableRequest) Extract() (*RequestData, error) {
	// Parse form data if not already parsed
	if r.Request.Form == nil {
		r.Reset() // Ensure body is readable

		// Parse form based on content type
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			if err := r.Request.ParseForm(); err != nil {
				return nil, err
			}
		} else if strings.Contains(contentType, "multipart/form-data") {
			if err := r.Request.ParseMultipartForm(32 << 20); err != nil { // 32 MB max memory
				return nil, err
			}
		}
	}

	// Extract query parameters
	query := make(map[string][]string)
	if r.URL.RawQuery != "" {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err == nil {
			for k, v := range values {
				query[k] = v
			}
		}
	}

	// Extract headers
	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	// Extract form data
	form := make(map[string][]string)
	if r.Request.Form != nil {
		for k, v := range r.Request.Form {
			form[k] = v
		}
	}

	// Parse JSON body if content type is JSON
	var bodyJSON map[string]interface{}
	var bodyXML map[string]interface{}

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") && len(r.body) > 0 {
		var parsedJSON map[string]interface{}
		if err := json.Unmarshal(r.body, &parsedJSON); err != nil {
			// Log JSON parsing failure but continue processing
			slog.Warn("Failed to parse JSON body", "error", err, "content_type", contentType)
			// Create error structure similar to XML for consistency
			bodyJSON = nil
		} else {
			// Wrap successful JSON parsing in standard structure for consistency
			bodyJSON = parsedJSON
		}
	} else {
		// Parse XML body if content type is XML
		if len(r.body) > 0 {
			for _, ct := range xmlContentTypes {
				if strings.Contains(contentType, ct) {
					// Parse XML into structured format
					parsedXML, err := parseXMLToGeneric(string(r.body))
					if err != nil {
						// Log XML parsing failure but continue processing
						slog.Warn("Failed to parse XML body", "error", err, "content_type", contentType)
						bodyXML = nil
					} else {
						bodyXML = parsedXML
					}
					break
				}
			}
		}
	}

	return &RequestData{
		Request:  r.Request,
		Headers:  headers,
		Query:    query,
		Form:     form,
		Body:     r.Body(),
		BodyJSON: bodyJSON,
		BodyXML:  bodyXML,
		Custom:   nil, // Custom data is no longer supported in Extract method
	}, nil
}
