package parser

import (
	"container/list"
	"sync"
	"text/template"
	"time"
)

// CachedTemplate holds a compiled template with metadata
type CachedTemplate struct {
	Template     *template.Template
	LastModified time.Time
	AccessTime   time.Time
	AccessCount  int64
}

// TemplateCache provides efficient caching of compiled templates
type TemplateCache struct {
	templates map[string]*CachedTemplate
	lruList   *list.List
	lruIndex  map[string]*list.Element
	maxSize   int
	funcMap   template.FuncMap
	mu        sync.RWMutex
}

// NewTemplateCache creates a new template cache
func NewTemplateCache(maxSize int, funcMap template.FuncMap) *TemplateCache {
	return &TemplateCache{
		templates: make(map[string]*CachedTemplate),
		lruList:   list.New(),
		lruIndex:  make(map[string]*list.Element),
		maxSize:   maxSize,
		funcMap:   funcMap,
	}
}

// Get retrieves a template from the cache or compiles it if not found
func (c *TemplateCache) Get(name string, loader TemplateLoader) (*template.Template, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if template exists in cache
	if cached, exists := c.templates[name]; exists {
		// Check if template needs to be reloaded
		lastMod, err := loader.LastModified(name)
		if err != nil {
			// If we can't get the modification time, use cached version
			c.updateAccess(name, cached)
			return cached.Template, nil
		}
		
		if lastMod.After(cached.LastModified) {
			// Template has been modified, reload it
			return c.loadAndCache(name, loader)
		}
		
		// Template is up to date, update access time and return
		c.updateAccess(name, cached)
		return cached.Template, nil
	}
	
	// Template not in cache, load and cache it
	return c.loadAndCache(name, loader)
}

// loadAndCache loads a template and adds it to the cache
func (c *TemplateCache) loadAndCache(name string, loader TemplateLoader) (*template.Template, error) {
	// Load template content
	content, err := loader.Load(name)
	if err != nil {
		return nil, err
	}
	
	// Get last modified time
	lastMod, err := loader.LastModified(name)
	if err != nil {
		lastMod = time.Now()
	}
	
	// Compile template
	tmpl := template.New(name)
	if c.funcMap != nil {
		tmpl = tmpl.Funcs(c.funcMap)
	}
	
	tmpl, err = tmpl.Parse(content)
	if err != nil {
		return nil, err
	}
	
	// Create cached template
	cached := &CachedTemplate{
		Template:     tmpl,
		LastModified: lastMod,
		AccessTime:   time.Now(),
		AccessCount:  1,
	}
	
	// Add to cache
	c.addToCache(name, cached)
	
	return tmpl, nil
}

// addToCache adds a template to the cache with LRU eviction
func (c *TemplateCache) addToCache(name string, cached *CachedTemplate) {
	// Remove existing entry if it exists
	if existing, exists := c.templates[name]; exists {
		c.removeFromLRU(name)
		_ = existing
	}
	
	// Add new entry
	c.templates[name] = cached
	element := c.lruList.PushFront(name)
	c.lruIndex[name] = element
	
	// Evict least recently used items if cache is full
	if c.maxSize > 0 && len(c.templates) > c.maxSize {
		c.evictLRU()
	}
}

// updateAccess updates the access time and count for a cached template
func (c *TemplateCache) updateAccess(name string, cached *CachedTemplate) {
	cached.AccessTime = time.Now()
	cached.AccessCount++
	
	// Move to front of LRU list
	if element, exists := c.lruIndex[name]; exists {
		c.lruList.MoveToFront(element)
	}
}

// removeFromLRU removes an item from the LRU tracking
func (c *TemplateCache) removeFromLRU(name string) {
	if element, exists := c.lruIndex[name]; exists {
		c.lruList.Remove(element)
		delete(c.lruIndex, name)
	}
}

// evictLRU evicts the least recently used template
func (c *TemplateCache) evictLRU() {
	if c.lruList.Len() == 0 {
		return
	}
	
	// Get the least recently used item
	back := c.lruList.Back()
	if back != nil {
		name := back.Value.(string)
		c.lruList.Remove(back)
		delete(c.lruIndex, name)
		delete(c.templates, name)
	}
}

// Remove removes a template from the cache
func (c *TemplateCache) Remove(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.templates[name]; exists {
		delete(c.templates, name)
		c.removeFromLRU(name)
	}
}

// Clear clears all templates from the cache
func (c *TemplateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.templates = make(map[string]*CachedTemplate)
	c.lruList = list.New()
	c.lruIndex = make(map[string]*list.Element)
}

// Stats returns cache statistics
func (c *TemplateCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := CacheStats{
		Size:     len(c.templates),
		MaxSize:  c.maxSize,
		HitCount: 0,
	}
	
	for _, cached := range c.templates {
		stats.HitCount += cached.AccessCount
	}
	
	return stats
}

// CacheStats holds cache statistics
type CacheStats struct {
	Size     int   // Current number of cached templates
	MaxSize  int   // Maximum cache size (0 = unlimited)
	HitCount int64 // Total number of cache hits
}