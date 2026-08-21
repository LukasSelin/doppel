package culture

import (
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Habitat channels: the culture prototype channels minus `package` — a
// habitat IS a package, so a member's package feature is constant within it
// and carries zero information. `cotags` becomes `tags` (there is no anchor
// tag to exclude). The 40/20/15/15 remainder renormalizes over 90 to
// integers summing to exactly 100, pinned by test.
var habitatChannelNames = [4]string{"calls", "flow", "tags", "role"}
var habitatChannelWeightPct = [4]int{44, 22, 17, 17}

// ChannelSurprise is one channel's contribution to a member's habitat strain.
type ChannelSurprise struct {
	Name     string
	Surprise float64 // mean feature surprisal under the habitat, nats
}

// habitatModel holds one package's practice distribution results.
type habitatModel struct {
	pkg         string
	members     []int // ascending unit indices
	strain      map[int]float64
	chStrain    map[int][]float64 // aligned with habitatChannelNames
	temperature float64           // median member strain
	norm        float64           // mean member fit
}

// buildHabitats models each package with enough members: per-member strain
// (weighted mean surprisal of its features under the habitat's own
// distribution), the habitat temperature (median strain), and per-member fit.
//
// Smoothing is the load-bearing identity: leave-one-out counts plus a single
// pseudo-count collapse to the plain presence fraction —
//
//	P_i(x|h) = (cnt_-i(x) + 1) / ((m-1) + 1) = cnt(x)/m
//
// so a member never certifies its own normality beyond the pseudo-count, no
// surprisal is ever infinite (P >= 1/m), and everything reduces to integer
// counts over one denominator — order-independent by construction.
func buildHabitats(m *Model, units []parser.CodeUnit, docs []concepter.ConceptDoc,
	uf *unitFeatures, opt Options) {

	byPkg := make(map[string][]int)
	for i := range units {
		// Units without a package clause have no meaningful habitat.
		if units[i].Package == "" {
			continue
		}
		byPkg[units[i].Package] = append(byPkg[units[i].Package], i) // ascending
	}

	features := func(i, ch int) []string {
		switch habitatChannelNames[ch] {
		case "calls":
			return uf.tokens[i]
		case "flow":
			return uf.flowFeats[i]
		case "tags":
			return uf.sortedPatterns[i]
		case "role":
			return []string{docs[i].Role}
		}
		return nil
	}

	for _, pkg := range sortedCountKeysInt(byPkg) {
		members := byPkg[pkg]
		mm := len(members)
		if mm < opt.MinHabitatMembers {
			continue
		}
		hm := &habitatModel{
			pkg:      pkg,
			members:  members,
			strain:   make(map[int]float64, mm),
			chStrain: make(map[int][]float64, mm),
		}

		for ch := range habitatChannelNames {
			cnt := make(map[string]int)
			emptyCnt := 0
			for _, i := range members {
				f := features(i, ch)
				if len(f) == 0 {
					emptyCnt++
				}
				for _, x := range f {
					cnt[x]++
				}
			}
			// Mean surprisal per feature, so feature-rich functions are not
			// penalized for richness. An empty feature set is scored as the
			// empty-set event — doing nothing can be the norm — so every
			// channel is always defined and no weight renormalization exists.
			for _, i := range members {
				f := features(i, ch)
				var e float64
				if len(f) == 0 {
					e = -math.Log(float64(emptyCnt) / float64(mm))
				} else {
					for _, x := range f { // sorted: fixed float order
						e += -math.Log(float64(cnt[x]) / float64(mm))
					}
					e /= float64(len(f))
				}
				hm.chStrain[i] = append(hm.chStrain[i], e)
			}
		}

		strains := make([]float64, 0, mm)
		for _, i := range members {
			var total float64
			for ch := range habitatChannelNames {
				total += float64(habitatChannelWeightPct[ch]) * hm.chStrain[i][ch]
			}
			total /= 100
			hm.strain[i] = total
			strains = append(strains, total)
		}
		sort.Float64s(strains)
		if mm%2 == 1 {
			hm.temperature = strains[mm/2]
		} else {
			hm.temperature = (strains[mm/2-1] + strains[mm/2]) / 2
		}

		var fitSum float64
		for _, i := range members {
			fitSum += fitOf(hm.strain[i], hm.temperature)
			if misfitOf(hm.strain[i], hm.temperature, opt.MisfitFactor) {
				m.stats.HabitatMisfits++
			}
			m.habitatByUnit[i] = hm
		}
		hm.norm = fitSum / float64(mm)

		m.habitats[pkg] = hm
		m.stats.HabitatsModeled++
	}

	// Superlatives for the stderr summary: strict comparisons over
	// name-sorted habitats, so ties resolve to the lexicographically
	// smaller name.
	first := true
	for _, pkg := range sortedHabitatKeys(m.habitats) {
		norm := m.habitats[pkg].norm
		if first {
			m.stats.MostUniformHabitat, m.stats.MostUniformNorm = pkg, norm
			m.stats.MostDiverseHabitat, m.stats.MostDiverseNorm = pkg, norm
			first = false
			continue
		}
		if norm > m.stats.MostUniformNorm {
			m.stats.MostUniformHabitat, m.stats.MostUniformNorm = pkg, norm
		}
		if norm < m.stats.MostDiverseNorm {
			m.stats.MostDiverseHabitat, m.stats.MostDiverseNorm = pkg, norm
		}
	}
}

// fitOf is the excess-energy Boltzmann factor: strain at or below the
// habitat norm is a perfect fit by definition (the median member reads
// exactly 1.0), and only excess strain decays, scaled by the habitat's own
// tolerance. A frozen habitat (temperature 0) makes any deviation maximally
// surprising; the branch order keeps all-identical habitats at 1.0.
func fitOf(strain, temperature float64) float64 {
	if strain <= temperature {
		return 1.0
	}
	if temperature == 0 {
		return 0.0
	}
	return math.Exp(-(strain - temperature) / temperature)
}

// misfitOf mirrors Atypical's median-relative rule (at factor 2.0 it is
// equivalent to fit < e^-1). The temperature-0 disjunct keeps frozen-habitat
// deviants flaggable, consistent with their 0.0 fit.
func misfitOf(strain, temperature, factor float64) bool {
	if temperature > 0 {
		return strain > factor*temperature
	}
	return strain > 0
}

func sortedHabitatKeys(m map[string]*habitatModel) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// HabitatStrain reports unit idx's local strain — the weighted mean
// surprisal of its features under its own package's distribution, in nats.
// ok is false when the package has no habitat model.
func (m *Model) HabitatStrain(idx int) (float64, bool) {
	hm := m.habitatByUnit[idx]
	if hm == nil {
		return 0, false
	}
	return hm.strain[idx], true
}

// HabitatTemperature reports a package's tolerance: the median member
// strain. 0 for a perfectly uniform habitat.
func (m *Model) HabitatTemperature(pkg string) (float64, bool) {
	hm := m.habitats[pkg]
	if hm == nil {
		return 0, false
	}
	return hm.temperature, true
}

// HabitatFit reports how well unit idx fits where it lives, in [0,1].
func (m *Model) HabitatFit(idx int) (float64, bool) {
	hm := m.habitatByUnit[idx]
	if hm == nil {
		return 0, false
	}
	return fitOf(hm.strain[idx], hm.temperature), true
}

// HabitatNorm reports a package's uniformity in fit units: the mean member
// fit — near 1 for a cold, regular habitat; dragged down by outliers.
func (m *Model) HabitatNorm(pkg string) (float64, bool) {
	hm := m.habitats[pkg]
	if hm == nil {
		return 0, false
	}
	return hm.norm, true
}

// Misfit reports whether unit idx is notably out of place in its package.
func (m *Model) Misfit(idx int) bool {
	hm := m.habitatByUnit[idx]
	if hm == nil {
		return false
	}
	return misfitOf(hm.strain[idx], hm.temperature, m.opt.MisfitFactor)
}

// ChannelSurprise returns the per-channel strain behind HabitatStrain, in
// the fixed habitat channel order, or nil when the unit's package has no
// habitat model.
func (m *Model) ChannelSurprise(idx int) []ChannelSurprise {
	hm := m.habitatByUnit[idx]
	if hm == nil {
		return nil
	}
	vals := hm.chStrain[idx]
	out := make([]ChannelSurprise, len(habitatChannelNames))
	for ch := range habitatChannelNames {
		out[ch] = ChannelSurprise{Name: habitatChannelNames[ch], Surprise: vals[ch]}
	}
	return out
}
