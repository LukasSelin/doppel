package main

// historylabel rolling-summary pools the two outcome comparisons over several
// origins per corpus: examples/rolling-origin.md. Each (corpus, origin) is a
// unit holding one ranker-outcome file and one clone-outcome file judged on
// that origin's own window.
//
//	historylabel rolling-summary -out summary.md -origins <dir of <corpus>.origins.tsv> \
//	    a.outcomes.json a.clone-outcomes.json b.o2.outcomes.json ...
//
// A file's origin is its rank among its corpus's distinct `since` dates,
// latest first: k = 1 is the cost study's T. The constants are the
// pre-registered rule and are not tunable; every draw is a fixed-seed LCG.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// unit is one corpus at one origin.
type unit struct {
	corpus string
	k      int
	files  []compareOut // the ranker file, the clone file, whichever exist
}

func (u unit) label() string { return fmt.Sprintf("%s o%d", u.corpus, u.k) }

// seedSuffix keeps k = 1's per-unit CIs exactly the earlier summaries'.
func (u unit) seedSuffix() string {
	if u.k == 1 {
		return ""
	}
	return fmt.Sprintf(" o%d", u.k)
}

// unitList is one method's top primaryK at one unit, as per-pair M1 values,
// with the file it came from.
type unitList struct {
	c    compareOut
	idx  []int
	file int // which of the unit's files: doppel is compared within one file
}

func (l unitList) m1s() []float64 {
	v := make([]float64, len(l.idx))
	for i, x := range l.idx {
		v[i] = m1(l.c.Pairs[x])
	}
	return v
}

func rollingSummaryMain(args []string) {
	fs := flag.NewFlagSet("rolling-summary", flag.ExitOnError)
	outPath := fs.String("out", "", "write the markdown here (default stdout)")
	origins := fs.String("origins", "", "directory of <corpus>.origins.tsv from scripts/rolling-origin.sh")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}
	var all []compareOut
	for _, p := range fs.Args() {
		var c compareOut
		if err := readJSON(p, &c); err != nil {
			fmt.Fprintln(os.Stderr, "historylabel rolling-summary:", err)
			os.Exit(1)
		}
		all = append(all, c)
	}
	var w io.Writer = os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "historylabel rolling-summary:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	writeRollingSummary(w, unitsOf(all), *origins)
}

// unitsOf groups files into units: corpora in first-seen order, origins
// latest first.
func unitsOf(all []compareOut) []unit {
	var corpora []string
	dates := map[string][]string{}
	for _, c := range all {
		if dates[c.Corpus] == nil {
			corpora = append(corpora, c.Corpus)
		}
		if !slices.Contains(dates[c.Corpus], c.SinceDate+" "+c.Since) {
			dates[c.Corpus] = append(dates[c.Corpus], c.SinceDate+" "+c.Since)
		}
	}
	var units []unit
	for _, corpus := range corpora {
		d := dates[corpus]
		sort.Sort(sort.Reverse(sort.StringSlice(d)))
		for k, s := range d {
			u := unit{corpus: corpus, k: k + 1}
			for _, c := range all {
				if c.Corpus == corpus && c.SinceDate+" "+c.Since == s {
					u.files = append(u.files, c)
				}
			}
			units = append(units, u)
		}
	}
	return units
}

// listsAt is every method's top primaryK at a unit; a method in both files
// keeps its first. doppel's list is kept per file, because a baseline is
// compared with the doppel list judged in its own file.
func listsAt(u unit) (names []string, lists map[string]unitList, dops []unitList) {
	lists = map[string]unitList{}
	for fi, c := range u.files {
		for _, v := range viewsOf(c) {
			l := unitList{c, v.slice(1, primaryK), fi}
			if v.name == doppelMethod {
				dops = append(dops, l)
			}
			if _, ok := lists[v.name]; ok {
				continue
			}
			names = append(names, v.name)
			lists[v.name] = l
		}
	}
	return names, lists, dops
}

// against is doppel's list and X's at unit i, from one file; ok is false
// when X has no pairs there.
func against(lists []map[string]unitList, dops [][]unitList, i int, x string) (dop, xl unitList, ok bool) {
	xl, ok = lists[i][x]
	if !ok || len(xl.idx) == 0 || xl.file >= len(dops[i]) || len(dops[i][xl.file].idx) == 0 {
		return unitList{}, unitList{}, false
	}
	return dops[i][xl.file], xl, true
}

// pooled is D(X) over units with its stratified 95% bootstrap CI: the mean
// over corpora of the mean over each corpus's scored units of
// M1(doppel) - M1(X). Each replicate resamples every unit's two lists
// independently and nothing crosses a unit.
type pooledCell struct {
	d, lo, hi    float64
	units, corps int
}

func pooledDiff(units []unit, lists []map[string]unitList, dops [][]unitList, x, seed string, keep func(unit) bool) pooledCell {
	type pairVals struct {
		corpus int
		a, b   []float64
	}
	var vals []pairVals
	corpIdx := map[string]int{}
	for i, u := range units {
		if !keep(u) {
			continue
		}
		dop, xl, ok := against(lists, dops, i, x)
		if !ok {
			continue
		}
		ci, ok := corpIdx[u.corpus]
		if !ok {
			ci = len(corpIdx)
			corpIdx[u.corpus] = ci
		}
		vals = append(vals, pairVals{ci, dop.m1s(), xl.m1s()})
	}
	cell := pooledCell{units: len(vals), corps: len(corpIdx)}
	if len(vals) == 0 {
		cell.d, cell.lo, cell.hi = math.NaN(), math.NaN(), math.NaN()
		return cell
	}
	sum := make([]float64, len(corpIdx))
	cnt := make([]float64, len(corpIdx))
	combine := func(d []float64) float64 {
		for i := range sum {
			sum[i], cnt[i] = 0, 0
		}
		for i, v := range vals {
			sum[v.corpus] += d[i]
			cnt[v.corpus]++
		}
		t := 0.0
		for i := range sum {
			t += sum[i] / cnt[i]
		}
		return t / float64(len(sum))
	}
	mean := func(v []float64) float64 {
		s := 0.0
		for _, y := range v {
			s += y
		}
		return s / float64(len(v))
	}
	d := make([]float64, len(vals))
	for i, v := range vals {
		d[i] = mean(v.a) - mean(v.b)
	}
	cell.d = combine(d)
	rng := lcg{seedOf("rolling-origin pooled", x, seed)}
	reps := make([]float64, 0, bootReps)
	for range bootReps {
		for i, v := range vals {
			sa, sb := 0.0, 0.0
			for range v.a {
				sa += v.a[rng.intn(len(v.a))]
			}
			for range v.b {
				sb += v.b[rng.intn(len(v.b))]
			}
			d[i] = sa/float64(len(v.a)) - sb/float64(len(v.b))
		}
		reps = append(reps, combine(d))
	}
	sort.Float64s(reps)
	cell.lo = reps[int(0.025*float64(len(reps)-1))]
	cell.hi = reps[int(math.Ceil(0.975*float64(len(reps)-1)))]
	return cell
}

func verdictOf(lo, hi float64) string {
	switch {
	case lo > 0:
		return "**doppel beats it**"
	case hi < 0:
		return "**it beats doppel**"
	}
	return "not distinguishable"
}

// originRow is one line of <corpus>.origins.tsv.
type originRow struct {
	corpus, k, since, sinceDate, end, endDate, commits string
}

func readOrigins(dir string, corpora []string) []originRow {
	var out []originRow
	for _, c := range corpora {
		f, err := os.Open(filepath.Join(dir, c+".origins.tsv"))
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for first := true; s.Scan(); first = false {
			x := strings.Split(s.Text(), "\t")
			if first || len(x) != 7 {
				continue
			}
			out = append(out, originRow{x[0], x[1], x[2], x[3], x[4], x[5], x[6]})
		}
		f.Close()
	}
	return out
}

func writeRollingSummary(w io.Writer, units []unit, originsDir string) {
	var corpora []string
	for _, u := range units {
		if !slices.Contains(corpora, u.corpus) {
			corpora = append(corpora, u.corpus)
		}
	}
	lists := make([]map[string]unitList, len(units))
	dops := make([][]unitList, len(units))
	var baselines []string
	for i, u := range units {
		names, l, d := listsAt(u)
		lists[i], dops[i] = l, d
		for _, n := range names {
			if n != doppelMethod && !slices.Contains(baselines, n) {
				baselines = append(baselines, n)
			}
		}
	}

	if originsDir != "" {
		fmt.Fprintf(w, "### Origins\n\n")
		fmt.Fprintf(w, "Window k is T_k..T_{k−1}; Go commits are non-merge commits touching `*.go` in it (the admissibility count, at least 100).\n\n")
		fmt.Fprintf(w, "| corpus | k | T_k | T_k date | window end date | Go commits | judged |\n| --- | ---: | --- | --- | --- | ---: | --- |\n")
		for _, o := range readOrigins(originsDir, corpora) {
			judged := "no"
			for _, u := range units {
				if u.corpus == o.corpus && fmt.Sprint(u.k) == o.k {
					judged = "yes"
				}
			}
			sha := o.since
			if len(sha) > 9 {
				sha = sha[:9]
			}
			fmt.Fprintf(w, "| %s | %s | `%s` | %s | %s | %s | %s |\n", o.corpus, o.k, sha, o.sinceDate, o.endDate, o.commits, judged)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "### Units\n\n")
	fmt.Fprintf(w, "| unit | T_k | T_k date | end date | window commits replayed | functions at T_k | calibration | union | unresolved | doppel's list identical in both files |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: | --- |\n")
	for _, u := range units {
		c := u.files[0]
		var replayed, unresolved []string
		for _, f := range u.files {
			replayed = append(replayed, fmt.Sprint(f.WindowCommits))
			unresolved = append(unresolved, fmt.Sprint(f.Unresolved))
		}
		same := "—"
		if len(u.files) == 2 {
			a, b := listKeys(u.files[0], doppelMethod), listKeys(u.files[1], doppelMethod)
			same = "yes"
			if len(a) != len(b) {
				same = "**no**"
			}
			for i := range min(len(a), len(b)) {
				if a[i] != b[i] {
					same = "**no**"
					break
				}
			}
		}
		fmt.Fprintf(w, "| %s | `%s` | %s | %s | %s | %d | %s | %d | %s | %s |\n", u.label(), c.Since[:9], c.SinceDate, c.PinDate,
			strings.Join(replayed, " / "), c.Functions, c.Calibration, c.Union, strings.Join(unresolved, " / "), same)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### M1@%d per unit\n\n", primaryK)
	fmt.Fprintf(w, "Each cell is M1@%d, then the pairs with any event in that top %d, then the distinct commits behind them. Pooled random holds three lists of %d.\n\n", primaryK, primaryK, primaryK)
	fmt.Fprintf(w, "| method |")
	for _, u := range units {
		fmt.Fprintf(w, " %s |", u.label())
	}
	fmt.Fprintf(w, "\n| --- |")
	for range units {
		fmt.Fprintf(w, " ---: |")
	}
	fmt.Fprintln(w)
	for _, m := range append([]string{doppelMethod}, baselines...) {
		fmt.Fprintf(w, "| %s |", m)
		for i := range units {
			l, ok := lists[i][m]
			if !ok || len(l.idx) == 0 {
				fmt.Fprintf(w, " — |")
				continue
			}
			r := rateOf(l.c, l.idx)
			events := int(math.Round(r.e * float64(r.n)))
			fmt.Fprintf(w, " %.3f · %d · %d |", r.m1, events, r.commits)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Per unit: M1@%d, doppel minus baseline\n\n", primaryK)
	fmt.Fprintf(w, "95%% percentile bootstrap per unit, as in the earlier studies (k = 1 reproduces their CIs). ✓ lower bound above 0, ✗ upper bound below 0.\n\n")
	fmt.Fprintf(w, "| baseline |")
	for _, u := range units {
		fmt.Fprintf(w, " %s |", u.label())
	}
	fmt.Fprintf(w, " units won | units lost | count rule |\n| --- |")
	for range units {
		fmt.Fprintf(w, " --- |")
	}
	fmt.Fprintf(w, " ---: | ---: | --- |\n")
	for _, b := range baselines {
		fmt.Fprintf(w, "| %s |", b)
		win, loss, scored := 0, 0, 0
		for i, u := range units {
			dop, xl, ok := against(lists, dops, i, b)
			if !ok {
				fmt.Fprintf(w, " — |")
				continue
			}
			d, lo, hi := diffCI(xl.c, dop.idx, xl.idx, b+u.seedSuffix())
			scored++
			mark := ""
			switch {
			case lo > 0:
				win++
				mark = " ✓"
			case hi < 0:
				loss++
				mark = " ✗"
			}
			fmt.Fprintf(w, " %+.3f [%+.3f, %+.3f]%s |", d, lo, hi, mark)
		}
		rule := "not distinguishable"
		switch {
		case 2*win >= scored && loss == 0 && win > 0:
			rule = "doppel beats it"
		case 2*loss >= scored && win == 0 && loss > 0:
			rule = "it beats doppel"
		}
		fmt.Fprintf(w, " %d of %d | %d | %s |\n", win, scored, loss, rule)
	}
	fmt.Fprintln(w)

	all := func(unit) bool { return true }
	fmt.Fprintf(w, "### Pooled verdict (primary)\n\n")
	fmt.Fprintf(w, "D = mean over corpora of the mean over that corpus's units of M1@%d(doppel) − M1@%d(X); 95%% bootstrap CI stratified by unit (%d replicates). doppel beats X when the lower bound is above 0, X beats doppel when the upper bound is below 0.\n\n", primaryK, primaryK, bootReps)
	fmt.Fprintf(w, "| baseline | units | corpora | D [95%% CI] | verdict |\n| --- | ---: | ---: | --- | --- |\n")
	for _, b := range baselines {
		c := pooledDiff(units, lists, dops, b, "", all)
		fmt.Fprintf(w, "| %s | %d | %d | %+.4f [%+.4f, %+.4f] | %s |\n", b, c.units, c.corps, c.d, c.lo, c.hi, verdictOf(c.lo, c.hi))
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Replication: earlier origins only (descriptive)\n\n")
	fmt.Fprintf(w, "The same pooled D over the units with k ≥ 2: windows none of the earlier studies read.\n\n")
	fmt.Fprintf(w, "| baseline | units | corpora | D [95%% CI] | reads as |\n| --- | ---: | ---: | --- | --- |\n")
	for _, b := range baselines {
		c := pooledDiff(units, lists, dops, b, "replication", func(u unit) bool { return u.k >= 2 })
		fmt.Fprintf(w, "| %s | %d | %d | %+.4f [%+.4f, %+.4f] | %s |\n", b, c.units, c.corps, c.d, c.lo, c.hi, strings.Trim(verdictOf(c.lo, c.hi), "*"))
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Leave one corpus out (descriptive)\n\n")
	fmt.Fprintf(w, "Pooled D [95%% CI] with the named corpus removed.\n\n| baseline |")
	for _, c := range corpora {
		fmt.Fprintf(w, " without %s |", c)
	}
	fmt.Fprintf(w, "\n| --- |")
	for range corpora {
		fmt.Fprintf(w, " --- |")
	}
	fmt.Fprintln(w)
	for _, b := range baselines {
		fmt.Fprintf(w, "| %s |", b)
		for _, out := range corpora {
			c := pooledDiff(units, lists, dops, b, "without "+out, func(u unit) bool { return u.corpus != out })
			mark := ""
			switch {
			case c.lo > 0:
				mark = " ✓"
			case c.hi < 0:
				mark = " ✗"
			}
			fmt.Fprintf(w, " %+.4f [%+.4f, %+.4f]%s |", c.d, c.lo, c.hi, mark)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Clustered M1 (descriptive)\n\n")
	fmt.Fprintf(w, "D with each co-change commit's credit divided among the pairs of the list it touches, pooled the same way, without a CI.\n\n")
	fmt.Fprintf(w, "| baseline | D clustered |\n| --- | ---: |\n")
	for _, b := range baselines {
		perCorpus := map[string][]float64{}
		for i, u := range units {
			dop, xl, ok := against(lists, dops, i, b)
			if !ok {
				continue
			}
			perCorpus[u.corpus] = append(perCorpus[u.corpus], rateOf(dop.c, dop.idx).m1clus-rateOf(xl.c, xl.idx).m1clus)
		}
		t, n := 0.0, 0
		for _, c := range corpora {
			if v := perCorpus[c]; len(v) > 0 {
				s := 0.0
				for _, x := range v {
					s += x
				}
				t += s / float64(len(v))
				n++
			}
		}
		fmt.Fprintf(w, "| %s | %+.4f |\n", b, t/float64(max(n, 1)))
	}
}
