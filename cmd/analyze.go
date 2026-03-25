package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/embedder"
	"github.com/lukse/doppel/internal/mapper"
	"github.com/lukse/doppel/internal/parser"
	"github.com/lukse/doppel/internal/reflector"
	"github.com/lukse/doppel/internal/reporter"
	"github.com/lukse/doppel/internal/tagger"
	"github.com/spf13/cobra"
)

var (
	threshold         float64
	topN              int
	model             string
	ollamaURL         string
	cacheFile         string
	maxInputBytes     int
	ollamaNumCtx      int
	reflectModel      string
	outputFile        string
	conceptModel      string
	configFile        string
	conceptPromptFile string
	reflectPromptFile string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <path>",
	Short: "Analyze a codebase for semantically similar functions",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := configFile
		if path == "" {
			path = ".doppel.json"
		}
		cfg, err := loadConfig(path)
		if err != nil {
			return err
		}
		if cfg != nil {
			applyConfig(cmd, cfg)
		}
		return nil
	},
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.85, "Minimum similarity score (0.0–1.0)")
	analyzeCmd.Flags().IntVarP(&topN, "top", "n", 20, "Maximum number of pairs to show")
	analyzeCmd.Flags().StringVarP(&model, "model", "m", "nomic-embed-text", "Ollama embedding model")
	analyzeCmd.Flags().StringVar(&ollamaURL, "ollama-url", "http://localhost:11434", "Ollama base URL")
	analyzeCmd.Flags().StringVar(&cacheFile, "cache", ".embeddings.json", "Embedding cache file path (empty to disable)")
	analyzeCmd.Flags().IntVar(&maxInputBytes, "max-input", 8192, "Max UTF-8 bytes of each function body sent to the embedder (auto-shrinks on context errors)")
	analyzeCmd.Flags().IntVar(&ollamaNumCtx, "ollama-num-ctx", 0, "Ollama options.num_ctx (tokens); 0 = server default. Use 32768 for Qwen3-Embedding-8B long context (see HF model card)")
	analyzeCmd.Flags().StringVar(&reflectModel, "reflect-model", "", "Ollama chat model for merge explanations (e.g. llama3.2, qwen2.5). Empty = disabled.")
	analyzeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report as markdown to this file (e.g. report.md). Stdout text report is still printed.")
	analyzeCmd.Flags().StringVar(&conceptModel, "concept-model", "", "Ollama chat model for concept doc generation (e.g. llama3.2). Empty = static analysis only.")
	analyzeCmd.Flags().StringVar(&configFile, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	analyzeCmd.Flags().StringVar(&conceptPromptFile, "concept-prompt-file", "", "Path to a text/template file for the concept prompt. Variables: {{.Name}}, {{.Package}}, {{.Signature}}, {{.Language}}, {{.Patterns}}, {{.Body}}")
	analyzeCmd.Flags().StringVar(&reflectPromptFile, "reflect-prompt-file", "", "Path to a text/template file for the reflect prompt. Variables: {{.Score}}, {{.A.Name}}, {{.A.Body}}, {{.B.Name}}, {{.B.Body}}, etc.")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	root := args[0]

	fmt.Fprintf(os.Stderr, "Scanning %s ...\n", root)
	var units []parser.CodeUnit
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		parsed, err := parser.Parse(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warn: %s: %v\n", path, err)
			return nil
		}
		units = append(units, parsed...)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	if len(units) == 0 {
		fmt.Println("No functions found.")
		return nil
	}

	for i := range units {
		units[i].Patterns = tagger.Tag(units[i].Body)
	}

	// Build call graph and generate concept documents for every unit.
	cg := concepter.BuildCallGraph(units)

	reflectPrompt, err := readPromptFile(reflectPromptFile)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Generating concept documents...\n")
	cptr := concepter.New()
	docs := mapper.Map(units, cg, cptr)
	conceptTexts := make([]string, len(docs))
	for i, doc := range docs {
		conceptTexts[i] = doc.Format()
	}

	fmt.Fprintf(os.Stderr, "Found %d functions. Generating embeddings...\n", len(units))

	emb, err := embedder.New(ollamaURL, model, cacheFile, ollamaNumCtx)
	if err != nil {
		return err
	}

	embeddings := make([][]float64, len(units))
	for i, u := range units {
		vec, err := embedWithBackoff(emb, conceptTexts[i], maxInputBytes)
		if err != nil {
			return fmt.Errorf("embed %s:%s: %w", u.File, u.Name, err)
		}
		embeddings[i] = vec
		if (i+1)%10 == 0 || i+1 == len(units) {
			fmt.Fprintf(os.Stderr, "  embedded %d/%d\r", i+1, len(units))
		}
	}
	fmt.Fprintln(os.Stderr)

	if err := emb.SaveCache(); err != nil {
		fmt.Fprintf(os.Stderr, "  warn: could not save cache: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Computing similarity...\n")
	pairs := analyzer.FindSimilar(units, embeddings, threshold, topN)

	if reflectModel != "" && len(pairs) > 0 {
		fmt.Fprintf(os.Stderr, "Reflecting on %d pairs with model %q...\n", len(pairs), reflectModel)
		ref := reflector.New(ollamaURL, reflectModel, reflectPrompt)
		for i := range pairs {
			fmt.Fprintf(os.Stderr, "  reflecting %d/%d\r", i+1, len(pairs))
			explanation, err := ref.Explain(pairs[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n  warn: reflect pair %d: %v\n", i+1, err)
				continue
			}
			pairs[i].Explanation = explanation
		}
		fmt.Fprintln(os.Stderr)
	}

	reporter.Print(os.Stdout, pairs, threshold, len(units))

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		reporter.PrintMarkdown(f, pairs, threshold, len(units))
		fmt.Fprintf(os.Stderr, "Markdown report written to %s\n", outputFile)
	}

	return nil
}


const minEmbedBytes = 256

// embedWithBackoff sends truncated body text; on Ollama context-length errors, retries with half the byte limit until it succeeds or cannot shrink further.
func embedWithBackoff(emb *embedder.Embedder, body string, maxBytes int) ([]float64, error) {
	if maxBytes < minEmbedBytes {
		maxBytes = minEmbedBytes
	}
	n := maxBytes
	for {
		text := truncateUTF8(body, n)
		vec, err := emb.Embed(text)
		if err == nil {
			return vec, nil
		}
		if !isOllamaContextLengthError(err) || len(text) <= minEmbedBytes {
			return nil, err
		}
		next := len(text) / 2
		if next < minEmbedBytes {
			next = minEmbedBytes
		}
		if next >= len(text) {
			return nil, err
		}
		n = next
	}
}

func isOllamaContextLengthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context length") || strings.Contains(msg, "exceeds the context")
}

// truncateUTF8 caps s to at most maxBytes UTF-8 bytes without splitting a rune.
func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8.RuneStart(s[len(s)-1]) {
		s = s[:len(s)-1]
	}
	return s
}

// readPromptFile reads a prompt template file and returns its contents.
// Returns "" without error if path is empty.
func readPromptFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return string(b), nil
}

func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git": true, ".claude": true, "node_modules": true, "vendor": true,
		".venv": true, "__pycache__": true, "dist": true, "build": true,
		".idea": true, ".vscode": true,
	}
	return skip[name]
}
