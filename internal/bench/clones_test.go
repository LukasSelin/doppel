package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// Method 8 of examples/baselines.md: an external clone detector, mapped onto
// function pairs. The detector runs outside this module
// (scripts/clone-baseline.sh, dupl installed into GOBIN), so this file reads
// only the clone groups it wrote:
//
//	DOPPEL_BENCH_BASELINES_CLONES=<dir>   <corpus>.<tag>.clones.json per corpus and threshold
//
// Unset, the method is absent from every table rather than scored as empty.

// cloneMethods are the detector runs scored, in display order. The tool's own
// default threshold is the method-8 row; t=25 is a sensitivity row, fixed
// before the scoring run, because dupl's default reports almost nothing on the
// small rungs.
var cloneMethods = []cloneMethod{
	{"dupl (t=100, default)", "dupl-t100"},
	{"dupl (t=25)", "dupl-t25"},
}

// cloneMethod is one detector run: its display name and the tag its clone file
// carries.
type cloneMethod struct{ name, tag string }

type cloneFile struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Threshold int    `json:"threshold"`
	Corpus    string `json:"corpus"`
	Groups    []struct {
		Fragments []cloneFragment `json:"fragments"`
	} `json:"groups"`
}

type cloneFragment struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

// cloneList is one detector run as function pairs: every pair it implies, with
// its key, sorted by ref.
type cloneList struct {
	name string
	refs []ref
	key  []float64
}

// unitEnd is the last line of a unit's declaration: Body is the source from
// the func keyword to the closing brace, so its newlines count the span.
func unitEnd(u parser.CodeUnit) int {
	return u.StartLine + strings.Count(u.Body, "\n")
}

type unitSpan struct{ idx, start, end int }

// clonePairs maps one clone file onto the run's units. A fragment covers every
// unit whose [StartLine, end] it overlaps, by that many lines; each pair of
// fragments in one group yields every cross pair of the units they cover, keyed
// by the smaller of the two overlaps — the detector's clone size, in lines,
// clipped to the two functions. dupl reports no token count, so lines are the
// size it has. A pair reached through several fragment pairs keeps its largest
// key. Pairs outside one build unit (parser.SameBuildUnit) are dropped, and
// fragments in files outside the population cover nothing.
func clonePairs(br *baselineRun, cf cloneFile) ([]ref, []float64) {
	r := br.run
	byFile := map[string][]unitSpan{}
	for i, u := range r.Units {
		f := snapshot.RelSlash(r.Root, u.File)
		byFile[f] = append(byFile[f], unitSpan{i, u.StartLine, unitEnd(u)})
	}
	type cover struct{ idx, lines int }
	covers := func(fr cloneFragment) []cover {
		var out []cover
		for _, s := range byFile[fr.File] {
			lo, hi := max(fr.Start, s.start), min(fr.End, s.end)
			if lo <= hi {
				out = append(out, cover{s.idx, hi - lo + 1})
			}
		}
		return out
	}
	best := map[ref]float64{}
	for _, g := range cf.Groups {
		cs := make([][]cover, len(g.Fragments))
		for i, fr := range g.Fragments {
			cs[i] = covers(fr)
		}
		for i := range cs {
			for j := i + 1; j < len(cs); j++ {
				for _, a := range cs[i] {
					for _, b := range cs[j] {
						if a.idx == b.idx || !parser.SameBuildUnit(r.Units[a.idx], r.Units[b.idx]) {
							continue
						}
						x := ref{min(a.idx, b.idx), max(a.idx, b.idx)}
						if k := float64(min(a.lines, b.lines)); k > best[x] {
							best[x] = k
						}
					}
				}
			}
		}
	}
	refs := make([]ref, 0, len(best))
	for x := range best {
		refs = append(refs, x)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].a != refs[j].a {
			return refs[i].a < refs[j].a
		}
		return refs[i].b < refs[j].b
	})
	key := make([]float64, len(refs))
	for i, x := range refs {
		key[i] = best[x]
	}
	return refs, key
}

// loadCloneLists reads every clone method's file for one corpus. A missing
// file leaves that method out for the corpus, and says so; a malformed one is
// an error.
func loadCloneLists(br *baselineRun, corpus string) ([]cloneList, []string, error) {
	dir := os.Getenv("DOPPEL_BENCH_BASELINES_CLONES")
	if dir == "" {
		return nil, nil, nil
	}
	return readCloneLists(br, corpus, dir, cloneMethods)
}

// readCloneLists is loadCloneLists over an explicit directory and method set.
func readCloneLists(br *baselineRun, corpus, dir string, methods []cloneMethod) ([]cloneList, []string, error) {
	var out []cloneList
	var notes []string
	for _, c := range methods {
		path := filepath.Join(dir, corpus+"."+c.tag+".clones.json")
		data, err := os.ReadFile(path)
		if err != nil {
			notes = append(notes, fmt.Sprintf("%s: no clone file (%s); method absent for this corpus", c.name, filepath.Base(path)))
			continue
		}
		var cf cloneFile
		if err := json.Unmarshal(data, &cf); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		if cf.Corpus != corpus {
			return nil, nil, fmt.Errorf("%s: written for corpus %q", path, cf.Corpus)
		}
		refs, key := clonePairs(br, cf)
		out = append(out, cloneList{c.name, refs, key})
		notes = append(notes, fmt.Sprintf("%s: %s %s, %d groups -> %d function pairs", c.name, cf.Tool, cf.Version, len(cf.Groups), len(refs)))
	}
	return out, notes, nil
}

// restrictTo keeps the clone pairs inside a pool, so setting A ranks only the
// pool's pairs; pool pairs the detector never reported stay unranked.
func (c cloneList) restrictTo(in map[ref]bool) ([]ref, []float64) {
	var refs []ref
	var key []float64
	for i, x := range c.refs {
		if in[x] {
			refs = append(refs, x)
			key = append(key, c.key[i])
		}
	}
	return refs, key
}
