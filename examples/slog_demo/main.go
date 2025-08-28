package main

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	parser "github.com/fabricates/parser"
)

func main() {
	// Configure slog to show all levels including debug
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// Test invalid JSON parsing
	slog.Info("Testing invalid JSON parsing...")
	request := &http.Request{
		Method: "POST",
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(`{"invalid": json}`)),
	}
	request.Header.Set("Content-Type", "application/json")

	rereadableReq, _ := parser.NewRereadableRequest(request)
	_, err := parser.ExtractRequestData(rereadableReq, nil)
	if err != nil {
		slog.Error("Request processing failed", "error", err)
	}

	// Test invalid XML parsing
	slog.Info("Testing invalid XML parsing...")
	request2 := &http.Request{
		Method: "POST",
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(`<invalid><xml`)),
	}
	request2.Header.Set("Content-Type", "application/xml")

	rereadableReq2, _ := parser.NewRereadableRequest(request2)
	_, err2 := parser.ExtractRequestData(rereadableReq2, nil)
	if err2 != nil {
		slog.Error("Request processing failed", "error", err2)
	}

	slog.Info("Demo completed")
}
