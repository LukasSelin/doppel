package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	ChannelK   *int     `json:"channel-k,omitempty"`
	Debug      *bool    `json:"debug,omitempty"`
	MaxPerFunc *int     `json:"max-per-func,omitempty"`
	Tests      *string  `json:"tests,omitempty"`
	Generated  *string  `json:"generated,omitempty"`
	Calibrate  *float64 `json:"calibrate,omitempty"`
	Format     *string  `json:"format,omitempty"`
	Families   *int     `json:"families,omitempty"`
	FamilyMin  *float64 `json:"family-min,omitempty"`
	HookNotify *string  `json:"hook-notify,omitempty"`
}

// Hook notification modes. These decide who a Stop hook's findings reach, and
// they are the one setting with a cost attached: reaching the agent means the
// turn continues, because a Stop hook cannot put text in the model's context
// any other way. See hookNotify.
const (
	NotifyAgent = "agent" // additionalContext to the model, plus a line for the user
	NotifyUser  = "user"  // systemMessage only; never continues a turn
	NotifyOff   = "off"   // silence
)

// hookNotify reads the notification mode for a hook run.
//
// Deliberately not part of Params. Params is compared field-by-field to decide
// whether a baseline still answers the same question, so anything in it
// invalidates a baseline when it changes. Who gets told about a finding has no
// bearing on what was measured, and switching modes mid-session must not throw
// away the session's origin.
func hookNotify(root string) (string, error) {
	cfg, err := loadConfig(filepath.Join(root, ".doppel.json"))
	if err != nil {
		return NotifyAgent, err
	}
	if cfg == nil || cfg.HookNotify == nil {
		return NotifyAgent, nil
	}
	mode := *cfg.HookNotify
	switch mode {
	case NotifyAgent, NotifyUser, NotifyOff:
		return mode, nil
	}
	return NotifyAgent, fmt.Errorf("invalid hook-notify value %q: want agent, user, or off", mode)
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
	if cfg.ChannelK != nil {
		set("channel-k", strconv.Itoa(*cfg.ChannelK))
	}
	if cfg.Debug != nil {
		set("debug", strconv.FormatBool(*cfg.Debug))
	}
	if cfg.MaxPerFunc != nil {
		set("max-per-func", strconv.Itoa(*cfg.MaxPerFunc))
	}
	if cfg.Tests != nil {
		set("tests", *cfg.Tests)
	}
	if cfg.Generated != nil {
		set("generated", *cfg.Generated)
	}
	if cfg.Calibrate != nil {
		set("calibrate", strconv.FormatFloat(*cfg.Calibrate, 'f', -1, 64))
	}
	if cfg.Format != nil {
		set("format", *cfg.Format)
	}
	if cfg.Families != nil {
		set("families", strconv.Itoa(*cfg.Families))
	}
	if cfg.FamilyMin != nil {
		set("family-min", strconv.FormatFloat(*cfg.FamilyMin, 'f', -1, 64))
	}
}

// hookParams derives the run parameters for a hook run from the repo's config.
//
// A hook honours .doppel.json for everything that decides what the corpus is —
// threshold, min-nodes, channel-k, and the test and generated populations —
// because a baseline should describe the repo the way its owner has configured
// doppel to see it.
//
// It then overrides everything that only decides what gets *shown*. Top-N, the
// per-function diversity cap and the struct-min filter drop pairs for
// presentation reasons, and a pair that vanishes from a report because it fell
// past rank 20 has not changed; reporting it as an impact would be a lie. The
// hook therefore always diffs the full candidate set.
func hookParams(root string) (Params, error) {
	p := Params{
		Threshold:  0.60,
		MinNodes:   12,
		ChannelK:   5,
		TestsMode:  "exclude",
		Generated:  "exclude",
		TopN:       0,
		MaxPerFunc: 0,
		StructMin:  0,
	}
	cfg, err := loadConfig(filepath.Join(root, ".doppel.json"))
	if err != nil {
		return p, err
	}
	if cfg == nil {
		return p, nil
	}
	if cfg.Threshold != nil {
		p.Threshold = *cfg.Threshold
	}
	if cfg.MinNodes != nil {
		p.MinNodes = *cfg.MinNodes
	}
	if cfg.ChannelK != nil {
		p.ChannelK = *cfg.ChannelK
	}
	if cfg.Tests != nil {
		p.TestsMode = *cfg.Tests
	}
	if cfg.Generated != nil {
		p.Generated = *cfg.Generated
	}
	if cfg.Calibrate != nil {
		p.Calibrate = *cfg.Calibrate
	}
	if err := validateMode("tests", p.TestsMode); err != nil {
		return p, err
	}
	return p, validateMode("generated", p.Generated)
}
