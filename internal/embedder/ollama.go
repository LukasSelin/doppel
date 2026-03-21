package embedder

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Embedder calls the Ollama API to generate text embeddings.
type Embedder struct {
	baseURL   string
	model     string
	numCtx    int
	cache     map[string][]float64
	cachePath string
}

type ollamaEmbedRequest struct {
	Model   string         `json:"model"`
	Input   string         `json:"input"`
	Options *ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// New creates an Embedder. cachePath may be empty to disable caching.
// numCtx is passed to Ollama as options.num_ctx (tokens); use 0 to omit and keep the server default.
func New(baseURL, model, cachePath string, numCtx int) (*Embedder, error) {
	e := &Embedder{
		baseURL:   baseURL,
		model:     model,
		numCtx:    numCtx,
		cache:     make(map[string][]float64),
		cachePath: cachePath,
	}
	if cachePath != "" {
		if err := e.loadCache(); err != nil {
			// Non-fatal: start with empty cache
			_ = err
		}
	}
	return e, nil
}

// Embed returns the embedding vector for the given text.
func (e *Embedder) Embed(text string) ([]float64, error) {
	key := cacheKey(e.model, e.numCtx, text)
	if v, ok := e.cache[key]; ok {
		return v, nil
	}

	reqBody := ollamaEmbedRequest{Model: e.model, Input: text}
	if e.numCtx > 0 {
		reqBody.Options = &ollamaOptions{NumCtx: e.numCtx}
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(e.baseURL+"/api/embed", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(b))
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}
	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama: empty embeddings")
	}
	vec := result.Embeddings[0]

	e.cache[key] = vec
	return vec, nil
}

// SaveCache persists the in-memory cache to disk.
func (e *Embedder) SaveCache() error {
	if e.cachePath == "" {
		return nil
	}
	data, err := json.Marshal(e.cache)
	if err != nil {
		return err
	}
	return os.WriteFile(e.cachePath, data, 0644)
}

func (e *Embedder) loadCache() error {
	data, err := os.ReadFile(e.cachePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &e.cache)
}

func cacheKey(model string, numCtx int, text string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s", model, numCtx, text)))
	return fmt.Sprintf("%x", h)
}
