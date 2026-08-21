package culture

import (
	"math"
	"testing"
)

// channelStrength unanimity endpoints: an empty channel and universal
// features (P = 1.0) carry no disorder — co-occurring universal features
// must NOT read as diversity, which is why disorder is Bernoulli-per-feature
// rather than entropy over the mass distribution.
func TestChannelStrengthUnanimity(t *testing.T) {
	if s := channelStrength(nil); s != 1.0 {
		t.Errorf("empty channel strength = %v, want 1.0", s)
	}
	universalPair := []Feature{{Name: "if", P: 1.0}, {Name: "return", P: 1.0}}
	if s := channelStrength(universalPair); s != 1.0 {
		t.Errorf("two universal features strength = %v, want exactly 1.0", s)
	}
}

// Coin-flip presence is maximal disorder; the skew pin is hand-computed:
// fractions 0.8/0.4 with weights {2/3, 1/3} give disorder
// (2/3)h(0.8) + (1/3)h(0.4) ≈ 0.5579, strength ≈ 0.195.
func TestChannelStrengthValues(t *testing.T) {
	coinFlips := []Feature{{Name: "a", P: 0.5}, {Name: "b", P: 0.5}, {Name: "c", P: 0.5}}
	if s := channelStrength(coinFlips); math.Abs(s) > 1e-12 {
		t.Errorf("coin-flip spread strength = %v, want exactly 0", s)
	}

	skewed := []Feature{{Name: "dominant", P: 0.8}, {Name: "rare", P: 0.4}}
	want := 1 - ((2.0/3.0)*bernoulliEntropy(0.8)+(1.0/3.0)*bernoulliEntropy(0.4))/math.Ln2
	if s := channelStrength(skewed); math.Abs(s-want) > 1e-12 {
		t.Errorf("skewed strength = %v, want %v (~0.195)", s, want)
	}
	if math.Abs(want-0.195) > 1e-3 {
		t.Fatalf("hand computation drifted: want ≈ 0.195, got %v", want)
	}

	// A half-used single feature is a weak convention, not unanimity.
	if s := channelStrength([]Feature{{Name: "x", P: 0.5}}); math.Abs(s) > 1e-12 {
		t.Errorf("single coin-flip feature strength = %v, want 0", s)
	}
}

// A concept whose members each carry a distinct call has zero calls-channel
// concentration; the composite follows the 40/20/15/15/10 weights over the
// per-channel strengths.
func TestConventionStrengthComposite(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())

	// error_wrapping: the five identical clones. Every channel is a single
	// feature (or unanimous), so the convention is exactly 1.0.
	got, ok := m.ConventionStrength("error_wrapping")
	if !ok {
		t.Fatal("no convention for error_wrapping")
	}
	if got != 1.0 {
		t.Errorf("clone-only concept convention = %v, want exactly 1.0", got)
	}

	// db_access: five clones + the alien. Hand-compose from the prototype:
	// calls {sql.Open 5/6} single feature -> 1.0; flow {if,return 5/6 each,
	// go,select 1/6 each}; cotags single; role two features; package two.
	proto, _ := m.Prototype("db_access")
	var want float64
	for ch := range channelNames {
		want += float64(channelWeightPct[ch]) * channelStrength(proto.Channels[ch].Features)
	}
	want /= 100
	got, _ = m.ConventionStrength("db_access")
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("db_access convention = %v, want %v", got, want)
	}
	if got >= 1.0 || got <= 0 {
		t.Errorf("mixed concept convention = %v, want strictly inside (0,1)", got)
	}
}

func TestConventionStrengthUnprototyped(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())
	if s, ok := m.ConventionStrength("retry"); ok || s != 0 {
		t.Errorf("unprototyped tag = (%v, %v), want (0, false)", s, ok)
	}
}

// Superlatives: strongest/loosest over prototyped tags, with the strict
// comparison resolving ties to the lexicographically smaller tag.
func TestConventionStatsSuperlatives(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())
	s := m.Stats()
	if s.StrongestConvention != "error_wrapping" || s.StrongestConventionStrength != 1.0 {
		t.Errorf("strongest = %s (%v), want error_wrapping (1.0)",
			s.StrongestConvention, s.StrongestConventionStrength)
	}
	if s.LoosestConvention != "db_access" || s.LoosestConventionStrength >= 1.0 {
		t.Errorf("loosest = %s (%v), want db_access (< 1.0)",
			s.LoosestConvention, s.LoosestConventionStrength)
	}
}
