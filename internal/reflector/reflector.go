package reflector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/parser"
)

// Reflector calls an Ollama chat/generate model to explain why a similar pair should be merged.
type Reflector struct {
	baseURL string
	model   string
}

// New creates a Reflector.
func New(baseURL, model string) *Reflector {
	return &Reflector{baseURL: baseURL, model: model}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"` // always false — avoids streaming complexity
}

type generateResponse struct {
	Response string `json:"response"`
}

// Explain sends both function bodies to the LLM and returns a merge rationale.
func (r *Reflector) Explain(pair analyzer.SimilarPair) (string, error) {
	prompt := buildPrompt(pair)

	payload, err := json.Marshal(generateRequest{
		Model:  r.model,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(r.baseURL+"/api/generate", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("ollama generate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama generate returned %d: %s", resp.StatusCode, string(b))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode ollama generate response: %w", err)
	}

	return strings.TrimSpace(result.Response), nil
}

func buildPrompt(pair analyzer.SimilarPair) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"You are a code reviewer. Two functions have been identified as semantically similar (similarity score: %.4f).\n\n",
		pair.Score,
	))
	sb.WriteString(unitBlock("A", pair.A))
	sb.WriteString("\n")
	sb.WriteString(unitBlock("B", pair.B))
	sb.WriteString("\nIn 2-3 sentences: explain what shared logic could be extracted and how these functions could be merged or refactored. Be specific and actionable.")
	return sb.String()
}

func unitBlock(label string, u parser.CodeUnit) string {
	var sb strings.Builder
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	loc := fmt.Sprintf("%s:%d", filepath.ToSlash(u.File), u.StartLine)
	sb.WriteString(fmt.Sprintf("Function %s — %s (%s)\n", label, name, loc))
	if u.Signature != "" {
		sb.WriteString(fmt.Sprintf("Signature: %s\n", u.Signature))
	}
	sb.WriteString("```go\n")
	sb.WriteString(u.Body)
	sb.WriteString("\n```\n")
	return sb.String()
}
