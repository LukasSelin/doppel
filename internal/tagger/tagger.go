package tagger

import (
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// patternRule maps a concept to the AST-level evidence that asserts it.
//
// Each field is one evidence channel from parser.TagSignals, and the channels
// deliberately have different matching semantics:
//
//	selectors  exact "x.Sel" expressions        http.Get, sync.Map
//	methods    exact method or bare call name   QueryRow, Rollback
//	receivers  exact receiver identifier        tx.Commit fires, mtx.Lock does not
//	imports    substring of an import path      database/sql
//	literals   substring of string CONTENTS     "SELECT " in a query, not in a comment
//	idents     substring of an identifier name  retryWithBackoff, maxRetries
//	flags      structural node-kind predicate   go statements, select, channels
//
// The predecessor of this table substring-scanned raw source, which could not
// tell a comment from a query string: a comment saying "COMMIT" tagged its
// function transaction, mtx.Lock() matched the keyword "tx.", and the
// error-wrapping rule matched %w only when it immediately preceded the closing
// quote. Its polyglot leftovers (axios, urllib, Promise.) are gone too — only
// .go files are ever parsed.
//
// Rules still name ontology terms so the vocabulary and the tagger cannot
// drift apart, and tagger_test enforces the reverse direction: every concrete
// concept has exactly one rule.
type patternRule struct {
	concept   ontology.TermID
	selectors []string
	methods   []string
	receivers []string
	imports   []string
	literals  []string
	idents    []string
	flags     func(parser.TagSignals) bool
}

// Order matters: tags are emitted in this declaration order.
var patternRules = []patternRule{
	{
		concept: ontology.ConRetry,
		// Retry has no structural AST handle — its evidence is genuinely
		// lexical, living in names like retryWithBackoff and MaxRetries.
		idents: []string{"retry", "Retry", "Retries", "backoff", "Backoff", "BackOff"},
	},
	{
		concept:   ontology.ConHTTPCall,
		selectors: []string{"http.Get", "http.Post", "http.Do", "http.NewRequest"},
		// Deliberately no net/http import signal: servers import it too, and
		// this tag means "makes an outbound call".
	},
	{
		concept:   ontology.ConDBAccess,
		selectors: []string{"sql.Open"},
		// Bare Query/Exec are too common on non-database types (url.Values has
		// a Query too); only the unambiguous method names match anywhere,
		// while the short ones need the conventional receiver.
		methods:   []string{"QueryRow", "QueryContext", "QueryRowContext", "ExecContext"},
		receivers: []string{"db"},
		imports:   []string{"database/sql"},
		literals:  []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE "},
	},
	{
		concept: ontology.ConValidation,
		idents:  []string{"validate", "Validate", "IsValid", "isValid", "ErrInvalid"},
		methods: []string{"Must", "assert"},
		literals: []string{
			"required", // validate:"required" struct-tag convention
		},
	},
	{
		concept:   ontology.ConMapping,
		selectors: []string{"json.Marshal", "json.Unmarshal"},
		idents:    []string{"transform", "Transform", "convert", "Convert", "ToDTO", "FromDTO", "toMap"},
	},
	{
		concept:   ontology.ConTransaction,
		methods:   []string{"Begin", "BeginTx", "Commit", "Rollback"},
		receivers: []string{"tx"},
		literals:  []string{"BEGIN TRANSACTION", "COMMIT", "ROLLBACK"},
	},
	{
		concept:   ontology.ConCaching,
		selectors: []string{"sync.Map"},
		receivers: []string{"cache", "redis"},
		idents:    []string{"cache", "Cache", "TTL", "memcache", "expire", "Expire"},
		imports:   []string{"redis", "memcache"},
	},
	{
		concept:   ontology.ConConcurrency,
		selectors: []string{"sync.Mutex", "sync.RWMutex", "sync.WaitGroup", "sync.Once"},
		receivers: []string{"atomic"},
		methods:   []string{"Lock", "Unlock"},
		flags: func(s parser.TagSignals) bool {
			return s.HasGoStmt || s.HasSelect || s.HasChan
		},
	},
	{
		concept: ontology.ConErrorWrapping,
		// Tightened on purpose: only genuine wrapping counts. A %w verb
		// anywhere in a format string (the old rule matched it only right
		// before the closing quote), or one of the pkg/errors wrap helpers.
		// Bare fmt.Errorf annotates without wrapping, and errors.As/Is
		// inspect errors rather than wrap them — none of them fire this tag
		// any more, which also makes it rare enough to be informative.
		literals:  []string{"%w"},
		selectors: []string{"errors.Wrap", "errors.Wrapf", "errors.WithMessage", "errors.WithStack"},
		methods:   []string{"Wrapf", "WithMessage", "WithStack"},
	},
}

// Tag returns the pattern labels detected in the unit's AST signals.
// Tags are returned in a deterministic order matching the rule declaration
// order; one matching channel is enough to apply a tag.
func Tag(u parser.CodeUnit) []string {
	var tags []string
	for _, rule := range patternRules {
		if rule.matches(u.Signals) {
			tags = append(tags, string(rule.concept))
		}
	}
	return tags
}

func (r patternRule) matches(s parser.TagSignals) bool {
	if len(r.selectors) > 0 && s.AnySelector(r.selectors...) {
		return true
	}
	if len(r.methods) > 0 && s.AnyMethod(r.methods...) {
		return true
	}
	if len(r.receivers) > 0 && s.AnyReceiver(r.receivers...) {
		return true
	}
	if len(r.imports) > 0 && s.AnyImport(r.imports...) {
		return true
	}
	if len(r.literals) > 0 && s.AnyLiteral(r.literals...) {
		return true
	}
	if len(r.idents) > 0 && s.AnyIdent(r.idents...) {
		return true
	}
	return r.flags != nil && r.flags(s)
}

// signalCount reports how many evidence channels a rule declares; every rule
// must declare at least one or it can never fire (checked by tagger_test).
func (r patternRule) signalCount() int {
	n := len(r.selectors) + len(r.methods) + len(r.receivers) +
		len(r.imports) + len(r.literals) + len(r.idents)
	if r.flags != nil {
		n++
	}
	return n
}
