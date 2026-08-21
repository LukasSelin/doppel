package culture

import (
	"sort"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/fingerprint"
	"github.com/lukse/doppel/internal/parser"
)

// The prototype channels, in fixed order, with integer percentage weights
// summing to exactly 100 (pinned by a test, mirroring the comparator's axiom
// discipline; integers so that a perfectly typical member scores exactly
// 1.0 with no float residue). Calls dominate because they are the richest
// signal of how a concept is realized; flow is how it is structured;
// co-tags, role, and package are context. Caller/callee-pattern channels are
// deliberately deferred — adding one is a local change to this table.
var channelNames = [5]string{"calls", "flow", "cotags", "role", "package"}
var channelWeightPct = [5]int{40, 20, 15, 15, 10}

// Feature is one entry of a prototype channel distribution.
type Feature struct {
	Name string
	P    float64 // fraction of the concept's members carrying the feature
}

// PrototypeChannel is one channel's feature distribution.
type PrototypeChannel struct {
	Name     string
	Features []Feature // sorted by (P desc, Name asc)
}

// Prototype is how this corpus normally realizes a concept: per-channel
// feature distributions over the concept's members.
type Prototype struct {
	Channels []PrototypeChannel // fixed channel order
}

// buildPrototypes fills m.concepts for every tag with enough members, and
// accumulates the model stats. All float arithmetic reduces to
// integer-counted numerators over fixed denominators, so results are
// independent of iteration order by construction; ordering rules exist for
// the slices we expose.
func buildPrototypes(m *Model, units []parser.CodeUnit, docs []concepter.ConceptDoc,
	uf *unitFeatures, opt Options) {

	// The cotags channel depends on the concept under consideration, so it
	// filters the shared sortedPatterns on the fly.
	features := func(i, ch int, tag string) []string {
		switch channelNames[ch] {
		case "calls":
			return uf.tokens[i]
		case "flow":
			return uf.flowFeats[i]
		case "cotags":
			var out []string
			for _, t := range uf.sortedPatterns[i] {
				if t != tag {
					out = append(out, t)
				}
			}
			return out
		case "role":
			return []string{docs[i].Role}
		case "package":
			return []string{units[i].Package}
		}
		return nil
	}

	membersByTag := make(map[string][]int)
	for i := range units {
		for _, t := range uf.sortedPatterns[i] {
			membersByTag[t] = append(membersByTag[t], i) // ascending: i ascends
		}
	}

	for _, tag := range sortedCountKeysInt(membersByTag) {
		members := membersByTag[tag]
		mm := len(members)
		if mm < opt.MinConceptMembers {
			continue
		}
		cm := &conceptModel{
			members: members,
			typ:     make(map[int]float64, mm),
			chTyp:   make(map[int][]float64, mm),
		}

		for ch := range channelNames {
			cnt := make(map[string]int)
			emptyCount := 0
			for _, i := range members {
				f := features(i, ch, tag)
				if len(f) == 0 {
					emptyCount++
				}
				for _, x := range f {
					cnt[x]++
				}
			}

			// Prototype distribution: P(x|c) = cnt(x)/m.
			pc := PrototypeChannel{Name: channelNames[ch]}
			for _, x := range sortedCountKeys(cnt) {
				pc.Features = append(pc.Features, Feature{Name: x, P: float64(cnt[x]) / float64(mm)})
			}
			sort.SliceStable(pc.Features, func(a, b int) bool {
				if pc.Features[a].P != pc.Features[b].P {
					return pc.Features[a].P > pc.Features[b].P
				}
				return pc.Features[a].Name < pc.Features[b].Name
			})
			cm.prototype.Channels = append(cm.prototype.Channels, pc)

			// Leave-one-out channel typicality. For a member with feature set
			// F this is the mean over the other members g of |F∩F_g|/|F| —
			// "on average, what fraction of what this function does is also
			// done by another member" — computed as an integer numerator:
			// Σ_{x∈F}(cnt(x)−1) over |F|·(m−1). A member with an empty F is
			// scored by how many other members also do nothing here, so every
			// channel is always defined and no weight renormalization exists.
			for _, i := range members {
				f := features(i, ch, tag)
				var t float64
				if len(f) == 0 {
					t = float64(emptyCount-1) / float64(mm-1)
				} else {
					shared := 0
					for _, x := range f {
						shared += cnt[x] - 1
					}
					t = float64(shared) / float64(len(f)*(mm-1))
				}
				cm.chTyp[i] = append(cm.chTyp[i], t)
			}
		}

		// Weighted composite, summed in fixed channel order; one division at
		// the end so identical members land on exactly 1.0.
		typs := make([]float64, 0, mm)
		for _, i := range members {
			var total float64
			for ch := range channelNames {
				total += float64(channelWeightPct[ch]) * cm.chTyp[i][ch]
			}
			total /= 100
			cm.typ[i] = total
			typs = append(typs, total)
		}
		sort.Float64s(typs)
		if mm%2 == 1 {
			cm.median = typs[mm/2]
		} else {
			cm.median = (typs[mm/2-1] + typs[mm/2]) / 2
		}

		cm.convention = conventionOf(cm.prototype)

		m.concepts[tag] = cm
		m.stats.ConceptsModeled++
		if cm.median > 0 {
			for _, i := range members {
				if cm.typ[i] < opt.AtypicalFactor*cm.median {
					m.stats.UnusualRealizations++
				}
			}
		}
	}

	// Convention superlatives for the stderr summary: strict comparisons over
	// name-sorted prototyped tags, so ties resolve to the lexicographically
	// smaller name.
	first := true
	for _, tag := range sortedConceptKeys(m.concepts) {
		strength := m.concepts[tag].convention
		if first {
			m.stats.StrongestConvention, m.stats.StrongestConventionStrength = tag, strength
			m.stats.LoosestConvention, m.stats.LoosestConventionStrength = tag, strength
			first = false
			continue
		}
		if strength > m.stats.StrongestConventionStrength {
			m.stats.StrongestConvention, m.stats.StrongestConventionStrength = tag, strength
		}
		if strength < m.stats.LoosestConventionStrength {
			m.stats.LoosestConvention, m.stats.LoosestConventionStrength = tag, strength
		}
	}
}

func sortedConceptKeys(m map[string]*conceptModel) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCountKeysInt(m map[string][]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// flowFeatures binarizes a unit's control-flow histogram into named labels.
func flowFeatures(u parser.CodeUnit) []string {
	var fl []string
	for k, n := range u.Fingerprint.Flow {
		if n > 0 {
			fl = append(fl, fingerprint.FlowLabels[k])
		}
	}
	sort.Strings(fl)
	return fl
}
