package parser

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RereadableRequest wraps an HTTP request to make it re-readable
type RereadableRequest struct {
	*http.Request
	body       []byte
	bodyReader io.ReadCloser
}

// NewRereadableRequest creates a new re-readable HTTP request
func NewRereadableRequest(r *http.Request) (*RereadableRequest, error) {
	// Read the entire body into memory
	var body []byte
	var err error

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body.Close()
	}

	// Create wrapper that uses the original request but makes body re-readable
	req := &RereadableRequest{
		Request: r, // Use the original request, don't create a copy
		body:    body,
	}

	// Reset the original request's body to be re-readable
	req.resetBody()

	return req, nil
}

// resetBody resets the body reader to the beginning
func (r *RereadableRequest) resetBody() {
	if len(r.body) > 0 {
		r.bodyReader = io.NopCloser(bytes.NewReader(r.body))
		r.Request.Body = r.bodyReader
		r.Request.ContentLength = int64(len(r.body))
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

// ExtractRequestData extracts structured data from the HTTP request for template use
func ExtractRequestData(r *RereadableRequest, customData interface{}) (*RequestData, error) {
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

	return &RequestData{
		Request: r.Request,
		Headers: headers,
		Query:   query,
		Form:    form,
		Body:    r.Body(),
		Custom:  customData,
	}, nil
}
