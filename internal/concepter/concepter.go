package concepter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lukse/doppel/internal/parser"
)

// ConceptDoc is a structured representation of a single CodeUnit, suitable for
// embedding as an architecture-level semantic signal.
type ConceptDoc struct {
	Name         string
	Package      string
	Summary      string   // LLM-generated; empty in static-only mode
	Inputs       []string // parameter types; from Go signature or LLM
	Outputs      []string // return types; from Go signature or LLM
	Dependencies []string // external packages/services; LLM-generated
	Callers      []string // functions that call this one; from call graph (never cached)
	Patterns     []string // tagger tags + optional LLM expansion
}

// Format renders the ConceptDoc into a flat text block used as the embedding input.
// Sections with zero items are omitted entirely.
func (d ConceptDoc) Format() string {
	var sb strings.Builder

	sb.WriteString("Name: " + d.Name + "\n")
	if d.Package != "" {
		sb.WriteString("Package: " + d.Package + "\n")
	}
	if d.Summary != "" {
		sb.WriteString("Summary: " + d.Summary + "\n")
	}
	writeList(&sb, "Inputs", d.Inputs)
	writeList(&sb, "Outputs", d.Outputs)
	writeList(&sb, "Dependencies", d.Dependencies)
	writeList(&sb, "Callers", d.Callers)
	writeList(&sb, "Patterns", d.Patterns)

	return sb.String()
}

func writeList(sb *strings.Builder, header string, items []string) {
	if len(items) == 0 {
		return
	}
	sb.WriteString(header + ":\n")
	for _, item := range items {
		sb.WriteString("- " + item + "\n")
	}
}

// Concepter generates ConceptDocs for CodeUnits using static analysis and
// optionally an Ollama chat model.
type Concepter struct {
	baseURL string
	model   string // empty = static-only mode
	cache   *ConceptCache
}

// New creates a Concepter. model may be empty for static-only mode.
func New(baseURL, model string, cache *ConceptCache) *Concepter {
	return &Concepter{baseURL: baseURL, model: model, cache: cache}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// Generate produces a ConceptDoc for the given unit.
// callers is the pre-built list of functions that reference this unit's Name.
// If the Concepter has no model, only statically-derivable fields are populated.
func (c *Concepter) Generate(unit parser.CodeUnit, callers []string) (ConceptDoc, error) {
	// Check cache first.
	if cached, ok := c.cache.Get(unit.Body); ok {
		cached.Callers = callers
		return cached, nil
	}

	// Always compute static fields.
	doc := ConceptDoc{
		Name:     unit.Name,
		Package:  unit.Package,
		Patterns: append([]string(nil), unit.Patterns...), // copy tagger output
	}
	if unit.Signature != "" {
		doc.Inputs, doc.Outputs = parseGoSignature(unit.Signature)
	}

	// Augment with LLM if a model is configured.
	if c.model != "" {
		llmDoc, err := c.callLLM(unit)
		if err != nil {
			// Non-fatal: log at call site, return static doc.
			doc.Callers = callers
			return doc, fmt.Errorf("llm: %w", err)
		}
		if llmDoc.Summary != "" {
			doc.Summary = llmDoc.Summary
		}
		// Use LLM inputs/outputs if static extraction produced nothing.
		if len(doc.Inputs) == 0 {
			doc.Inputs = llmDoc.Inputs
		}
		if len(doc.Outputs) == 0 {
			doc.Outputs = llmDoc.Outputs
		}
		doc.Dependencies = llmDoc.Dependencies
		doc.Patterns = mergeUnique(doc.Patterns, llmDoc.Patterns)
	}

	// Cache the doc without Callers (callers are always re-injected from the live graph).
	c.cache.Set(unit.Body, doc)

	doc.Callers = callers
	return doc, nil
}

// callLLM sends the unit to Ollama and returns the parsed concept response.
func (c *Concepter) callLLM(unit parser.CodeUnit) (llmConceptResponse, error) {
	prompt := buildConceptPrompt(unit)

	payload, err := json.Marshal(generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return llmConceptResponse{}, err
	}

	resp, err := http.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(payload))
	if err != nil {
		return llmConceptResponse{}, fmt.Errorf("ollama generate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return llmConceptResponse{}, fmt.Errorf("ollama generate returned %d: %s", resp.StatusCode, string(b))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return llmConceptResponse{}, fmt.Errorf("decode generate response: %w", err)
	}

	return parseConceptResponse(result.Response)
}

// mergeUnique returns a slice containing all items from base followed by any
// items from extra that are not already present (case-sensitive).
func mergeUnique(base, extra []string) []string {
	seen := make(map[string]bool, len(base))
	for _, v := range base {
		seen[v] = true
	}
	result := append([]string(nil), base...)
	for _, v := range extra {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
