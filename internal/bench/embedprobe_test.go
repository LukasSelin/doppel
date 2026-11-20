package bench

import (
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// TestEmbedProbe asks one question before anything is built: would a
// function embedding doppel computes itself separate the pairs a reviewer
// called a merge from the ones they called a false positive — mirror
// operations and same-shape-different-type pairs above all — better than
// what the tool already reads?
//
// Three model-free readings of a body with its names kept are scored beside
// code-shape and the existing lenses, on a labelled corpus:
//
//	chain   IDF-weighted cosine over token 1/2/3-grams of the body as
//	        written, identifiers split into words. A "chain of tokens"
//	        vector; hashing it into a dense array would approximate this.
//	dist    a corpus-trained dense embedding by random indexing: every
//	        token's vector is the sum of its neighbours' index vectors over
//	        the corpus, a function's is the IDF-weighted sum of its tokens',
//	        mean-centred. Count-based stand-in for what a neural model
//	        learns: words are alike when their contexts are.
//	vocab   the existing vocab lens (canonical tree, names and values kept).
//
// It also prints how close dist puts known opposite words (get/set,
// encode/decode), which is the mechanism behind the answer. Asserts nothing.
//
// Measured on doppel's own labels and on cobra, and not adopted. On doppel
// every names-kept reading orders merges above false positives better than
// code-shape does, mirrors included. On cobra chain and dist do worse than
// code-shape, because a labelled false positive (Gt beside Eq) is the same
// token chain with one operator changed, and dist reads it above every
// labelled merge. CLAUDE.md's measurement list has the numbers.
//
//	DOPPEL_BENCH_EMBED="<corpus>=<labels>" go test ./internal/bench/ -v -run TestEmbedProbe
func TestEmbedProbe(t *testing.T) {
	spec := os.Getenv("DOPPEL_BENCH_EMBED")
	if spec == "" {
		t.Skip("set DOPPEL_BENCH_EMBED=<corpus>=<labels> to probe native embeddings")
	}
	for _, s := range parseLabeledSpecs(t, spec) {
		kr := loadKindRun(t, s)
		units := kr.run.Units
		fns, _ := lensFuncs(units)

		streams := make([][]string, len(units))
		for i, fn := range fns {
			if fn != nil {
				streams[i] = tokenStream(fn.Body)
			}
		}
		chain := newChainVectors(streams)
		dist := newDistVectors(streams)

		type sig struct{ key, shape, skel, vocab, verb, chain, dist float64 }
		names := []string{"key", "shape", "skel", "vocab", "verb", "chain", "dist"}
		get := func(s sig) []float64 { return []float64{s.key, s.shape, s.skel, s.vocab, s.verb, s.chain, s.dist} }
		read := func(p analyzer.SimilarPair) sig {
			ks := kr.signals(p)
			return sig{
				key:   RankKey(kr.run, p),
				shape: p.Score,
				skel:  ks.lens[fingerprint.LensSkeleton].Jaccard,
				vocab: ks.lens[fingerprint.LensVocab].Jaccard,
				verb:  ks.lens[fingerprint.LensVerbatim].Jaccard,
				chain: chain.cos(p.AIdx, p.BIdx),
				dist:  dist.cos(p.AIdx, p.BIdx),
			}
		}

		labelled := kr.labelled()
		t.Logf("[%s] %d units, %d compared pairs, %d of %d labels resolved",
			kr.name, len(units), len(kr.run.Pairs), len(labelled), len(s.lf.Labels))

		buckets := map[string][]sig{}
		for _, lp := range labelled {
			b := labelBucket(lp.label)
			buckets[b] = append(buckets[b], read(lp.pair))
			if lp.label.Class == "false_positive" {
				buckets["false_positive (all)"] = append(buckets["false_positive (all)"], read(lp.pair))
			}
		}
		var keys []string
		for k := range buckets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Logf("  mean per label class            n  %s", strings.Join(padAll(names), ""))
		for _, k := range keys {
			row := fmt.Sprintf("  %-30s %3d", k, len(buckets[k]))
			for c := range names {
				sum := 0.0
				for _, s := range buckets[k] {
					sum += get(s)[c]
				}
				row += fmt.Sprintf("  %5.2f", sum/float64(len(buckets[k])))
			}
			t.Log(row)
		}

		auc := func(pos, neg []sig, c int) float64 {
			if len(pos) == 0 || len(neg) == 0 {
				return math.NaN()
			}
			wins := 0.0
			for _, p := range pos {
				for _, n := range neg {
					switch a, b := get(p)[c], get(n)[c]; {
					case a > b:
						wins++
					case a == b:
						wins += 0.5
					}
				}
			}
			return wins / float64(len(pos)*len(neg))
		}
		fp := buckets["false_positive (all)"]
		act := append(append([]sig(nil), buckets["merge"]...), buckets["refactor"]...)
		t.Logf("  AUC (1.00 = every true pair above every false positive)")
		for _, cmp := range []struct {
			name     string
			pos, neg []sig
		}{
			{"merge vs false positive", buckets["merge"], fp},
			{"merge+refactor vs false positive", act, fp},
			{"merge vs mirror", buckets["merge"], buckets["fp/mirror"]},
			{"merge vs skeleton", buckets["merge"], buckets["fp/skeleton"]},
			{"merge vs already-factored", buckets["merge"], buckets["fp/already-factored"]},
			{"refactor vs false positive", buckets["refactor"], fp},
		} {
			row := fmt.Sprintf("  %-30s    ", cmp.name)
			for c := range names {
				row += fmt.Sprintf("  %5.2f", auc(cmp.pos, cmp.neg, c))
			}
			t.Logf("%s   (%d vs %d)", row, len(cmp.pos), len(cmp.neg))
		}

		// The same reading by the kind production assigns, over every compared
		// pair at code-shape >= ForkShapeFloor: a larger sample than the
		// labels, and the mirror row is the complaint itself.
		byKind := map[string][]sig{}
		for _, p := range kr.run.Pairs {
			if p.Score < analyzer.ForkShapeFloor {
				continue
			}
			k := kr.signals(p).prodKind
			if k == "" {
				k = "(no kind)"
			}
			byKind[k] = append(byKind[k], read(p))
		}
		keys = keys[:0]
		for k := range byKind {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Logf("  mean per production kind, code-shape >= %.2f", analyzer.ForkShapeFloor)
		for _, k := range keys {
			row := fmt.Sprintf("  %-30s %3d", k, len(byKind[k]))
			for c := range names {
				sum := 0.0
				for _, s := range byKind[k] {
					sum += get(s)[c]
				}
				row += fmt.Sprintf("  %5.2f", sum/float64(len(byKind[k])))
			}
			t.Log(row)
		}

		// Every labelled pair, so a mean cannot hide which pairs carry it.
		sort.Slice(labelled, func(i, j int) bool {
			bi, bj := labelBucket(labelled[i].label), labelBucket(labelled[j].label)
			if bi != bj {
				return bi < bj
			}
			return read(labelled[i].pair).vocab > read(labelled[j].pair).vocab
		})
		t.Logf("  labelled pairs                  %s", strings.Join(padAll(names), ""))
		for _, lp := range labelled {
			s := read(lp.pair)
			row := fmt.Sprintf("  %-18s", labelBucket(lp.label))
			for _, v := range get(s) {
				row += fmt.Sprintf("  %5.2f", v)
			}
			t.Logf("%s  %s <-> %s", row, lp.label.A, lp.label.B)
		}

		// The mechanism: where does the distributional embedding put words
		// that mean opposite things?
		dist.logOpposites(t)
	}
}

func padAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = fmt.Sprintf("  %5s", n)
	}
	return out
}

var probeWordRE = regexp.MustCompile(`[A-Z]?[a-z]+|[A-Z]+(?:[a-z]+)?|\d+`)

// tokenStream linearizes a body in visit order with its names kept:
// identifiers and selected names split into lower-case words, literals by
// kind and a bounded prefix of their value, operators by their token, every
// other named node by its kind.
func tokenStream(root *syntax.Node) []string {
	var out []string
	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n == nil {
			return true
		}
		switch n.Kind {
		case syntax.KindIdent, syntax.KindSelector:
			for _, w := range probeWordRE.FindAllString(n.Label, -1) {
				out = append(out, "w:"+strings.ToLower(w))
			}
		case syntax.KindLit:
			v := n.Text
			if len(v) > 12 {
				v = v[:12]
			}
			out = append(out, "lit:"+n.Label+":"+v)
		case syntax.KindOther:
		default:
			out = append(out, fmt.Sprintf("k%d:%s", n.Kind, n.Label))
		}
		return true
	})
	return out
}

func probeHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// chainVectors is the exact form of a hashed token-chain embedding: each
// function a sparse vector over its token 1/2/3-grams, weighted count x
// ln(N/df).
type chainVectors struct {
	vecs []map[uint64]float64
	norm []float64
}

func newChainVectors(streams [][]string) *chainVectors {
	grams := make([]map[uint64]float64, len(streams))
	df := map[uint64]int{}
	for i, s := range streams {
		g := map[uint64]float64{}
		for j := range s {
			g[probeHash(s[j])]++
			if j+1 < len(s) {
				g[probeHash(s[j]+"\x00"+s[j+1])]++
			}
			if j+2 < len(s) {
				g[probeHash(s[j]+"\x00"+s[j+1]+"\x00"+s[j+2])]++
			}
		}
		grams[i] = g
		for k := range g {
			df[k]++
		}
	}
	n := float64(len(streams))
	cv := &chainVectors{vecs: grams, norm: make([]float64, len(streams))}
	for i, g := range grams {
		sum := 0.0
		for k, c := range g {
			w := c * math.Log(n/float64(df[k]))
			g[k] = w
			sum += w * w
		}
		cv.norm[i] = math.Sqrt(sum)
	}
	return cv
}

func (cv *chainVectors) cos(a, b int) float64 {
	if cv.norm[a] == 0 || cv.norm[b] == 0 {
		return 0
	}
	va, vb := cv.vecs[a], cv.vecs[b]
	if len(vb) < len(va) {
		va, vb = vb, va
	}
	dot := 0.0
	for k, w := range va {
		dot += w * vb[k]
	}
	return dot / (cv.norm[a] * cv.norm[b])
}

// distVectors is a corpus-trained dense embedding by random indexing.
type distVectors struct {
	dim   int
	words map[string][]float64 // unit-length context vector per token
	count map[string]int
	fn    [][]float64 // mean-centred, unit-length function vectors
}

const (
	distDim     = 256
	distNonzero = 8
	distWindow  = 3
)

// indexVector is a token's fixed sparse random signature: distNonzero signed
// positions derived from its hash, so the embedding is deterministic.
func indexVector(tok string) (pos [distNonzero]int, sign [distNonzero]float64) {
	x := probeHash(tok)
	for i := range pos {
		x ^= x >> 30
		x *= 0xbf58476d1ce4e5b9
		x ^= x >> 27
		x *= 0x94d049bb133111eb
		x ^= x >> 31
		pos[i] = int(x % distDim)
		sign[i] = 1
		if x&(1<<40) != 0 {
			sign[i] = -1
		}
	}
	return
}

func newDistVectors(streams [][]string) *distVectors {
	df := map[string]int{}
	count := map[string]int{}
	for _, s := range streams {
		seen := map[string]bool{}
		for _, tok := range s {
			count[tok]++
			if !seen[tok] {
				seen[tok] = true
				df[tok]++
			}
		}
	}
	n := float64(len(streams))
	idf := func(tok string) float64 { return math.Log(n / float64(df[tok])) }

	words := map[string][]float64{}
	for _, s := range streams {
		for i, tok := range s {
			v := words[tok]
			if v == nil {
				v = make([]float64, distDim)
				words[tok] = v
			}
			for j := max(0, i-distWindow); j <= min(len(s)-1, i+distWindow); j++ {
				if j == i {
					continue
				}
				w := idf(s[j])
				pos, sign := indexVector(s[j])
				for k := range pos {
					v[pos[k]] += sign[k] * w
				}
			}
		}
	}
	for _, v := range words {
		unit(v)
	}

	fn := make([][]float64, len(streams))
	mean := make([]float64, distDim)
	for i, s := range streams {
		v := make([]float64, distDim)
		for _, tok := range s {
			w := idf(tok)
			for k, x := range words[tok] {
				v[k] += w * x
			}
		}
		unit(v)
		fn[i] = v
		for k, x := range v {
			mean[k] += x / n
		}
	}
	for _, v := range fn {
		for k := range v {
			v[k] -= mean[k]
		}
		unit(v)
	}
	return &distVectors{dim: distDim, words: words, count: count, fn: fn}
}

func unit(v []float64) {
	sum := 0.0
	for _, x := range v {
		sum += x * x
	}
	if sum == 0 {
		return
	}
	s := math.Sqrt(sum)
	for i := range v {
		v[i] /= s
	}
}

func dot(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func (dv *distVectors) cos(a, b int) float64 { return dot(dv.fn[a], dv.fn[b]) }

// logOpposites prints, for every opposite word pair the corpus uses at least
// ten times a side, the cosine of the two words' vectors and the share of
// all other frequent words that sit further from the first word than its
// opposite does. A share near 1.00 means the embedding reads the opposite as
// one of the word's nearest neighbours.
func (dv *distVectors) logOpposites(t *testing.T) {
	const minCount = 10
	var frequent []string
	for w, c := range dv.count {
		if c >= minCount && strings.HasPrefix(w, "w:") {
			frequent = append(frequent, w)
		}
	}
	sort.Strings(frequent)
	t.Logf("  opposite words in the corpus-trained embedding (%d words used >= %d times)", len(frequent), minCount)
	var shares []float64
	seen := map[string]bool{}
	for _, o := range opposites {
		a, b := "w:"+o[0], "w:"+o[1]
		if dv.count[a] < minCount || dv.count[b] < minCount || seen[a+b] {
			continue
		}
		seen[a+b] = true
		c := dot(dv.words[a], dv.words[b])
		further := 0
		for _, w := range frequent {
			if w != a && w != b && dot(dv.words[a], dv.words[w]) < c {
				further++
			}
		}
		share := float64(further) / float64(len(frequent)-2)
		shares = append(shares, share)
		t.Logf("    %-10s %-10s cos %.2f   closer than %3.0f%% of other words", o[0], o[1], c, 100*share)
	}
	if len(shares) > 0 {
		sort.Float64s(shares)
		t.Logf("    median: an opposite is closer than %.0f%% of other words", 100*shares[len(shares)/2])
	}
}
