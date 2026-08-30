package identity

import (
	"fmt"
	"sort"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// A Series is N snapshots read as one run of history: the N-1 deltas between
// consecutive pairs, and one Track per function lifeline across the whole of
// it.
//
// # Why consecutive pairs and not every pair
//
// Compare is a two-snapshot matcher and stays one. A series is that matcher
// applied N-1 times, which is the only reading that keeps every reported number
// exact: a match between step 0 and step 7 inferred by chaining is a claim
// about seven intervening comparisons, not a measurement of one, and the
// weighted Jaccard it would have to print does not exist. So nothing here
// matches across a gap. A Track is the transitive closure of consecutive
// one-to-one matches and says exactly that much.
//
// # What a Track may and may not claim
//
// Only the one-to-one classes join a track: unchanged, edited, renamed, moved.
// Split and merged deliberately do not. A split is one body becoming several
// and a merge is several becoming one, and there is no evidence deciding which
// of the several inherits the lifeline — picking the lowest-keyed part would be
// an arbitrary answer presented as a finding. So a split ends its track and
// starts new ones, the ending is recorded as the track's Fate, and the page can
// say "split into three here" rather than pretending one part continued.
//
// # Refusal
//
// Incomparability is a Series, not an error, mirroring Compare and Since. One
// refusing step refuses the whole series: a page that stepped over a gap would
// be presenting two incommensurable halves as one history. Reason names the
// step so the reader knows which file to look at.
//
// Params, Ontology and build mismatches are *allowed* here exactly as Compare
// allows them, and their notes are collected. Whether a series should be
// stricter than a single comparison is a caller's judgment and lives at the
// command boundary — see cmd/timeline.go, which does refuse on Params, because
// a page plotting a number across steps is making a stronger claim than a
// single diff does.
type Series struct {
	Comparable bool   `json:"comparable"`
	Reason     string `json:"reason,omitempty"`

	// Notes are the allowed mismatches from every step, deduplicated and in
	// first-appearance order.
	Notes []string `json:"notes,omitempty"`

	// Deltas[i] is Since(snaps[i], snaps[i+1]), so there are exactly one
	// fewer than there are snapshots.
	Deltas []Delta `json:"deltas,omitempty"`

	// Tracks are the function lifelines, ordered by first step then first
	// key. Track.ID is the index into this slice.
	Tracks []Track `json:"tracks,omitempty"`
}

// A Track is one function followed across the series.
//
// Points is in ascending step order and is never empty. The steps it covers are
// contiguous by construction — a track is joined only through consecutive
// matches — so a function that vanished and returned under the same name is two
// tracks, which is the honest reading: nothing observed it in between.
type Track struct {
	ID     int          `json:"id"`
	Points []TrackPoint `json:"points"`

	// Fate is the class of the change that ended this track before the last
	// step: deleted, split or merged. Empty when the track reaches the final
	// snapshot.
	Fate Class `json:"fate,omitempty"`
}

// A TrackPoint is one function at one step.
//
// Class is how the point was reached from the step before it — unchanged,
// edited, renamed or moved for a continuation, new or split or merged for the
// first point of a track that began mid-series. It is empty only at step 0,
// where there is no previous step to have arrived from.
type TrackPoint struct {
	Step int    `json:"step"`
	Key  string `json:"key"`

	Class Class `json:"class,omitempty"`
}

// oneToOne is the set of classes that join a track. See the Series doc for why
// split and merged are absent.
func oneToOne(c Class) bool {
	return c == Unchanged || c == Edited || c == Renamed || c == Moved
}

// Chain compares each consecutive pair of snapshots and joins the one-to-one
// matches into tracks.
//
// The snapshots are in series order and that order is the caller's to establish
// — a Snapshot carries no timestamp by design, so nothing here can sort them.
func Chain(snaps []snapshot.Snapshot, opt Options) (Series, error) {
	if len(snaps) < 2 {
		return Series{}, fmt.Errorf("a series needs at least two snapshots, got %d", len(snaps))
	}

	s := Series{Comparable: true, Deltas: make([]Delta, 0, len(snaps)-1)}
	seen := make(map[string]bool)

	for i := 0; i+1 < len(snaps); i++ {
		d, err := Since(snaps[i], snaps[i+1], opt)
		if err != nil {
			return Series{}, fmt.Errorf("step %d to %d: %w", i, i+1, err)
		}
		if !d.Comparable {
			return Series{
				Comparable: false,
				Reason:     fmt.Sprintf("step %d to %d: %s", i, i+1, d.Reason),
			}, nil
		}
		for _, n := range d.Notes {
			if !seen[n] {
				seen[n] = true
				s.Notes = append(s.Notes, n)
			}
		}
		s.Deltas = append(s.Deltas, d)
	}

	s.Tracks = buildTracks(snaps, s.Deltas)
	return s, nil
}

// buildTracks joins every step's one-to-one matches with a union-find over
// (step, key) nodes, then emits one Track per component.
//
// Determinism is by construction rather than by a final sort of everything:
// nodes are numbered in each snapshot's own Units order (which snapshot stores
// sorted by key), unions run in delta order over Changes as classify emitted
// them, and the components are collected by walking the node list in that same
// order. No map decides anything but a lookup.
func buildTracks(snaps []snapshot.Snapshot, deltas []Delta) []Track {
	offset := make([]int, len(snaps))
	total := 0
	index := make([]map[string]int, len(snaps))
	for i, s := range snaps {
		offset[i] = total
		index[i] = make(map[string]int, len(s.Units))
		for j, u := range s.Units {
			// A snapshot's keys are unique by construction (collisions are
			// disambiguated with @file), so first-wins never fires; it is
			// here so a hand-written or corrupt file cannot corrupt the
			// numbering.
			if _, dup := index[i][u.Key]; !dup {
				index[i][u.Key] = total + j
			}
		}
		total += len(s.Units)
	}

	uf := newUnionFind(total)

	// arrived[n] is the class of the change that produced node n, on the new
	// side of the delta that ends at its step. fate[n] is the class of the
	// change that consumed it, on the old side of the delta that starts at
	// its step. Both are plain lookups.
	arrived := make([]Class, total)
	fate := make([]Class, total)

	for i, d := range deltas {
		for _, c := range d.Changes {
			for _, m := range c.Old {
				if n, ok := index[i][m.Key]; ok {
					fate[n] = c.Class
				}
			}
			for _, m := range c.New {
				if n, ok := index[i+1][m.Key]; ok {
					arrived[n] = c.Class
				}
			}
			if !oneToOne(c.Class) || len(c.Old) != 1 || len(c.New) != 1 {
				continue
			}
			a, okA := index[i][c.Old[0].Key]
			b, okB := index[i+1][c.New[0].Key]
			if okA && okB {
				uf.union(a, b)
			}
		}
	}

	// Collect components in node order, so a component is discovered at its
	// earliest step and, within a step, at its snapshot position.
	tracks := make([]Track, 0)
	slot := make(map[int]int, total)
	for i, s := range snaps {
		for j, u := range s.Units {
			n := offset[i] + j
			root := uf.find(n)
			at, ok := slot[root]
			if !ok {
				at = len(tracks)
				slot[root] = at
				tracks = append(tracks, Track{ID: at})
			}
			tracks[at].Points = append(tracks[at].Points, TrackPoint{
				Step:  i,
				Key:   u.Key,
				Class: arrived[n],
			})
		}
	}

	last := len(snaps) - 1
	for i := range tracks {
		pts := tracks[i].Points
		sort.SliceStable(pts, func(a, b int) bool { return pts[a].Step < pts[b].Step })
		end := pts[len(pts)-1]
		if end.Step < last {
			tracks[i].Fate = fate[index[end.Step][end.Key]]
		}
	}
	return tracks
}

// unionFind is the standard disjoint set with path halving and union by size.
// Ties in union by size go to the lower index, so the representative of a
// component never depends on the order two equal-sized halves were joined in.
type unionFind struct {
	parent []int
	size   []int
}

func newUnionFind(n int) *unionFind {
	u := &unionFind{parent: make([]int, n), size: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i
		u.size[i] = 1
	}
	return u
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return
	}
	if u.size[ra] < u.size[rb] || (u.size[ra] == u.size[rb] && rb < ra) {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	u.size[ra] += u.size[rb]
}
