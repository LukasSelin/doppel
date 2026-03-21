package concepter

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

// ConceptCache persists generated ConceptDocs to a JSON file.
// Cache keys are SHA-256(model + "\x00" + bodyText) so changing the model
// automatically invalidates all cached docs.
type ConceptCache struct {
	path  string
	model string
	data  map[string]ConceptDoc
}

// NewConceptCache loads an existing cache file or starts with an empty cache.
// A missing or corrupt file is non-fatal.
func NewConceptCache(path, model string) (*ConceptCache, error) {
	cc := &ConceptCache{
		path:  path,
		model: model,
		data:  make(map[string]ConceptDoc),
	}
	if path != "" {
		if err := cc.load(); err != nil {
			// Non-fatal: start with empty cache.
			_ = err
		}
	}
	return cc, nil
}

// Get returns the cached ConceptDoc for the given function body text.
// Callers is always empty in cached docs (it is re-injected from the live call graph).
func (cc *ConceptCache) Get(bodyText string) (ConceptDoc, bool) {
	doc, ok := cc.data[cc.cacheKey(bodyText)]
	return doc, ok
}

// Set stores a ConceptDoc in the in-memory cache.
// Callers is cleared before storing so stale caller lists never persist.
func (cc *ConceptCache) Set(bodyText string, doc ConceptDoc) {
	doc.Callers = nil
	cc.data[cc.cacheKey(bodyText)] = doc
}

// Save persists the in-memory cache to disk. A no-op when path is empty.
func (cc *ConceptCache) Save() error {
	if cc.path == "" {
		return nil
	}
	data, err := json.Marshal(cc.data)
	if err != nil {
		return err
	}
	return os.WriteFile(cc.path, data, 0644)
}

func (cc *ConceptCache) load() error {
	data, err := os.ReadFile(cc.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &cc.data)
}

func (cc *ConceptCache) cacheKey(text string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s", cc.model, text)))
	return fmt.Sprintf("%x", h)
}
