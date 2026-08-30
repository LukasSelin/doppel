package bench

import (
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestMinNodesDistribution reports, per fetched corpus, the quantiles of the
// body-node distribution and where the shipped absolute --min-nodes 18 sits in
// it. It exists to answer whether the absolute floor could be replaced by a
// corpus-relative one (a quantile of the corpus's own node distribution, the
// way calibrate derives thresholds), which would let a small corpus keep a
// shape channel that 18 nearly closes.
//
// The two pins any relative floor has to satisfy simultaneously are printed
// alongside: cobra's 16-node `Less` false positive must stay excluded, and
// conc's `Wait` clone family must be admitted.
//
// Asserts nothing.
//
//	DOPPEL_BENCH_MINNODES=1 go test ./internal/bench/ -v -run TestMinNodesDistribution
func TestMinNodesDistribution(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_MINNODES") != "1" {
		t.Skip("set DOPPEL_BENCH_MINNODES=1 to run the min-nodes distribution measurement")
	}
	qs := []float64{0.10, 0.20, 0.25, 0.30, 0.40, 0.50}

	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		path, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		units, err := Load(path, "exclude")
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		nodes := make([]int, 0, len(units))
		for _, u := range units {
			nodes = append(nodes, u.Fingerprint.Nodes)
		}
		sort.Ints(nodes)

		var parts []string
		for _, q := range qs {
			i := int(q * float64(len(nodes)))
			if i >= len(nodes) {
				i = len(nodes) - 1
			}
			parts = append(parts, "q"+trimQ(q)+"="+itoa(nodes[i]))
		}
		// Where does the absolute 18 sit in this corpus?
		below := sort.SearchInts(nodes, 18)
		t.Logf("[%s] n=%d  %s  |  18 excludes %d units (%.0f%% quantile)",
			c.Name, len(nodes), strings.Join(parts, " "), below,
			100*float64(below)/float64(len(nodes)))

		// The two pins.
		for _, u := range units {
			q := qualifiedName(u)
			if strings.HasSuffix(q, ".Less") || strings.HasSuffix(q, "].Wait") || strings.HasSuffix(q, "Pool.Wait") {
				t.Logf("    pin %-46s nodes=%d", q, u.Fingerprint.Nodes)
			}
		}
	}
}

// TestMinNodesLadder scores every labeled corpus at each candidate
// --min-nodes floor. The interesting region is 15..18: cobra's `Less` false
// positive is 15 nodes and conc's `Wait` clone family is 16, so 16 is the only
// absolute floor that excludes the first while admitting the second.
//
// Asserts nothing.
//
//	DOPPEL_BENCH_MINNODES=1 go test ./internal/bench/ -v -run TestMinNodesLadder
func TestMinNodesLadder(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_MINNODES") != "1" {
		t.Skip("set DOPPEL_BENCH_MINNODES=1 to run the min-nodes ladder")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora`")
	}
	for _, lc := range corpora {
		saved := snapshotRetrieval(lc.run)
		for _, mn := range []int{12, 14, 15, 16, 17, 18, 20} {
			opt := retriever.DefaultOptions()
			opt.MinNodes = mn
			lc.run.Reretrieve(opt)
			sc := Score(lc.run, lc.lf)
			t.Logf("[%s] min-nodes %2d  pairs %5d  %s", lc.name, mn, len(lc.run.Pairs), scLine(sc))
		}
		saved.restore(lc.run)
	}
}

func trimQ(q float64) string {
	s := []byte{byte('0' + int(q*100)/10), byte('0' + int(q*100)%10)}
	return string(s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
