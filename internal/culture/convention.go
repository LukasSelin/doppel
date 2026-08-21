package culture

import "math"

// ConventionStrength reports how strict the corpus's practice is for a
// concept, in [0,1]: 1 − normalized Shannon entropy of the prototype's
// feature-mass distributions, combined over the five prototype channels with
// the same integer weights the prototype uses. ok is false when the tag has
// no prototype. A strong convention (one dominant realization) reads near 1;
// diverse accepted practice reads low — introducing yet another shape where
// the convention is strong is more notable than where practice is loose.
func (m *Model) ConventionStrength(tag string) (float64, bool) {
	c := m.concepts[tag]
	if c == nil {
		return 0, false
	}
	return c.convention, true
}

// conventionOf computes the cached convention strength from a built
// prototype: Σ pct·S_ch / 100 in fixed channel order.
func conventionOf(p Prototype) float64 {
	var total float64
	for ch := range channelNames {
		total += float64(channelWeightPct[ch]) * channelStrength(p.Channels[ch].Features)
	}
	return total / 100
}

// channelStrength is 1 − the channel's normalized disorder, where disorder
// is the mass-weighted mean Bernoulli entropy of each feature's presence
// across members: h(P) = −P·lnP − (1−P)·ln(1−P), weights w_j = P_j/ΣP.
//
// Why Bernoulli-per-feature rather than entropy over the mass distribution:
// a concept where EVERY member does the same two things (say if+return, both
// at P=1.0) has uniform masses and would read maximal entropy — zero
// convention despite perfect unanimity. Presence entropy fixes that: a
// universal feature (P=1) contributes no disorder, a coin-flip feature
// (P=0.5) the most, and mass weighting keeps rare tail features from
// diluting a dominant form. This measures predictability of practice — a
// dispersion proxy, not realization clustering. An empty channel is
// unanimity of absence and scores 1.0 by convention, the same "doing nothing
// can be the norm" stance as typicality. Iteration follows the pinned
// (P desc, Name asc) Feature order.
func channelStrength(features []Feature) float64 {
	if len(features) == 0 {
		return 1.0
	}
	var sum float64
	for _, f := range features {
		sum += f.P
	}
	var disorder float64
	for _, f := range features {
		disorder += (f.P / sum) * bernoulliEntropy(f.P)
	}
	s := 1 - disorder/math.Ln2
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

func bernoulliEntropy(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}
	return -p*math.Log(p) - (1-p)*math.Log(1-p)
}
