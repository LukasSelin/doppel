package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	timelineOutput string
	timelineFormat string
	timelineLabels []string
	timelineTarget string
)

// The payload bounds. A timeline carries N runs' findings, so the caps are
// per-step rather than per-file, and every one of them is reported on the page
// — see dashboard.TimelineBounds for why a silent truncation is the one
// misreading a history page must not invite.
const (
	maxStepChanges = 200 // classified findings listed per transition
	maxStepPairs   = 60  // created (and separately dissolved) pairs per transition
	maxTracks      = 400 // function lifelines drawn
)

var timelineCmd = &cobra.Command{
	Use:   "timeline <a.json> <b.json> [more.json...]",
	Short: "Step through a series of analysis snapshots as one history",
	Long: `Reads snapshots written by ` + "`doppel analyze --format json`" + ` and renders them as
one page you can step through: what happened to every function at each
revision, which near-duplicate pairs those changes created or dissolved, and
each function's lifeline across the whole series.

Argument order is series order. A snapshot carries no timestamp — by design,
since a wall-clock inside one would break byte-identical reproducibility — so
nothing here can sort them and the caller says what the order is.

Doppel reads no git history, so producing the series is not this command's
job. Analyse each revision yourself and pass the files in; ` + "`task timeline`" + ` is a
worked example of doing that over a git repository.

# Every step must share one operating point

Nearly every number doppel produces is corpus-relative, and calibration is on
by default — so a series of independently calibrated runs has a different
threshold per revision and its pair sets are not comparable step to step. This
command refuses such a series rather than drawing it. Analyse every revision at
the same explicit --threshold and --struct-min, which turns calibration off.

Even pinned, the learned concept vocabulary, roles, habitat fit and the
nearest-neighbour percentiles remain properties of each revision's own corpus.
The page states that; it does not draw a trend line through them.

Exit codes: 0 rendered, 1 a file could not be read, 2 the series refuses
comparison (mismatched schema, canon rule set, or operating point).`,
	Example: `  doppel timeline runs/*.json -o timeline.html
  doppel timeline old.json new.json --format json`,
	Args:          cobra.MinimumNArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runTimeline(cmd, args)
		if err == nil {
			return nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		// Shared with `doppel diff`: one definition of what each code means,
		// so two commands reading the same files cannot report differently.
		diffExit(diffExitCode(err))
		return err
	},
}

func init() {
	timelineCmd.Flags().StringVarP(&timelineOutput, "output", "o", "", "Write the timeline page to this file (HTML)")
	timelineCmd.Flags().StringVar(&timelineFormat, "format", "text", "Output format for stdout: text or json")
	timelineCmd.Flags().StringSliceVar(&timelineLabels, "labels", nil, "Step labels, in order; defaults to each file's base name")
	timelineCmd.Flags().StringVar(&timelineTarget, "target", "", "Name of the analysed project; defaults to the working directory's name")
	rootCmd.AddCommand(timelineCmd)
}

func runTimeline(cmd *cobra.Command, args []string) error {
	if timelineFormat != "text" && timelineFormat != "json" {
		return fmt.Errorf("--format must be text or json, got %q", timelineFormat)
	}
	if len(timelineLabels) != 0 && len(timelineLabels) != len(args) {
		return fmt.Errorf("--labels has %d entries for %d snapshots", len(timelineLabels), len(args))
	}

	snaps := make([]snapshot.Snapshot, len(args))
	for i, path := range args {
		s, err := readSnapshot(path)
		if err != nil {
			return err
		}
		snaps[i] = s
	}

	// The operating-point check is this command's own, deliberately stricter
	// than internal/identity. That package allows a Params mismatch and notes
	// it, because a two-run classification reads only Units — keys, digests
	// and bags, all properties of a function's own AST. A timeline is a
	// stronger claim: it puts each run's pair counts and corpus metrics on one
	// axis, and those are exactly the corpus-relative numbers a moved
	// threshold invalidates.
	if why := sameOperatingPoint(args, snaps); why != "" {
		return incomparableError{reason: why}
	}

	series, err := identity.Chain(snaps, identity.Options{})
	if err != nil {
		return err
	}
	if !series.Comparable {
		return incomparableError{reason: series.Reason}
	}

	warnReportCaps(cmd.ErrOrStderr(), snaps[0].Params)

	labels := stepLabels(args)
	p := buildTimeline(snaps, series, labels)

	if timelineOutput != "" {
		if err := writeTimelinePage(cmd, p); err != nil {
			return err
		}
	}
	if timelineFormat == "json" {
		return dashboard.WriteTimelineJSON(cmd.OutOrStdout(), p)
	}
	printTimelineText(cmd.OutOrStdout(), p)
	return nil
}

func writeTimelinePage(cmd *cobra.Command, p dashboard.TimelinePayload) error {
	f, err := os.Create(timelineOutput)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()
	if err := dashboard.PrintTimeline(f, p); err != nil {
		return err
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Timeline written to %s\n", timelineOutput)
	return nil
}

// sameOperatingPoint reports why the series cannot be read as one history, or
// "" when every step agrees.
//
// It names the first offending file rather than counting them: a mismatched
// series is almost always one run produced by a different command line, and the
// fix is to re-analyse that revision.
func sameOperatingPoint(paths []string, snaps []snapshot.Snapshot) string {
	base := snaps[0].Params
	for i := 1; i < len(snaps); i++ {
		if !base.Equal(snaps[i].Params) {
			return fmt.Sprintf("%s was analysed at a different operating point than %s (%s vs %s); "+
				"re-analyse every revision at the same explicit --threshold and --struct-min",
				paths[i], paths[0], paramsLine(snaps[i].Params), paramsLine(base))
		}
	}
	if base.Calibrate > 0 {
		return fmt.Sprintf("every step calibrated its own thresholds (--calibrate %g), so the runs "+
			"are answers to different questions; re-analyse at an explicit --threshold and --struct-min",
			base.Calibrate)
	}
	return ""
}

// warnReportCaps says so when the series was analysed with the report-time caps
// still on.
//
// It warns rather than refuses, deliberately. A capped series is a true history
// of the pairs it carries and is a perfectly reasonable quick look; what would
// be wrong is letting it read as a complete one. The same caps are why
// cmd/hook.go overrides `top` and `max-per-func` for its own snapshots.
func warnReportCaps(w io.Writer, p snapshot.Params) {
	if p.Top == 0 && p.MaxPerFunc == 0 {
		return
	}
	fmt.Fprintf(w, "Note: these snapshots were written with --top %d and --max-per-func %d, "+
		"so each one stores the ranked report list rather than the full candidate set. "+
		"The pair half of this timeline is bounded by that; re-analyse with "+
		"--top 0 --max-per-func 0 for the whole set.\n", p.Top, p.MaxPerFunc)
}

// paramsLine renders the operating point for a reader, in the order the flags
// are documented in.
func paramsLine(p snapshot.Params) string {
	s := fmt.Sprintf("threshold %.2f · struct-min %.2f · min-nodes %d · channel-k %d · tests %s · generated %s",
		p.Threshold, p.StructMin, p.MinNodes, p.ChannelK, p.TestsMode, p.Generated)
	if len(p.Languages) > 0 {
		s += " · " + strings.Join(p.Languages, ",")
	}
	if len(p.Exclude) > 0 {
		s += " · excluding " + strings.Join(p.Exclude, ",")
	}
	return s
}

// indexPrefix is the counter a series generator puts in front of a file name to
// keep a shell glob in series order. It is a naming convention, not a format:
// stripping it is a courtesy to the label, and a file without one keeps its
// whole base name.
var indexPrefix = regexp.MustCompile(`^[0-9]+[-_]`)

func stepLabels(paths []string) []string {
	if len(timelineLabels) == len(paths) {
		return timelineLabels
	}
	out := make([]string, len(paths))
	for i, p := range paths {
		b := filepath.Base(p)
		b = strings.TrimSuffix(b, filepath.Ext(b))
		b = indexPrefix.ReplaceAllString(b, "")
		if b == "" {
			b = fmt.Sprintf("%d", i)
		}
		out[i] = b
	}
	return out
}

func timelineTargetName() string {
	if timelineTarget != "" {
		return timelineTarget
	}
	wd, err := os.Getwd()
	if err != nil {
		return "timeline"
	}
	return filepath.Base(wd)
}

// buildTimeline flattens the snapshots and the chained series into the page's
// payload.
//
// Nothing is recomputed here. Every classification, score, verdict and
// explanation sentence was settled by internal/identity or stored by the run
// that held the pair; this function selects, bounds and renames.
func buildTimeline(snaps []snapshot.Snapshot, series identity.Series, labels []string) dashboard.TimelinePayload {
	p := dashboard.TimelinePayload{
		Schema: dashboard.TimelineSchema,
		Target: timelineTargetName(),
		Params: paramsLine(snaps[0].Params),
		Notes:  series.Notes,
		Bounds: dashboard.TimelineBounds{
			MaxChanges:       maxStepChanges,
			MaxPairs:         maxStepPairs,
			MaxTracks:        maxTracks,
			ReportTop:        snaps[0].Params.Top,
			ReportMaxPerFunc: snaps[0].Params.MaxPerFunc,
		},
	}

	for i, s := range snaps {
		p.Steps = append(p.Steps, dashboard.Step{
			Index:       i,
			Label:       labels[i],
			Functions:   s.Functions,
			Pairs:       len(s.Pairs),
			MergeWorthy: s.MergeWorthy(),
			Concepts:    len(s.Concepts),
			Compression: compressionOf(s.CorpusMetrics),
			NNScored:    s.CorpusMetrics.NNScored,
			NNP50:       s.CorpusMetrics.NNP50,
			NNP90:       s.CorpusMetrics.NNP90,
			UnusedSeeds: s.UnusedSeeds,
		})
	}

	for i, d := range series.Deltas {
		p.Changes = append(p.Changes, stepChange(i, d))
	}

	p.Tracks, p.Bounds.FlatTracks, p.Bounds.TracksOmitted = timelineTracks(series.Tracks, len(snaps))
	return p
}

func compressionOf(m snapshot.CorpusMetrics) float64 {
	if m.UniqueSubtrees == 0 {
		return 0
	}
	return float64(m.TotalNodes) / float64(m.UniqueSubtrees)
}

func stepChange(i int, d identity.Delta) dashboard.StepChange {
	sc := dashboard.StepChange{From: i, To: i + 1}
	for _, c := range d.Counts {
		sc.Counts = append(sc.Counts, dashboard.ClassCount{Class: string(c.Class), Count: c.Count})
	}

	// `unchanged` is counted and never listed. It is the bulk of any real
	// comparison and the least informative line in it — the same cut
	// identity's own text report makes.
	for _, c := range d.Changes {
		if c.Class == identity.Unchanged {
			continue
		}
		sc.ChangesTotal++
		if len(sc.Changes) < maxStepChanges {
			sc.Changes = append(sc.Changes, changeRow(c))
		}
	}

	sc.CreatedTotal = len(d.Created)
	sc.DissolvedTotal = len(d.Dissolved)
	for _, pc := range d.Created {
		if pc.Attributable() {
			sc.AttributableNew++
		}
	}
	// Both lists arrive in identity's own order — attributable first, then
	// merge-worthy, then score — so a prefix is exactly the bound the page
	// promises: the least corroborated are what falls off, never the
	// attributable ones.
	sc.Created = pairRows(d.Created, maxStepPairs)
	sc.Dissolved = pairRows(d.Dissolved, maxStepPairs)
	return sc
}

func changeRow(c identity.Change) dashboard.ChangeRow {
	r := dashboard.ChangeRow{
		Class:          string(c.Class),
		Jaccard:        c.Jaccard,
		Containment:    c.Containment,
		DigestEqual:    c.DigestEqual,
		NameChanged:    c.NameChanged,
		PackageChanged: c.PackageChanged,
	}
	for _, m := range c.Old {
		r.Old = append(r.Old, m.Key)
	}
	for _, m := range c.New {
		r.New = append(r.New, m.Key)
	}
	// The new side's location is where a reader goes to look. A deletion has
	// no new side, so it cites where the function used to be.
	switch {
	case len(c.New) > 0:
		r.File, r.Line = c.New[0].File, c.New[0].Line
	case len(c.Old) > 0:
		r.File, r.Line = c.Old[0].File, c.Old[0].Line
	}
	return r
}

func pairRows(ps []identity.PairChange, limit int) []dashboard.PairRow {
	if len(ps) > limit {
		ps = ps[:limit]
	}
	out := make([]dashboard.PairRow, 0, len(ps))
	for _, p := range ps {
		out = append(out, dashboard.PairRow{
			A: p.A, B: p.B,
			Score:        p.Score,
			Overlap:      p.Overlap,
			MergeWorthy:  p.MergeWorthy,
			Explain:      p.Explain,
			AClass:       string(p.AClass),
			BClass:       string(p.BClass),
			Attributable: p.Attributable(),
		})
	}
	return out
}

// timelineTracks drops the flat lifelines, orders what is left by how much
// happened to it, and caps the result.
//
// A flat track is one that ran the whole series with nothing but `unchanged` on
// it. Those are the bulk of any history and say nothing; they are counted, not
// listed. The remainder is ordered by event count so the cap takes the quietest
// tracks rather than an arbitrary tail, with the first step and the ID breaking
// ties into a total order — this file must produce byte-identical output for an
// unchanged series.
func timelineTracks(ts []identity.Track, steps int) (rows []dashboard.TimelineTrack, flat, omitted int) {
	type scored struct {
		t      identity.Track
		events int
	}
	var keep []scored
	for _, t := range ts {
		events := 0
		for _, pt := range t.Points {
			if pt.Class != "" && pt.Class != identity.Unchanged {
				events++
			}
		}
		first, last := t.Points[0].Step, t.Points[len(t.Points)-1].Step
		if events == 0 && t.Fate == "" && first == 0 && last == steps-1 {
			flat++
			continue
		}
		keep = append(keep, scored{t: t, events: events})
	}

	sort.SliceStable(keep, func(i, j int) bool {
		if keep[i].events != keep[j].events {
			return keep[i].events > keep[j].events
		}
		fi, fj := keep[i].t.Points[0].Step, keep[j].t.Points[0].Step
		if fi != fj {
			return fi < fj
		}
		return keep[i].t.ID < keep[j].t.ID
	})

	if len(keep) > maxTracks {
		omitted = len(keep) - maxTracks
		keep = keep[:maxTracks]
	}

	for _, s := range keep {
		t := s.t
		row := dashboard.TimelineTrack{
			ID:    t.ID,
			First: t.Points[0].Step,
			Last:  t.Points[len(t.Points)-1].Step,
			Fate:  string(t.Fate),
			Label: t.Points[len(t.Points)-1].Key,
		}
		for _, pt := range t.Points {
			row.Points = append(row.Points, dashboard.TrackStop{
				Step:  pt.Step,
				Key:   pt.Key,
				Class: string(pt.Class),
			})
		}
		rows = append(rows, row)
	}
	return rows, flat, omitted
}
