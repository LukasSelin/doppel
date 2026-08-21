package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// analyzeFlagNames are every flag applyConfig can touch.
var analyzeFlagNames = []string{"threshold", "top", "min-nodes", "struct-min", "output", "channel-k", "debug", "max-per-func", "tests", "format"}

// resetAnalyzeFlags restores analyzeCmd to its registered defaults. The command
// is a package-level singleton, so tests must not leak state into each other.
func resetAnalyzeFlags(t *testing.T) {
	t.Helper()
	for _, name := range analyzeFlagNames {
		f := analyzeCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag %q is not registered", name)
		}
		if err := f.Value.Set(f.DefValue); err != nil {
			t.Fatalf("reset flag %q: %v", name, err)
		}
		f.Changed = false
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".doppel.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadConfigMissingFileIsNotAnError(t *testing.T) {
	cfg, err := loadConfig(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("loadConfig on missing file: %v", err)
	}
	if cfg != nil {
		t.Errorf("cfg = %+v, want nil", cfg)
	}
}

func TestLoadConfigMalformedJSONIsAnError(t *testing.T) {
	if _, err := loadConfig(writeConfig(t, "{not json")); err == nil {
		t.Error("loadConfig on malformed JSON returned no error")
	}
}

func TestApplyConfigSetsUntouchedFlags(t *testing.T) {
	resetAnalyzeFlags(t)
	t.Cleanup(func() { resetAnalyzeFlags(t) })

	cfg, err := loadConfig(writeConfig(t, `{"threshold":0.9,"top":5,"min-nodes":30,"struct-min":0.5,"output":"r.md","channel-k":9,"debug":true,"max-per-func":4,"tests":"only"}`))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	applyConfig(analyzeCmd, cfg)

	if threshold != 0.9 {
		t.Errorf("threshold = %v, want 0.9", threshold)
	}
	if topN != 5 {
		t.Errorf("top = %v, want 5", topN)
	}
	if minNodes != 30 {
		t.Errorf("min-nodes = %v, want 30", minNodes)
	}
	if structMin != 0.5 {
		t.Errorf("struct-min = %v, want 0.5", structMin)
	}
	if outputFile != "r.md" {
		t.Errorf("output = %q, want r.md", outputFile)
	}
	if channelK != 9 {
		t.Errorf("channel-k = %v, want 9", channelK)
	}
	if !debugFlag {
		t.Errorf("debug = %v, want true", debugFlag)
	}
	if maxPerFunc != 4 {
		t.Errorf("max-per-func = %v, want 4", maxPerFunc)
	}
	if testsMode != "only" {
		t.Errorf("tests = %q, want only", testsMode)
	}
}

// An explicit CLI flag must beat the config file.
func TestApplyConfigDoesNotOverrideExplicitFlags(t *testing.T) {
	resetAnalyzeFlags(t)
	t.Cleanup(func() { resetAnalyzeFlags(t) })

	// Flags().Set marks the flag as Changed, which is how cobra records that
	// the user passed it on the command line.
	if err := analyzeCmd.Flags().Set("threshold", "0.6"); err != nil {
		t.Fatalf("set threshold: %v", err)
	}

	cfg, err := loadConfig(writeConfig(t, `{"threshold":0.9,"top":5}`))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	applyConfig(analyzeCmd, cfg)

	if threshold != 0.6 {
		t.Errorf("threshold = %v, want 0.6: the CLI flag should win", threshold)
	}
	if topN != 5 {
		t.Errorf("top = %v, want 5: an untouched flag should still take the config value", topN)
	}
}

// Unknown keys are ignored rather than rejected, so an old config file does not
// break the run.
func TestApplyConfigIgnoresUnknownKeys(t *testing.T) {
	resetAnalyzeFlags(t)
	t.Cleanup(func() { resetAnalyzeFlags(t) })

	cfg, err := loadConfig(writeConfig(t, `{"reflect-model":"llama3.2","threshold":0.75}`))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	applyConfig(analyzeCmd, cfg)

	if threshold != 0.75 {
		t.Errorf("threshold = %v, want 0.75", threshold)
	}
}

// The format key exists so a repo can pin machine-readable output in
// .doppel.json, and it follows the same precedence rule as every other key.
func TestApplyConfigFormat(t *testing.T) {
	resetAnalyzeFlags(t)
	path := writeConfig(t, `{"format":"json"}`)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	applyConfig(analyzeCmd, cfg)

	if got := analyzeCmd.Flags().Lookup("format").Value.String(); got != "json" {
		t.Errorf("format = %q, want json", got)
	}
}

func TestFormatFlagWinsOverConfig(t *testing.T) {
	resetAnalyzeFlags(t)
	if err := analyzeCmd.Flags().Set("format", "text"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	path := writeConfig(t, `{"format":"json"}`)

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	applyConfig(analyzeCmd, cfg)

	if got := analyzeCmd.Flags().Lookup("format").Value.String(); got != "text" {
		t.Errorf("format = %q, want the explicit flag value text", got)
	}
}

// hookParams reads the corpus-defining keys from .doppel.json but must ignore
// the presentation keys: a pair that fell past rank 20 has not changed, and
// reporting it as this session's impact would be a lie.
func TestHookParamsHonorsCorpusKeysAndIgnoresPresentation(t *testing.T) {
	dir := t.TempDir()
	body := `{"threshold":0.8,"min-nodes":20,"channel-k":9,"tests":"include","top":5,"max-per-func":3,"struct-min":0.7}`
	if err := os.WriteFile(filepath.Join(dir, ".doppel.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	p, err := hookParams(dir)
	if err != nil {
		t.Fatalf("hookParams: %v", err)
	}

	if p.Threshold != 0.8 || p.MinNodes != 20 || p.ChannelK != 9 || p.TestsMode != "include" {
		t.Errorf("corpus-defining keys not honored: %+v", p)
	}
	if p.TopN != 0 || p.MaxPerFunc != 0 || p.StructMin != 0 {
		t.Errorf("presentation keys leaked into a hook run: %+v", p)
	}
}

func TestHookParamsWithoutConfig(t *testing.T) {
	p, err := hookParams(t.TempDir())
	if err != nil {
		t.Fatalf("hookParams with no config: %v", err)
	}
	if p.Threshold != 0.60 || p.MinNodes != 12 || p.ChannelK != 5 || p.TestsMode != "exclude" {
		t.Errorf("defaults wrong: %+v", p)
	}
}

func TestHookParamsRejectsBadTestsMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".doppel.json"), []byte(`{"tests":"sometimes"}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := hookParams(dir); err == nil {
		t.Error("hookParams accepted an invalid tests mode")
	}
}
