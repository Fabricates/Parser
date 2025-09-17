package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fabricates/parser"
)

var (
	parserMutex  sync.Mutex
	p            parser.GenericParser[map[string]any]
	lastMod      time.Time
	templatePath string
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: sparser <template_file>")
	}

	templatePath = os.Args[1]

	// Initial load
	err := loadTemplate()
	if err != nil {
		log.Fatalf("Failed to load template: %v", err)
	}

	log.Printf("Loaded template from %s", templatePath)

	// HTTP handler
	http.HandleFunc("/parse", func(w http.ResponseWriter, r *http.Request) {
		parserMutex.Lock()
		defer parserMutex.Unlock()

		// Check if template file has changed
		if hasTemplateChanged() {
			err := reloadTemplate()
			if err != nil {
				http.Error(w, "Failed to reload template: "+err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("Reloaded template from %s", templatePath)
		}

		// Parse request using template
		result, _, err := p.Parse("main", r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// Start server
	log.Println("Starting web server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadTemplate() error {
	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// Get file mod time
	stat, err := os.Stat(templatePath)
	if err != nil {
		return err
	}
	lastMod = stat.ModTime()

	// Create custom funcmap with toJson
	fm := parser.DefaultFuncMap()
	fm["toJson"] = func(v interface{}) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	// Create parser configuration
	config := parser.Config{
		MaxCacheSize: 100,
		FuncMap:      fm,
	}

	// Create parser
	p, err = parser.NewGenericParser[map[string]any](config)
	if err != nil {
		return err
	}

	// Load template
	err = p.UpdateTemplate("main", string(content))
	if err != nil {
		return err
	}

	return nil
}

func hasTemplateChanged() bool {
	stat, err := os.Stat(templatePath)
	if err != nil {
		log.Printf("Error checking template file: %v", err)
		return false
	}
	return stat.ModTime().After(lastMod)
}

func reloadTemplate() error {
	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// Update mod time
	stat, err := os.Stat(templatePath)
	if err != nil {
		return err
	}
	lastMod = stat.ModTime()

	// Reload template
	err = p.UpdateTemplate("main", string(content))
	if err != nil {
		return err
	}

	return nil
}
