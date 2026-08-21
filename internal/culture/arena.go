package culture

import (
	"math"
	"sort"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/parser"
)

// The concept arena: instead of treating a function's tags as independent
// booleans, every function is an arena where candidate concepts compete for
// its evidence under deterministic replicator dynamics. The equilibrium is a
// concept profile — masses summing to 1 — plus an ecosystem state. The
// interaction matrix and all evidence are corpus-derived: the PMI ecology is
// the arena's physics, so a concept can only invade a function through a
// reported association, and the same corpus always reaches the same
// equilibrium.

// Ecosystem states, in classification-precedence order (weak first).
const (
	StateDominance = "dominance" // one concept holds the niche
	StateCoalition = "coalition" // compatible concepts share it
	StateConflict  = "conflict"  // incompatible concepts both survive
	StateWeak      = "weak"      // candidates existed, evidence did not
)

// Dynamics constants — deliberately not Options: changing them is a
// semantics change, not tuning.
const (
	arenaEta            = 0.25 // replicator learning rate inside exp(eta·(F−maxF))
	arenaMaxRounds      = 64   // hard cap; Converged=false when hit
	arenaConvergeEps    = 1e-9 // stop when max |Δx| < eps, measured post-clamp
	arenaExtinctionMass = 1e-6 // masses below this clamp to exactly 0 and never revive
)

// ConceptMass is one concept's equilibrium mass in an arena.
type ConceptMass struct {
	Tag  string
	Mass float64
}

// ArenaProfile is one function's equilibrium: who survived, who went
// extinct, and what kind of ecosystem the evidence supports.
type ArenaProfile struct {
	State         string
	Survivors     []ConceptMass // mass >= SurvivorMass, sorted (Mass desc, Tag asc); never empty
	Extinct       []ConceptMass // the rest, same sort; nil when everyone survived
	TotalEvidence float64       // Σ Evidence(f,c) over candidates, nats, pre-dynamics
	Rounds        int
	Converged     bool
}

// ArenaProfile returns unit idx's equilibrium concept profile. ok is false
// when the unit had an empty candidate set — silence, not a state.
func (m *Model) ArenaProfile(idx int) (ArenaProfile, bool) {
	p, ok := m.arenas[idx]
	return p, ok
}

// tagPMI is one reported positive association endpoint.
type tagPMI struct {
	tag string
	pmi float64
}

// buildArenas runs one replicator competition per unit with candidates.
func buildArenas(m *Model, units []parser.CodeUnit, docs []concepter.ConceptDoc,
	uf *unitFeatures, opt Options) {

	n := len(units)
	if n == 0 {
		return
	}

	// Prebuilt lookups from the already-sorted association list. "Positive"
	// is the ecology's own predicate (PMI >= ln 2); the arena inherits the
	// ecology's cutoffs and never re-derives them.
	tagDF := make(map[string]int)
	for i := range units {
		for _, t := range uf.sortedPatterns[i] {
			tagDF[t]++
		}
	}
	posCallTag := make(map[string][]tagPMI)
	posRoleTag := make(map[string][]tagPMI)
	inter := make(map[[2]string]float64)
	for _, a := range m.associations {
		switch a.Kind {
		case TagCall:
			if a.PMI >= math.Ln2 {
				posCallTag[a.B] = append(posCallTag[a.B], tagPMI{tag: a.A, pmi: a.PMI})
			}
		case TagRole:
			if a.PMI >= math.Ln2 {
				posRoleTag[a.B] = append(posRoleTag[a.B], tagPMI{tag: a.A, pmi: a.PMI})
			}
		case TagTag:
			pmi := a.PMI
			if math.IsInf(pmi, -1) {
				// "Never co-occurs" becomes the largest-magnitude finite
				// repulsion the corpus can express, so infinities cannot
				// poison fitness arithmetic while the ordering
				// never < rarely < uncorrelated survives.
				pmi = -math.Log(float64(n))
			}
			inter[[2]string{a.A, a.B}] = pmi
		}
	}
	// Multi-valued entries sorted (Tag asc) so candidate enumeration and
	// evidence sums never depend on association-list grouping.
	for _, lst := range posCallTag {
		sort.Slice(lst, func(i, j int) bool { return lst[i].tag < lst[j].tag })
	}
	for _, lst := range posRoleTag {
		sort.Slice(lst, func(i, j int) bool { return lst[i].tag < lst[j].tag })
	}

	interactionOf := func(c, d string) float64 {
		if c == d {
			return 0 // the replicator's own mass term carries self-reinforcement
		}
		if c > d {
			c, d = d, c
		}
		return inter[[2]string{c, d}]
	}

	for i := range units {
		// Candidate set: own tags plus concepts positively associated with
		// the unit's call tokens or role. Empty set = silence.
		cand := make(map[string]bool)
		for _, t := range uf.sortedPatterns[i] {
			cand[t] = true
		}
		for _, tok := range uf.tokens[i] {
			for _, tp := range posCallTag[tok] {
				cand[tp.tag] = true
			}
		}
		for _, tp := range posRoleTag[docs[i].Role] {
			cand[tp.tag] = true
		}
		candidates := sortedStrings(cand)
		if len(candidates) == 0 {
			continue
		}

		// Evidence per candidate, fixed component order: direct tag IC,
		// then call support, then role support.
		evidence := make([]float64, len(candidates))
		hasTag := make(map[string]bool, len(uf.sortedPatterns[i]))
		for _, t := range uf.sortedPatterns[i] {
			hasTag[t] = true
		}
		for ci, c := range candidates {
			var e float64
			if hasTag[c] {
				e += math.Log(float64(n) / float64(tagDF[c]))
			}
			for _, tok := range uf.tokens[i] {
				for _, tp := range posCallTag[tok] {
					if tp.tag == c {
						e += tp.pmi
					}
				}
			}
			for _, tp := range posRoleTag[docs[i].Role] {
				if tp.tag == c {
					e += tp.pmi
				}
			}
			evidence[ci] = e
		}
		var totalEvidence float64
		for _, e := range evidence {
			totalEvidence += e
		}

		profile := runReplicator(candidates, evidence, interactionOf)
		profile.TotalEvidence = totalEvidence
		profile.classify(interactionOf, opt)

		m.arenas[i] = profile
		m.stats.ArenaProfiled++
		switch profile.State {
		case StateDominance:
			m.stats.ArenaDominance++
		case StateCoalition:
			m.stats.ArenaCoalition++
		case StateConflict:
			m.stats.ArenaConflict++
		case StateWeak:
			m.stats.ArenaWeak++
		}
	}
}

// runReplicator iterates the discrete replicator update to (bounded)
// equilibrium. Everything runs in the fixed ascending-candidate order; the
// softmax shift is a max, so its value is scan-order independent, and after
// the shift every exponent is <= 0, so overflow is impossible.
func runReplicator(candidates []string, evidence []float64,
	interactionOf func(c, d string) float64) ArenaProfile {

	k := len(candidates)
	x := make([]float64, k)
	for i := range x {
		x[i] = 1 / float64(k)
	}

	fitness := make([]float64, k)
	next := make([]float64, k)
	rounds := 0
	converged := false
	for rounds < arenaMaxRounds {
		rounds++
		for ci := range candidates {
			f := evidence[ci]
			for di := range candidates {
				f += interactionOf(candidates[ci], candidates[di]) * x[di]
			}
			fitness[ci] = f
		}
		shift := fitness[0]
		for _, f := range fitness[1:] {
			if f > shift {
				shift = f
			}
		}
		var z float64
		for ci := range candidates {
			next[ci] = x[ci] * math.Exp(arenaEta*(fitness[ci]-shift))
			z += next[ci]
		}
		for ci := range candidates {
			next[ci] /= z
		}
		// Extinction clamp then renormalize: a clamped concept's
		// multiplicative update is zero forever, so clamping only
		// determinizes what the dynamics already do.
		var z2 float64
		for ci := range candidates {
			if next[ci] < arenaExtinctionMass {
				next[ci] = 0
			}
			z2 += next[ci]
		}
		var delta float64
		for ci := range candidates {
			next[ci] /= z2
			if d := math.Abs(next[ci] - x[ci]); d > delta {
				delta = d
			}
		}
		x, next = next, x
		if delta < arenaConvergeEps {
			converged = true
			break
		}
	}

	masses := make([]ConceptMass, k)
	for ci := range candidates {
		masses[ci] = ConceptMass{Tag: candidates[ci], Mass: x[ci]}
	}
	return ArenaProfile{Rounds: rounds, Converged: converged, Survivors: masses}
}

// classify splits Survivors/Extinct on the SurvivorMass floor and assigns
// the ecosystem state in the pinned precedence order: weak, then conflict,
// then dominance, then coalition. Weak first because an equilibrium over
// noise is still noise; conflict before dominance because the actionable
// smell must not be masked by a large top mass — real dominance means the
// winner drove its rivals extinct.
func (p *ArenaProfile) classify(interactionOf func(c, d string) float64, opt Options) {
	all := p.Survivors
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Mass != all[j].Mass {
			return all[i].Mass > all[j].Mass
		}
		return all[i].Tag < all[j].Tag
	})
	var survivors, extinct []ConceptMass
	for _, cm := range all {
		if cm.Mass >= opt.SurvivorMass {
			survivors = append(survivors, cm)
		} else {
			extinct = append(extinct, cm)
		}
	}
	p.Survivors = survivors
	p.Extinct = extinct

	switch {
	case p.TotalEvidence < opt.MinArenaEvidence:
		p.State = StateWeak
	case len(survivors) >= 2 && hasNegativePair(survivors, interactionOf):
		p.State = StateConflict
	case len(survivors) == 1 || survivors[0].Mass >= opt.DominanceMass:
		p.State = StateDominance
	default:
		p.State = StateCoalition
	}
}

func hasNegativePair(survivors []ConceptMass, interactionOf func(c, d string) float64) bool {
	for i := range survivors {
		for j := i + 1; j < len(survivors); j++ {
			if interactionOf(survivors[i].Tag, survivors[j].Tag) < 0 {
				return true
			}
		}
	}
	return false
}
