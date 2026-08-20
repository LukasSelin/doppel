package parser

import (
	"strings"
	"testing"
)

// parseOne parses a snippet expected to contain exactly one declaration.
func parseOne(t *testing.T, src string) CodeUnit {
	t.Helper()
	units, err := ParseSource("test.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("got %d units, want 1", len(units))
	}
	return units[0]
}

func TestExtractSignals(t *testing.T) {
	u := parseOne(t, `package demo

import (
	"database/sql"
	"fmt"
)

// fetchUser loads one row.
func fetchUser(db *sql.DB, id int) error {
	// SELECT is mentioned here only in a comment.
	row := db.QueryRow("SELECT name FROM users WHERE id = ?", id)
	var name string
	if err := row.Scan(&name); err != nil {
		return fmt.Errorf("fetch user %d: %w", id, err)
	}
	return nil
}
`)
	sig := u.Signals

	if !sig.AnyImport("database/sql") {
		t.Errorf("imports = %v, want database/sql", sig.Imports)
	}
	if !sig.AnySelector("db.QueryRow") || !sig.AnySelector("row.Scan") {
		t.Errorf("selectors = %v, want db.QueryRow and row.Scan", sig.Selectors)
	}
	if !sig.AnyLiteral("SELECT ") {
		t.Errorf("literals = %v, want the SQL query text", sig.StringLits)
	}
	if !sig.AnyLiteral("%w") {
		t.Errorf("literals = %v, want the %%w-carrying format string", sig.StringLits)
	}
	// The comment's SELECT must NOT be in the literals — separating those
	// channels is the reason this type exists.
	for _, lit := range sig.StringLits {
		if strings.Contains(lit, "only in a comment") {
			t.Errorf("comment text leaked into string literals: %q", lit)
		}
	}
	if !sig.AnyIdent("fetchUser") {
		t.Errorf("idents = %v, want the function's own name", sig.IdentNames)
	}
	if sig.HasGoStmt || sig.HasSelect || sig.HasChan {
		t.Errorf("no concurrency constructs in the fixture, got go=%t select=%t chan=%t",
			sig.HasGoStmt, sig.HasSelect, sig.HasChan)
	}
}

func TestExtractSignalsConcurrency(t *testing.T) {
	u := parseOne(t, `package demo

func fanIn(inputs []<-chan int) <-chan int {
	out := make(chan int)
	for _, in := range inputs {
		go func(c <-chan int) {
			for v := range c {
				select {
				case out <- v:
				default:
				}
			}
		}(in)
	}
	return out
}
`)
	sig := u.Signals
	if !sig.HasGoStmt || !sig.HasSelect || !sig.HasChan {
		t.Errorf("go=%t select=%t chan=%t, want all true", sig.HasGoStmt, sig.HasSelect, sig.HasChan)
	}
}

// A channel appearing only in the signature still marks the function: it is
// coordinating concurrent work even if its body never mentions one.
func TestExtractSignalsChanInSignatureOnly(t *testing.T) {
	u := parseOne(t, `package demo

func drain(c <-chan int) {
	_ = c
}
`)
	if !u.Signals.HasChan {
		t.Error("HasChan = false for a function taking a channel parameter")
	}
}

func TestReceiverMatchingIsExact(t *testing.T) {
	u := parseOne(t, `package demo

import "sync"

func guarded(mtx *sync.Mutex, done func()) {
	mtx.Lock()
	defer mtx.Unlock()
	done()
}
`)
	sig := u.Signals
	// "mtx.Lock" contains the substring "tx." — the false positive the old
	// body-substring tagger had. Exact receiver matching must not fire.
	if sig.AnyReceiver("tx") {
		t.Errorf("AnyReceiver(tx) fired on mtx: selectors = %v", sig.Selectors)
	}
	if !sig.AnyReceiver("mtx") {
		t.Errorf("AnyReceiver(mtx) did not fire: selectors = %v", sig.Selectors)
	}
	if !sig.AnyMethod("Lock") {
		t.Errorf("AnyMethod(Lock) did not fire: selectors = %v", sig.Selectors)
	}
}

// Bare calls arrive as identifiers, and AnyMethod must see them too.
func TestAnyMethodSeesBareCalls(t *testing.T) {
	u := parseOne(t, `package demo

func run() {
	Rollback()
}
`)
	if !u.Signals.AnyMethod("Rollback") {
		t.Errorf("AnyMethod(Rollback) missed a bare call: idents = %v", u.Signals.IdentNames)
	}
}

func TestSignalsAreSortedAndDeduped(t *testing.T) {
	u := parseOne(t, `package demo

import "fmt"

func noisy() {
	fmt.Println("b")
	fmt.Println("a")
	fmt.Println("a")
}
`)
	sig := u.Signals
	for name, list := range map[string][]string{
		"Imports": sig.Imports, "Selectors": sig.Selectors,
		"StringLits": sig.StringLits, "IdentNames": sig.IdentNames,
	} {
		for i := 1; i < len(list); i++ {
			if list[i-1] >= list[i] {
				t.Errorf("%s not strictly sorted/deduped: %v", name, list)
				break
			}
		}
	}
}

// The signals walk must not disturb what already existed: Callees feeds the
// call graph and Fingerprint the code-similarity score.
func TestSignalsDoNotChangeCalleesOrFingerprint(t *testing.T) {
	src := `package demo

import "fmt"

func greet(name string) {
	fmt.Println("hello", name)
	helper()
}

func helper() {}
`
	units, err := ParseSource("test.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("got %d units, want 2", len(units))
	}
	g := units[0]
	wantCallees := []string{"fmt.Println", "helper"}
	if len(g.Callees) != len(wantCallees) {
		t.Fatalf("Callees = %v, want %v", g.Callees, wantCallees)
	}
	for i := range wantCallees {
		if g.Callees[i] != wantCallees[i] {
			t.Fatalf("Callees = %v, want %v", g.Callees, wantCallees)
		}
	}
	if g.Fingerprint.Nodes == 0 || len(g.Fingerprint.Shingles) == 0 {
		t.Error("fingerprint went missing")
	}
}
