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
// Keys mirror the flag names, so they are kebab-case, not snake_case.
// Pointer fields let us distinguish "not set" from zero values.
type AnalysisConfig struct {
	Threshold  *float64 `json:"threshold,omitempty"`
	TopN       *int     `json:"top,omitempty"`
	MinNodes   *int     `json:"min-nodes,omitempty"`
	StructMin  *float64 `json:"struct-min,omitempty"`
	OutputFile *string  `json:"output,omitempty"`
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
	if cfg.MinNodes != nil {
		set("min-nodes", strconv.Itoa(*cfg.MinNodes))
	}
	if cfg.StructMin != nil {
		set("struct-min", strconv.FormatFloat(*cfg.StructMin, 'f', -1, 64))
	}
	if cfg.OutputFile != nil {
		set("output", *cfg.OutputFile)
	}
}
