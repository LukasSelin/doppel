package concepter

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	gparser "github.com/lukse/doppel/internal/parser"
)

// llmConceptResponse is the JSON structure the LLM is asked to return.
type llmConceptResponse struct {
	Summary      string   `json:"summary"`
	Inputs       []string `json:"inputs"`
	Outputs      []string `json:"outputs"`
	Dependencies []string `json:"dependencies"`
	Patterns     []string `json:"patterns"`
}

// buildConceptPrompt constructs the prompt sent to Ollama /api/generate.
func buildConceptPrompt(unit gparser.CodeUnit) string {
	var sb strings.Builder

	detectedPatterns := "none"
	if len(unit.Patterns) > 0 {
		detectedPatterns = strings.Join(unit.Patterns, ", ")
	}

	sb.WriteString("Analyze this function. Reply with ONLY a JSON object, no explanation.\n\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", unit.Name))
	if unit.Package != "" {
		sb.WriteString(fmt.Sprintf("Package: %s\n", unit.Package))
	}
	if unit.Signature != "" {
		sb.WriteString(fmt.Sprintf("Signature: %s\n", unit.Signature))
	}
	sb.WriteString(fmt.Sprintf("Language: %s\n", unit.Language))
	sb.WriteString(fmt.Sprintf("Detected patterns: %s\n", detectedPatterns))
	sb.WriteString("\nSource:\n```\n")
	sb.WriteString(unit.Body)
	sb.WriteString("\n```\n")
	sb.WriteString(`
JSON schema:
{
  "summary": "one sentence describing what this function does",
  "inputs":       ["one type per item, no parameter names"],
  "outputs":      ["one type per item"],
  "dependencies": ["external packages, services, or subsystems used"],
  "patterns":     ["design patterns or idioms, e.g. retry loop, error wrapping, db_access"]
}

JSON:`)
	return sb.String()
}

// parseConceptResponse extracts a JSON object from the LLM response string.
// It strips common markdown code fences and tolerates trailing prose.
func parseConceptResponse(raw string) (llmConceptResponse, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown fences: ```json ... ``` or ``` ... ```
	for _, fence := range []string{"```json", "```"} {
		if idx := strings.Index(raw, fence); idx != -1 {
			raw = raw[idx+len(fence):]
			if end := strings.Index(raw, "```"); end != -1 {
				raw = raw[:end]
			}
			raw = strings.TrimSpace(raw)
			break
		}
	}

	// Find the first '{' and last '}' to isolate the JSON object.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return llmConceptResponse{}, fmt.Errorf("no JSON object found in response")
	}
	raw = raw[start : end+1]

	var resp llmConceptResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return llmConceptResponse{}, fmt.Errorf("unmarshal concept response: %w", err)
	}
	return resp, nil
}

// parseGoSignature splits a Go signature string like
//
//	"(ctx context.Context, id string) (User, error)"
//
// into input types ["context.Context", "string"] and output types ["User", "error"].
// Parameter names are stripped; only types are kept.
// Falls back to a simple comma-split on parse error.
func parseGoSignature(sig string) (inputs, outputs []string) {
	if sig == "" {
		return nil, nil
	}

	// Wrap in a synthetic function to let go/parser handle it.
	src := fmt.Sprintf("package p\nfunc _placeholder%s{}", sig)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return splitSignatureFallback(sig)
	}

	decls := f.Decls
	if len(decls) == 0 {
		return splitSignatureFallback(sig)
	}
	fn, ok := decls[0].(*ast.FuncDecl)
	if !ok || fn.Type == nil {
		return splitSignatureFallback(sig)
	}

	inputs = extractFieldTypes(fn.Type.Params)
	outputs = extractFieldTypes(fn.Type.Results)
	return inputs, outputs
}

// extractFieldTypes returns the type strings for each field in a field list,
// expanding unnamed multi-name fields (e.g. "a, b int" → ["int", "int"]).
func extractFieldTypes(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	var types []string
	for _, field := range fl.List {
		typeStr := typeExprString(field.Type)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			types = append(types, typeStr)
		}
	}
	return types
}

// typeExprString returns a readable string for an AST type expression.
func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[...]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.ChanType:
		return "chan " + typeExprString(t.Value)
	default:
		return "any"
	}
}

// splitSignatureFallback is a best-effort comma-split used when AST parsing fails.
func splitSignatureFallback(sig string) (inputs, outputs []string) {
	// Find the boundary between params and returns: last ')' before the final '('.
	parts := strings.SplitN(sig, ") (", 2)
	if len(parts) == 2 {
		inputs = splitTypes(strings.Trim(parts[0], "( )"))
		outputs = splitTypes(strings.Trim(parts[1], "( )"))
		return
	}
	// Single return: everything in the first parens is inputs.
	inner := strings.Trim(sig, "() ")
	inputs = splitTypes(inner)
	return
}

func splitTypes(s string) []string {
	if s == "" {
		return nil
	}
	var types []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// If "name type", take the last word as the type.
		fields := strings.Fields(part)
		types = append(types, fields[len(fields)-1])
	}
	return types
}
