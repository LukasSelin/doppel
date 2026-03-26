package reflector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/comparator"
	"github.com/lukse/doppel/internal/parser"
)

// Reflector calls an Ollama chat/generate model to explain why a similar pair should be merged.
type Reflector struct {
	baseURL        string
	model          string
	promptTemplate string // empty = use built-in prompt
}

// New creates a Reflector.
// promptTemplate may be empty to use the built-in prompt; otherwise it is a
// Go text/template string with variables {{.Score}}, {{.A.Name}}, {{.A.Package}},
// {{.A.Signature}}, {{.A.Body}}, {{.A.Location}}, and the same for {{.B}}.
func New(baseURL, model, promptTemplate string) *Reflector {
	return &Reflector{baseURL: baseURL, model: model, promptTemplate: promptTemplate}
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
	prompt, err := buildPrompt(pair, r.promptTemplate)
	if err != nil {
		return "", fmt.Errorf("build prompt: %w", err)
	}

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

// reflectUnit holds template-accessible fields for one function in a similar pair.
type reflectUnit struct {
	Name      string
	Package   string
	Signature string
	Body      string
	Location  string // "file:line"
}

// reflectPromptData is the template data available when using a custom reflect prompt template.
type reflectPromptData struct {
	Score    float64
	A        reflectUnit
	B        reflectUnit
	Evidence *comparator.StructuralEvidence
}

func buildPrompt(pair analyzer.SimilarPair, tmpl string) (string, error) {
	if tmpl != "" {
		t, err := template.New("reflect").Parse(tmpl)
		if err != nil {
			return "", fmt.Errorf("parse reflect prompt template: %w", err)
		}
		toUnit := func(u parser.CodeUnit) reflectUnit {
			name := u.Name
			if u.Package != "" {
				name = u.Package + "." + name
			}
			return reflectUnit{
				Name:      name,
				Package:   u.Package,
				Signature: u.Signature,
				Body:      u.Body,
				Location:  fmt.Sprintf("%s:%d", filepath.ToSlash(u.File), u.StartLine),
			}
		}
		data := reflectPromptData{
			Score:    pair.Score,
			A:        toUnit(pair.A),
			B:        toUnit(pair.B),
			Evidence: pair.Evidence,
		}
		var buf strings.Builder
		if err := t.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("execute reflect prompt template: %w", err)
		}
		return buf.String(), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"You are a code reviewer. Two functions have been identified as semantically similar (similarity score: %.4f).\n\n",
		pair.Score,
	))
	sb.WriteString(fmt.Sprintf("Function A: %s\n", pair.A.Name))
	sb.WriteString(unitBlock("A", pair.A))
	sb.WriteString("\n-----\n")
	sb.WriteString(fmt.Sprintf("Function B: %s\n", pair.B.Name))
	sb.WriteString(unitBlock("B", pair.B))
	sb.WriteString("\n-----\n")
	if pair.Evidence != nil {
		sb.WriteString("\nStructural Context:\n")
		for _, reason := range pair.Evidence.Reasons {
			sb.WriteString("- " + reason + "\n")
		}
		sb.WriteString(fmt.Sprintf("- Structural overlap score: %.2f\n", pair.Evidence.OverlapScore))
		if pair.Evidence.MergeWorthy {
			sb.WriteString("- Heuristic: merge-worthy\n")
		} else {
			sb.WriteString("- Heuristic: not merge-worthy\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Based on the code and structural context above, please answer:

- Are these functions likely to change together?
- Do they encode the same business rule?
- Would a shared abstraction be simpler than two copies?
- Would merging reduce bugs without increasing coordination cost?
- Can we share a lower-level primitive instead of merging the whole function?
- Given the structural evidence, should these functions be merged? Why or why not?
`)

	return sb.String(), nil
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
