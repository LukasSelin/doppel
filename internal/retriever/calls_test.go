package retriever

import (
	"math"
	"testing"

	"github.com/lukse/doppel/internal/concepter"
)

// Requirement: syntactically different functions sharing a rare
// import-qualified external call meet through the call channel, keyed on the
// full import path so different packages converge on the same token.
func TestCallChannelRetrievesSharedExternalAPI(t *testing.T) {
	fileA := `package orders

import "database/sql"

func OpenOrders(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}
`
	fileB := `package billing

import "database/sql"

func ConnectBilling(cfg map[string]string) (int, error) {
	total := 0
	for key, dsn := range cfg {
		if key == "" {
			continue
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return total, err
		}
		total++
		_ = db
	}
	return total, nil
}
`
	fillerC := `package filler

func Pad(xs []int) int {
	n := 0
	for _, x := range xs {
		if x > 0 {
			n += x
		}
	}
	return n
}
`
	units := parseUnits(t, "orders.go", fileA, "billing.go", fileB, "filler.go", fillerC)
	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.Threshold = 0.99

	cands, stats := retrieveAll(t, units, opt)
	a := unitIndex(t, units, "OpenOrders")
	b := unitIndex(t, units, "ConnectBilling")
	c, ok := findCandidate(cands, a, b)
	if !ok {
		t.Fatal("pair sharing database/sql.Open not retrieved via the call channel")
	}
	if c.Call <= 0 {
		t.Errorf("Call evidence = %v, want > 0", c.Call)
	}
	hasCall := false
	for _, ch := range c.Channels {
		if ch == ChannelCall {
			hasCall = true
		}
	}
	if !hasCall {
		t.Errorf("Channels = %v, want to include call", c.Channels)
	}
	if stats.CallPairs < 1 {
		t.Errorf("CallPairs = %d, want >= 1", stats.CallPairs)
	}
}

// Shared resolved internal callees are call evidence too, through the
// qualified call-graph edges.
func TestCallChannelRetrievesSharedInternalCallees(t *testing.T) {
	src := `package fix

func normalize(s string) string {
	out := ""
	for _, c := range s {
		if c != ' ' {
			out += string(c)
		}
	}
	return out
}

func CleanLabel(label string) string {
	if label == "" {
		return label
	}
	trimmed := normalize(label)
	return trimmed
}

func CanonicalName(parts []string) string {
	joined := ""
	for _, p := range parts {
		joined += normalize(p)
	}
	return joined
}
`
	units := parseUnits(t, "fix.go", src)
	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.Threshold = 0.99

	cands, _ := retrieveAll(t, units, opt)
	a := unitIndex(t, units, "CleanLabel")
	b := unitIndex(t, units, "CanonicalName")
	c, ok := findCandidate(cands, a, b)
	if !ok {
		t.Fatal("pair sharing internal callee fix.normalize not retrieved")
	}
	if c.Call <= 0 {
		t.Errorf("Call evidence = %v, want > 0", c.Call)
	}
}

// Sharing only corpus-wide calls must not create candidates: with the df cap
// at 3, the fmt.Sprintf token (df=5) drops out of the index, and only the
// pair that also shares a rare call is admitted — with mass counting the
// rare token alone.
func TestCallChannelSuppressesCommonCalls(t *testing.T) {
	src := `package fix

import (
	"database/sql"
	"fmt"
)

func FmtOnlyA(x int) string {
	out := ""
	for i := 0; i < x; i++ {
		out = fmt.Sprintf("%s-%d", out, i)
	}
	return out
}

func FmtOnlyB(y int) string {
	res := ""
	for j := 0; j < y; j++ {
		res = fmt.Sprintf("%s+%d", res, j)
	}
	return res
}

func FmtOnlyC(z int) string {
	acc := ""
	for k := 0; k < z; k++ {
		acc = fmt.Sprintf("%s*%d", acc, k)
	}
	return acc
}

func RareA(dsn string) error {
	label := fmt.Sprintf("conn-%s", dsn)
	db, err := sql.Open("postgres", label)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

func RareB(cfg string) error {
	name := fmt.Sprintf("db-%s", cfg)
	handle, err := sql.Open("mysql", name)
	if err != nil {
		return err
	}
	defer handle.Close()
	return nil
}
`
	units := parseUnits(t, "fix.go", src)
	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.Threshold = 1.01 // no structural admissions at all
	opt.MaxCallDF = 3

	cands, _ := retrieveAll(t, units, opt)

	fmtOnly := []string{"FmtOnlyA", "FmtOnlyB", "FmtOnlyC"}
	for i := 0; i < len(fmtOnly); i++ {
		for j := i + 1; j < len(fmtOnly); j++ {
			a, b := unitIndex(t, units, fmtOnly[i]), unitIndex(t, units, fmtOnly[j])
			if _, ok := findCandidate(cands, a, b); ok {
				t.Errorf("%s/%s retrieved on fmt.Sprintf alone; common calls must carry no evidence",
					fmtOnly[i], fmtOnly[j])
			}
		}
	}

	a, b := unitIndex(t, units, "RareA"), unitIndex(t, units, "RareB")
	c, ok := findCandidate(cands, a, b)
	if !ok {
		t.Fatal("rare-call pair not retrieved")
	}
	// df(database/sql.Open) = 2 over 5 units; the capped fmt.Sprintf must not
	// contribute, and defer'd Close calls are variable-receiver (excluded), so
	// the mass is exactly ln(5/2).
	if want := math.Log(5.0 / 2.0); math.Abs(c.Call-want) > 1e-9 {
		t.Errorf("Call evidence = %v, want ln(5/2) = %v (rare token only)", c.Call, want)
	}
}

// callTokens must not double-count an internal package called through its own
// import: the resolved graph edge covers it, and the import-qualified form is
// skipped.
func TestCallTokensDoNotDoubleCountInternalCalls(t *testing.T) {
	helperFile := `package helper

func Normalize(s string) string {
	out := ""
	for _, c := range s {
		if c != ' ' {
			out += string(c)
		}
	}
	return out
}
`
	callerFile := `package caller

import "example.com/proj/helper"

func Clean(s string) string {
	if s == "" {
		return s
	}
	return helper.Normalize(s)
}
`
	units := parseUnits(t, "helper.go", helperFile, "caller.go", callerFile)
	g := concepter.BuildCallGraph(units)

	internal := map[string]bool{}
	for i := range units {
		internal[concepter.QualifiedName(units[i])] = true
	}
	tokens := callTokens(units[unitIndex(t, units, "Clean")], g, internal)
	count := 0
	for _, tok := range tokens {
		if tok == "helper.Normalize" || tok == "example.com/proj/helper.Normalize" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("tokens = %v; internal call must appear exactly once, got %d forms", tokens, count)
	}
}

// Bare-name and variable-receiver calls never become tokens: unresolved
// matching is forbidden.
func TestCallTokensExcludeUnresolvedCalls(t *testing.T) {
	src := `package fix

func Use(tx interface{ Commit() error }) error {
	doWork()
	return tx.Commit()
}
`
	units := parseUnits(t, "fix.go", src)
	g := concepter.BuildCallGraph(units)
	tokens := callTokens(units[0], g, map[string]bool{})
	if len(tokens) != 0 {
		t.Errorf("tokens = %v, want none: doWork is bare and tx.Commit has a variable receiver", tokens)
	}
}
