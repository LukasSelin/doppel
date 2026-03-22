package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// AnalysisConfig holds optional overrides for analyze flags.
// Pointer fields let us distinguish "not set" from zero values.
type AnalysisConfig struct {
	Threshold    *float64 `json:"threshold,omitempty"`
	TopN         *int     `json:"top,omitempty"`
	Model        *string  `json:"model,omitempty"`
	OllamaURL    *string  `json:"ollama-url,omitempty"`
	CacheFile    *string  `json:"cache,omitempty"`
	MaxInput     *int     `json:"max-input,omitempty"`
	OllamaNumCtx *int     `json:"ollama-num-ctx,omitempty"`
	ReflectModel *string  `json:"reflect-model,omitempty"`
	OutputFile   *string  `json:"output,omitempty"`
	ConceptModel      *string `json:"concept-model,omitempty"`
	ConceptCache      *string `json:"concept-cache,omitempty"`
	ConceptPromptFile *string `json:"concept-prompt-file,omitempty"`
	ReflectPromptFile *string `json:"reflect-prompt-file,omitempty"`
}

// loadConfig reads a JSON config file. Returns nil (no error) if the file does not exist.
func loadConfig(path string) (*AnalysisConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg AnalysisConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// applyConfig sets flag values from cfg, skipping any flag the user already set explicitly.
func applyConfig(cmd *cobra.Command, cfg *AnalysisConfig) {
	set := func(name, value string) {
		if !cmd.Flags().Changed(name) {
			_ = cmd.Flags().Set(name, value)
		}
	}
	if cfg.Threshold != nil {
		set("threshold", strconv.FormatFloat(*cfg.Threshold, 'f', -1, 64))
	}
	if cfg.TopN != nil {
		set("top", strconv.Itoa(*cfg.TopN))
	}
	if cfg.Model != nil {
		set("model", *cfg.Model)
	}
	if cfg.OllamaURL != nil {
		set("ollama-url", *cfg.OllamaURL)
	}
	if cfg.CacheFile != nil {
		set("cache", *cfg.CacheFile)
	}
	if cfg.MaxInput != nil {
		set("max-input", strconv.Itoa(*cfg.MaxInput))
	}
	if cfg.OllamaNumCtx != nil {
		set("ollama-num-ctx", strconv.Itoa(*cfg.OllamaNumCtx))
	}
	if cfg.ReflectModel != nil {
		set("reflect-model", *cfg.ReflectModel)
	}
	if cfg.OutputFile != nil {
		set("output", *cfg.OutputFile)
	}
	if cfg.ConceptModel != nil {
		set("concept-model", *cfg.ConceptModel)
	}
	if cfg.ConceptCache != nil {
		set("concept-cache", *cfg.ConceptCache)
	}
	if cfg.ConceptPromptFile != nil {
		set("concept-prompt-file", *cfg.ConceptPromptFile)
	}
	if cfg.ReflectPromptFile != nil {
		set("reflect-prompt-file", *cfg.ReflectPromptFile)
	}
}
