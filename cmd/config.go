package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// AnalysisConfig holds optional overrides for analyze flags.
// Keys mirror the flag names, so they are kebab-case, not snake_case.
// Pointer fields let us distinguish "not set" from zero values.
type AnalysisConfig struct {
	Threshold  *float64  `json:"threshold,omitempty"`
	TopN       *int      `json:"top,omitempty"`
	MinNodes   *int      `json:"min-nodes,omitempty"`
	StructMin  *float64  `json:"struct-min,omitempty"`
	OutputFile *string   `json:"output,omitempty"`
	ChannelK   *int      `json:"channel-k,omitempty"`
	Debug      *bool     `json:"debug,omitempty"`
	MaxPerFunc *int      `json:"max-per-func,omitempty"`
	Tests      *string   `json:"tests,omitempty"`
	Generated  *string   `json:"generated,omitempty"`
	Languages  *[]string `json:"languages,omitempty"`
	Calibrate  *float64  `json:"calibrate,omitempty"`
	Format     *string   `json:"format,omitempty"`
	Families   *int      `json:"families,omitempty"`
	FamilyMin  *float64  `json:"family-min,omitempty"`
	MapMetric  *string   `json:"map-metric,omitempty"`
	HookNotify *string   `json:"hook-notify,omitempty"`
}

// defaultCalibrateRate is the fraction of random unrelated pairs a run's
// thresholds may admit, and the default for --calibrate.
//
// The thresholds it derives replace fixed numbers that could not mean the same
// thing twice: a 0.60 code-shape floor is loose on a corpus of 81 functions and
// strict on one of 8000, so every user inherited an operating point calibrated
// for somebody else's repo. A rate is corpus-relative by construction — "admit
// 1% of random pairs" is the same question at any size — which is what lets the
// numeric flags stop being something an end user has to reason about.
//
// 0.01 rather than a looser rate because it is the rate the golden benchmark
// measured: neutral on cobra's labels at every rate from 0.005 to 0.05, with the
// candidate set growing from 816 to 1029 without a labeled pair changing rank.
const defaultCalibrateRate = 0.01

// defaultMinNodes is the body-size floor for the structural retrieval channel,
// and the default for --min-nodes on every command that registers it.
//
// It is a separate knob from the calibrated thresholds and deliberately stays
// one: calibration derives a *score* floor from the corpus's null distribution,
// while this is an *eligibility* rule about which functions the shape channel
// indexes at all. Calibration cannot subsume it — a corpus of accessors has a
// null distribution made of accessors, so a rate would happily admit them.
//
// 16 rather than the historical 12 because the shape channel indexes
// Weisfeiler-Lehman labels now. A WL bag over a canonicalized body carries
// structure a token 3-gram bag did not, so small bodies produce labels that
// look distinctive without being informative. 16 rather than the first
// recalibration's 18 because the pins are one node apart: cobra's labeled
// one-line false positive (`Less`/`Less`) is 15 nodes and conc's genuine
// generic-wrapper clones are 16 — measured on the min-nodes ladder
// (TestMinNodesLadder), 16/17/18 score identically on the labels while 16
// reopens conc's shape channel (3 → 26 admissions, its lost 1.00 clone pair
// returns).
const defaultMinNodes = 16

// defaultThreshold is the static code-shape floor used only when calibration
// is off (--threshold pinned) or declined (a corpus too small to calibrate).
// 0.38 is the median of the six ladder corpora that calibrate at the default
// rate under the WL metric (prometheus 0.33, moby 0.35, hugo 0.35, gin 0.41,
// cobra 0.44, chi 0.45; conc declines); cobra's labels are flat across
// 0.30–0.60, so the choice is nearly free where it can be measured.
const defaultThreshold = 0.38

// calibrationOptOut turns calibration off for a run that pinned a threshold by
// hand, and reports whether it did.
//
// Calibration replaces --threshold and --struct-min outright — a half-calibrated
// run is the mixed question Params equality exists to forbid — so with it on by
// default, an explicit --threshold would be silently discarded. Opting the whole
// run out instead keeps the all-or-nothing property and lets the explicit flag
// win, which is the precedence a user expects.
//
// It reads Changed rather than the values, so it must run after applyConfig:
// config keys are applied through Flags().Set, which marks them Changed. That is
// deliberate and load-bearing — an existing .doppel.json pinning threshold or
// struct-min keeps its behaviour exactly, and only a config that names
// calibrate explicitly opts back in.
func calibrationOptOut(cmd *cobra.Command, rate *float64) bool {
	if cmd.Flags().Changed("calibrate") {
		return false
	}
	for _, name := range []string{"threshold", "struct-min", "family-min"} {
		if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
			*rate = 0
			return true
		}
	}
	return false
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
	if cfg.Languages != nil {
		set("languages", strings.Join(*cfg.Languages, ","))
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
	if cfg.MapMetric != nil {
		set("map-metric", *cfg.MapMetric)
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
		Threshold:       defaultThreshold,
		MinNodes:        defaultMinNodes,
		ChannelK:        5,
		TestsMode:       "exclude",
		Generated:       "exclude",
		TopN:            0,
		MaxPerFunc:      0,
		StructMin:       0,
		NoOverlapFilter: true,
		Calibrate:       defaultCalibrateRate,
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
	// Corpus-defining, like tests and generated: which languages are read
	// decides what the population is, so a hook must measure the same one.
	if cfg.Languages != nil {
		p.Languages = *cfg.Languages
	}
	if cfg.Calibrate != nil {
		p.Calibrate = *cfg.Calibrate
	}
	if err := validateMode("tests", p.TestsMode); err != nil {
		return p, err
	}
	return p, validateMode("generated", p.Generated)
}
