package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// labeledSpec is one corpus named by a DOPPEL_BENCH_* spec entry
// "<corpus dir>=<labels file>[=<family-min>]".
type labeledSpec struct {
	root   string
	lf     LabelsFile
	family family.Options
}

// parseLabeledSpecs reads a ';'-separated spec list. The family-min is
// optional and defaults to family.DefaultOptions' static 0.60; pass the value
// `doppel analyze` calibrates for the corpus to match what the tool groups.
func parseLabeledSpecs(t *testing.T, spec string) []labeledSpec {
	t.Helper()
	var out []labeledSpec
	for _, item := range strings.Split(spec, ";") {
		parts := strings.Split(item, "=")
		if len(parts) < 2 || len(parts) > 3 {
			t.Fatalf("bad entry %q: want <corpus>=<labels>[=<family-min>]", item)
		}
		data, err := os.ReadFile(parts[1])
		if err != nil {
			t.Fatal(err)
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatalf("%s: %v", parts[1], err)
		}
		fo := family.DefaultOptions()
		if len(parts) == 3 {
			if _, err := fmt.Sscanf(parts[2], "%g", &fo.Min); err != nil {
				t.Fatalf("bad family-min in %q: %v", item, err)
			}
		}
		out = append(out, labeledSpec{root: parts[0], lf: lf, family: fo})
	}
	return out
}

// kindSignals is every per-pair signal the kind lenses read: the five
// fingerprint lenses, and the call-graph, naming and build-boundary facts the
// lenses cannot see because they read one body at a time.
type kindSignals struct {
	shape                       float64 // production code-shape
	lens                        fingerprint.LensProfile
	class                       string // LensProfile.Class
	calleeJ, resolvedJ, callerJ float64
	callSim                     float64
	maxNodes, minNodes          int
	sharedResolved              int
	antonym                     bool // names differ only by an opposite pair (Encode/Decode, Get/Set...)
	sameReceiver                bool
	entry                       bool // both are main/init/run
	separateBinaries            bool // both in package main, different directories
	sameFile                    bool
	prodKind                    string // analyzer.ClassifyPairIn's kind, "" when none
}

// opposites are the word pairs whose presence, with everything else in the
// two names equal, marks a pair as one operation and its inverse or sibling.
var opposites = [][2]string{
	{"encode", "decode"}, {"marshal", "unmarshal"}, {"read", "write"}, {"get", "set"},
	{"get", "delete"}, {"set", "delete"}, {"get", "put"}, {"put", "delete"}, {"add", "remove"},
	{"min", "max"}, {"super", "sub"}, {"sub", "super"}, {"old", "new"}, {"source", "sink"},
	{"open", "close"}, {"lock", "unlock"}, {"push", "pop"}, {"load", "save"}, {"load", "store"},
	{"to", "from"}, {"split", "merge"}, {"splits", "merges"}, {"union", "intersection"},
	{"begin", "end"}, {"start", "stop"}, {"enable", "disable"}, {"acquire", "release"},
	{"lock", "release"}, {"assemble", "split"}, {"compress", "decompress"}, {"serialize", "deserialize"},
	{"inc", "dec"}, {"up", "down"}, {"left", "right"}, {"first", "last"}, {"prev", "next"},
	{"before", "after"}, {"in", "out"}, {"import", "export"}, {"send", "receive"}, {"send", "recv"},
}

var nameWord = regexp.MustCompile(`[A-Z]?[a-z]+|[A-Z]+(?:[a-z]+)?|\d+`)

func nameWords(u parser.CodeUnit) []string {
	n := parser.MethodName(u)
	var out []string
	for _, w := range nameWord.FindAllString(n, -1) {
		out = append(out, strings.ToLower(w))
	}
	return out
}

func isAntonymPair(a, b parser.CodeUnit) bool {
	wa, wb := nameWords(a), nameWords(b)
	if len(wa) != len(wb) {
		return false
	}
	diff := -1
	for i := range wa {
		if wa[i] != wb[i] {
			if diff >= 0 {
				return false
			}
			diff = i
		}
	}
	if diff < 0 {
		return false
	}
	for _, o := range opposites {
		if (wa[diff] == o[0] && wb[diff] == o[1]) || (wa[diff] == o[1] && wb[diff] == o[0]) {
			return true
		}
	}
	return false
}

func jaccardStrings(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	in := map[string]bool{}
	for _, x := range a {
		in[x] = true
	}
	union := len(in)
	shared := 0
	seen := map[string]bool{}
	for _, x := range b {
		if seen[x] {
			continue
		}
		seen[x] = true
		if in[x] {
			shared++
		} else {
			union++
		}
	}
	return float64(shared) / float64(union)
}

// kindRun is one labelled corpus with its lens bags built once.
type kindRun struct {
	name string
	spec labeledSpec
	run  *Run
	bags [][][]fingerprint.LabelCount
	idfs []*fingerprint.LabelIDF
	thin [][]fingerprint.LabelCount // analyzer.BuildThinVocab, what production reads
}

func loadKindRun(t *testing.T, s labeledSpec) *kindRun {
	t.Helper()
	units, err := Load(s.root, Population(s.lf.Population))
	if err != nil || len(units) == 0 {
		t.Fatalf("%s: load: %v (%d units)", s.root, err, len(units))
	}
	run := Analyze(units, retriever.DefaultOptions())
	run.Root = s.root
	fns, _ := lensFuncs(run.Units)
	lenses := fingerprint.Lenses()
	kr := &kindRun{name: filepath.Base(s.root), spec: s, run: run,
		bags: make([][][]fingerprint.LabelCount, len(lenses)), idfs: make([]*fingerprint.LabelIDF, len(lenses))}
	for li, l := range lenses {
		kr.bags[li] = make([][]fingerprint.LabelCount, len(fns))
		for i := range fns {
			kr.bags[li][i] = l.Bag(fns[i])
		}
		kr.idfs[li] = fingerprint.LabelWeights(kr.bags[li])
	}
	kr.thin = analyzer.BuildThinVocab(run.Units)
	return kr
}

func (kr *kindRun) signals(p analyzer.SimilarPair) kindSignals {
	a, b := kr.run.Units[p.AIdx], kr.run.Units[p.BIdx]
	da, db := kr.run.Docs[p.AIdx], kr.run.Docs[p.BIdx]
	var s kindSignals
	s.shape = p.Score
	for li := range kr.bags {
		s.lens[li] = fingerprint.ScoreLens(kr.bags[li][p.AIdx], kr.bags[li][p.BIdx], kr.idfs[li])
	}
	s.class = s.lens.Class()
	s.calleeJ = jaccardStrings(a.Callees, b.Callees)
	s.resolvedJ = jaccardStrings(da.ResolvedCallees, db.ResolvedCallees)
	s.callerJ = jaccardStrings(da.Callers, db.Callers)
	if p.Retrieval != nil {
		s.callSim = p.Retrieval.CallSim
	}
	s.maxNodes = max(a.Fingerprint.Nodes, b.Fingerprint.Nodes)
	s.minNodes = min(a.Fingerprint.Nodes, b.Fingerprint.Nodes)
	for _, c := range da.ResolvedCallees {
		if slices.Contains(db.ResolvedCallees, c) {
			s.sharedResolved++
		}
	}
	s.antonym = isAntonymPair(a, b)
	s.sameReceiver = a.ReceiverType != "" && a.ReceiverType == b.ReceiverType
	entry := func(u parser.CodeUnit) bool {
		n := parser.MethodName(u)
		return n == "main" || n == "init" || n == "run"
	}
	s.entry = entry(a) && entry(b)
	s.separateBinaries = a.Package == "main" && b.Package == "main" &&
		filepath.Dir(a.File) != filepath.Dir(b.File)
	s.sameFile = a.File == b.File
	if k := analyzer.ClassifyPairIn(a, b, p.Score, analyzer.ForkShapeFloor, analyzer.PairContext{
		ResolvedA: da.ResolvedCallees, ResolvedB: db.ResolvedCallees,
		VocabA: kr.thin[p.AIdx], VocabB: kr.thin[p.BIdx],
		CallersA: da.Callers, CallersB: db.Callers,
		CallerPkgsA: da.CallerPackages, CallerPkgsB: db.CallerPackages,
	}); k != nil {
		s.prodKind = k.Kind
	}
	return s
}

// labelledPair is one label resolved to its compared pair.
type labelledPair struct {
	corpus string
	label  Label
	pair   analyzer.SimilarPair
	sig    kindSignals
}

func (kr *kindRun) labelled() []labelledPair {
	byKey := map[string][]scoredPair{}
	idx := map[string][]int{}
	for i, p := range kr.run.Pairs {
		sp := kr.run.scoredPair(p)
		byKey[sp.key] = append(byKey[sp.key], sp)
		idx[sp.key] = append(idx[sp.key], i)
	}
	var out []labelledPair
	for _, l := range kr.spec.lf.Labels {
		k := pairKey(l.A, l.B)
		for j, c := range byKey[k] {
			if _, ok := firstMatch([]scoredPair{c}, l, kr.run.Root); ok {
				p := kr.run.Pairs[idx[k][j]]
				out = append(out, labelledPair{kr.name, l, p, kr.signals(p)})
				break
			}
		}
	}
	return out
}

func labelBucket(l Label) string {
	if l.Class == "false_positive" {
		if l.Kind == "" {
			return "fp/?"
		}
		return "fp/" + l.Kind
	}
	return l.Class
}

// kindFlag is one candidate lens rule: a predicate over a pair's signals that
// claims the pair is a false positive of some kind.
type kindFlag struct {
	name string
	on   func(kindSignals) bool
}

func kindFlags() []kindFlag {
	return []kindFlag{
		{"inverse (lens)", func(s kindSignals) bool { return s.class == "inverse" }},
		{"opposite names", func(s kindSignals) bool { return s.antonym }},
		{"mirror = opposite names | inverse", func(s kindSignals) bool { return s.antonym || s.class == "inverse" }},
		{"entry or separate binaries", func(s kindSignals) bool { return s.entry || s.separateBinaries }},
		{"thin: max nodes<=30 & shared helper", func(s kindSignals) bool { return s.maxNodes <= 30 && s.sharedResolved > 0 }},
		{"thin: max nodes<=20", func(s kindSignals) bool { return s.maxNodes <= 20 }},
		{"template (lens)", func(s kindSignals) bool { return s.class == "template" }},
		{"payload differs: callee J<0.25", func(s kindSignals) bool { return s.calleeJ < 0.25 }},
		{"skeleton gain >= 0.15", func(s kindSignals) bool {
			return s.lens[fingerprint.LensSkeleton].Jaccard-s.lens[fingerprint.LensShape].Jaccard >= 0.15
		}},
		{"sibling: same receiver, opposite names", func(s kindSignals) bool { return s.sameReceiver && s.antonym }},
		{"thin & names differ: <=30, helper, vocab<0.8", thinNamesDiffer},
		{"combined: sibling | callee J<0.25 | thin&vocab<0.8", combinedKinds},
		{"production: " + analyzer.KindMirror, func(s kindSignals) bool { return s.prodKind == analyzer.KindMirror }},
		{"production: " + analyzer.KindThinWrappers, func(s kindSignals) bool { return s.prodKind == analyzer.KindThinWrappers }},
		{"production: " + analyzer.KindDifferentCalls, func(s kindSignals) bool { return s.prodKind == analyzer.KindDifferentCalls }},
		{"production: any of the three", func(s kindSignals) bool {
			return s.prodKind == analyzer.KindMirror || s.prodKind == analyzer.KindThinWrappers || s.prodKind == analyzer.KindDifferentCalls
		}},
	}
}

// thinNamesDiffer is the thin-wrapper rule with the vocab lens as its guard:
// a small body delegating to a shared helper is a wrapper when the two sides
// name different things, and a clone worth merging when they name the same.
func thinNamesDiffer(s kindSignals) bool {
	return s.maxNodes <= 30 && s.sharedResolved > 0 && s.lens[fingerprint.LensVocab].Jaccard < 0.8
}

// combinedKinds is every rule that fired on no merge or refactor in the first
// pooled measurement, OR'd.
func combinedKinds(s kindSignals) bool {
	return (s.sameReceiver && s.antonym) || s.calleeJ < 0.25 || thinNamesDiffer(s)
}

// TestKindLenses measures per-kind false-positive signals against labelled
// corpora given as DOPPEL_BENCH_KINDS="<corpus>=<labels>[=<family-min>];…".
//
// Three tables. The first is each continuous signal's median per label class
// (merge, refactor, and each false-positive kind), which is what a threshold
// would be chosen from. The second is each candidate flag's firing rate per
// class, pooled over corpora: a flag earns attention when it fires on its own
// kind and rarely on merge and refactor. The third ranks the labels with a
// flagged pair's retrieval mass discounted, the only lever that moves the
// report, and prints the pooled scorecard. Asserts nothing: a flag drawn from
// the same labels it is scored on is a hypothesis, not a result.
func TestKindLenses(t *testing.T) {
	spec := os.Getenv("DOPPEL_BENCH_KINDS")
	if spec == "" {
		t.Skip("set DOPPEL_BENCH_KINDS=<corpus>=<labels>;... to measure the per-kind lens signals")
	}
	var runs []*kindRun
	var all []labelledPair
	for _, s := range parseLabeledSpecs(t, spec) {
		kr := loadKindRun(t, s)
		runs = append(runs, kr)
		lp := kr.labelled()
		t.Logf("[%s] %d functions, %d compared pairs, %d of %d labels compared", kr.name,
			len(kr.run.Units), len(kr.run.Pairs), len(lp), len(s.lf.Labels))
		all = append(all, lp...)
	}

	for _, lp := range all {
		if lp.sig.prodKind != "" && lp.label.Class != "false_positive" {
			t.Logf("production kind %q on a %s label: %s / %s — %s", lp.sig.prodKind, lp.label.Class, lp.label.A, lp.label.B, lp.label.Note)
		}
	}

	buckets := map[string][]labelledPair{}
	for _, lp := range all {
		b := labelBucket(lp.label)
		buckets[b] = append(buckets[b], lp)
	}
	var names []string
	for b := range buckets {
		names = append(names, b)
	}
	sort.Slice(names, func(i, j int) bool {
		rank := func(s string) int {
			switch s {
			case "merge":
				return 0
			case "refactor":
				return 1
			}
			return 2
		}
		if rank(names[i]) != rank(names[j]) {
			return rank(names[i]) < rank(names[j])
		}
		return names[i] < names[j]
	})

	median := func(ps []labelledPair, f func(kindSignals) float64) float64 {
		v := make([]float64, len(ps))
		for i, p := range ps {
			v[i] = f(p.sig)
		}
		sort.Float64s(v)
		if len(v) == 0 {
			return 0
		}
		return v[len(v)/2]
	}
	cols := []struct {
		name string
		f    func(kindSignals) float64
	}{
		{"shape", func(s kindSignals) float64 { return s.shape }},
		{"skel", func(s kindSignals) float64 { return s.lens[fingerprint.LensSkeleton].Jaccard }},
		{"ordGap", func(s kindSignals) float64 {
			return s.lens[fingerprint.LensShape].Jaccard - s.lens[fingerprint.LensOrdered].Jaccard
		}},
		{"vocab", func(s kindSignals) float64 { return s.lens[fingerprint.LensVocab].Jaccard }},
		{"calleeJ", func(s kindSignals) float64 { return s.calleeJ }},
		{"resolvJ", func(s kindSignals) float64 { return s.resolvedJ }},
		{"callerJ", func(s kindSignals) float64 { return s.callerJ }},
		{"callSim", func(s kindSignals) float64 { return s.callSim }},
		{"maxNode", func(s kindSignals) float64 { return float64(s.maxNodes) }},
	}
	head := fmt.Sprintf("  %-22s %4s", "class", "n")
	for _, c := range cols {
		head += fmt.Sprintf(" %7s", c.name)
	}
	t.Log("medians per label class, pooled")
	t.Log(head)
	for _, b := range names {
		line := fmt.Sprintf("  %-22s %4d", b, len(buckets[b]))
		for _, c := range cols {
			line += fmt.Sprintf(" %7.2f", median(buckets[b], c.f))
		}
		t.Log(line)
	}

	t.Log("flag firing rate per label class, pooled (fired/n)")
	for _, fl := range kindFlags() {
		line := fmt.Sprintf("  %-38s", fl.name)
		tp, tpN, fp, fpN := 0, 0, 0, 0
		for _, b := range names {
			n := 0
			for _, lp := range buckets[b] {
				if fl.on(lp.sig) {
					n++
				}
			}
			line += fmt.Sprintf(" %s %d/%d", strings.TrimPrefix(b, "fp/"), n, len(buckets[b]))
			if strings.HasPrefix(b, "fp/") {
				fp += n
				fpN += len(buckets[b])
			} else {
				tp += n
				tpN += len(buckets[b])
			}
		}
		prec := 0.0
		if tp+fp > 0 {
			prec = float64(fp) / float64(tp+fp)
		}
		t.Logf("%s  || fp %d/%d, true %d/%d, precision %.2f", line, fp, fpN, tp, tpN, prec)
	}

	t.Log("ranking with a flagged pair's retrieval mass discounted, pooled over corpora")
	t.Logf("  %-38s %5s %6s %6s %6s %6s %5s %5s", "flag", "disc", "merge", "refac", "fp", "fp@20", "viol", "m@50")
	discount := func(fl *kindFlag, d float64) {
		var mSum, rSum, fSum float64
		var mN, rN, fN, fp20, viol, m50 int
		for _, kr := range runs {
			orig := kr.run.Pairs
			pairs := make([]analyzer.SimilarPair, len(orig))
			copy(pairs, orig)
			if fl != nil {
				for i := range pairs {
					if pairs[i].Retrieval != nil && fl.on(kr.signals(pairs[i])) {
						r := *pairs[i].Retrieval
						r.Total *= d
						pairs[i].Retrieval = &r
					}
				}
			}
			kr.run.Pairs = pairs
			sc := Score(kr.run, kr.spec.lf)
			kr.run.Pairs = orig
			mSum += sc.MeanRank["merge"] * float64(sc.Present["merge"])
			mN += sc.Present["merge"]
			rSum += sc.MeanRank["refactor"] * float64(sc.Present["refactor"])
			rN += sc.Present["refactor"]
			fSum += sc.MeanRank["false_positive"] * float64(sc.Present["false_positive"])
			fN += sc.Present["false_positive"]
			fp20 += len(sc.FPInTop20)
			viol += len(sc.FPInTop20) + len(sc.FPAboveMerge) + len(sc.MergeMissing)
			m50 += sc.MergeInTop50
		}
		name, ds := "production", "-"
		if fl != nil {
			name, ds = fl.name, fmt.Sprintf("%.2f", d)
		}
		t.Logf("  %-38s %5s %6.1f %6.1f %6.1f %6d %5d %5d", name, ds,
			mSum/float64(max(mN, 1)), rSum/float64(max(rN, 1)), fSum/float64(max(fN, 1)), fp20, viol, m50)
	}
	discount(nil, 1)
	for _, fl := range kindFlags() {
		for _, d := range []float64{0.5, 0.25, 0.1} {
			discount(&fl, d)
		}
	}

	// The combined rule per corpus: a rule tuned on pooled labels has to hold
	// on each corpus separately, or it is fitting one corpus's idiom.
	t.Log("combined rule per corpus, production -> discount 0.25")
	for _, kr := range runs {
		base := Score(kr.run, kr.spec.lf)
		orig := kr.run.Pairs
		pairs := make([]analyzer.SimilarPair, len(orig))
		copy(pairs, orig)
		for i := range pairs {
			if pairs[i].Retrieval != nil && combinedKinds(kr.signals(pairs[i])) {
				r := *pairs[i].Retrieval
				r.Total *= 0.25
				pairs[i].Retrieval = &r
			}
		}
		kr.run.Pairs = pairs
		sc := Score(kr.run, kr.spec.lf)
		kr.run.Pairs = orig
		t.Logf("  %-8s merge %5.1f -> %5.1f  refactor %5.1f -> %5.1f  fp %5.1f -> %5.1f  fp@20 %2d -> %2d  merges@50 %d -> %d",
			kr.name, base.MeanRank["merge"], sc.MeanRank["merge"], base.MeanRank["refactor"], sc.MeanRank["refactor"],
			base.MeanRank["false_positive"], sc.MeanRank["false_positive"], len(base.FPInTop20), len(sc.FPInTop20),
			base.MergeInTop50, sc.MergeInTop50)
	}
}
