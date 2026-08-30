// Package tagger holds the seed vocabulary: fourteen rules that propose a
// starting member set for a concept, and decide nothing else.
//
// These rules used to be the answer. Every term in them — "httpClient",
// "SELECT ", "gobreaker", "ToDTO" — is a guess about how some codebase writes
// something, and the comments below are a record of what maintaining those
// guesses cost: http.Do was dead weight for years because net/http has no such
// function, %w matched only immediately before a closing quote, and mapping had
// to surrender json.Marshal the day serialization arrived. The approach does
// not scale — not to more concepts, not to a repository that calls its database
// wrapper "store", and not at all to another language, where all of it would
// have to be written again by hand.
//
// So the rules were demoted rather than deleted. internal/lexicon takes each
// rule's matches as a *seed* — which functions to look at — and learns from the
// corpus what those functions actually share, producing concepts named after
// their own evidence with graded membership. A seed that grows nothing is a
// finding: this corpus has no such practice. And the learner works with no
// seeds at all, which is what makes another language's frontend a matter of
// filling parser.TagSignals rather than rewriting this file.
//
// What a rule still buys is a head start on a concept a human has a word for,
// so the table stays, and stays honest about the AST evidence it reads.

package tagger

import (
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// patternRule maps a concept to the AST-level evidence that seeds it.
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
// Rules still name ontology terms so the seed vocabulary and the tagger cannot
// drift apart, and tagger_test enforces the reverse direction: every concrete
// concept in the authored taxonomy has exactly one rule.
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
		concept: ontology.ConHTTPCall,
		// http.Do was dead weight for years — net/http has no such function
		// (Do is a method on *http.Client), so the term could never match.
		// NewRequestWithContext is the idiomatic constructor since Go 1.13.
		// The httpClient receiver catches wrapper clients (c.httpClient.Do),
		// which the nested-selector tail in extractSignals makes visible.
		selectors: []string{"http.Get", "http.Head", "http.NewRequest", "http.NewRequestWithContext", "http.Post", "http.PostForm"},
		receivers: []string{"httpClient"},
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
		concept: ontology.ConMapping,
		// json.Marshal/Unmarshal moved to the serialization rule when that
		// leaf arrived — otherwise every json function would carry both tags
		// forever. Mapping is now purely the in-memory conversion vocabulary.
		idents: []string{"transform", "Transform", "convert", "Convert", "ToDTO", "FromDTO", "toMap"},
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
	{
		concept: ontology.ConGRPCCall,
		// Client-side dialing only, matching http_call's outbound-call stance:
		// a gRPC *server* registers services and never dials. No bare grpc
		// import signal for the same reason net/http has none.
		selectors: []string{"grpc.Dial", "grpc.DialContext", "grpc.NewClient"},
	},
	{
		concept: ontology.ConCircuitBreaker,
		// Like retry, the evidence is genuinely lexical — Go has no structural
		// circuit-breaker handle — plus the one conventional library import.
		idents:  []string{"circuitBreaker", "CircuitBreaker", "breaker", "Breaker", "halfOpen", "HalfOpen"},
		imports: []string{"gobreaker"},
	},
	{
		concept: ontology.ConSerialization,
		// Selector evidence only — an encoding/json import is file-level and
		// half of Go imports it, so an import signal would tag every function
		// in every file that touches json anywhere. The method names are the
		// exact stdlib interface implementations.
		selectors: []string{
			"json.Marshal", "json.MarshalIndent", "json.NewDecoder", "json.NewEncoder", "json.Unmarshal",
			"xml.Marshal", "xml.NewDecoder", "xml.NewEncoder", "xml.Unmarshal",
			"yaml.Marshal", "yaml.Unmarshal",
			"gob.NewDecoder", "gob.NewEncoder",
			"proto.Marshal", "proto.Unmarshal",
		},
		methods: []string{"MarshalJSON", "UnmarshalJSON", "MarshalBinary", "UnmarshalBinary", "MarshalText", "UnmarshalText"},
	},
	{
		concept: ontology.ConFileIO,
		// os is far too ubiquitous for an import signal; the selectors are the
		// filesystem verbs themselves.
		selectors: []string{
			"filepath.Walk", "filepath.WalkDir",
			"io.Copy", "io.ReadAll",
			"os.Create", "os.MkdirAll", "os.Open", "os.OpenFile",
			"os.ReadDir", "os.ReadFile", "os.Remove", "os.RemoveAll", "os.WriteFile",
		},
	},
	{
		concept: ontology.ConLogging,
		// Package-level stdlib loggers by selector, wrapper loggers by the
		// conventional receiver names. Never an import-substring "log" — that
		// fragment is inside "dialog", "catalog" and half the module paths on
		// earth, and imports are file-level anyway.
		selectors: []string{
			"log.Fatal", "log.Fatalf", "log.Fatalln",
			"log.Panic", "log.Panicf", "log.Panicln",
			"log.Print", "log.Printf", "log.Println",
			"slog.Debug", "slog.Error", "slog.Info", "slog.Warn", "slog.With",
		},
		receivers: []string{"logger", "logrus", "zap"},
	},
}

// Tag returns the seed labels the unit's AST signals match.
//
// This is a proposal, not a verdict: internal/lexicon uses the union of what
// each label matched as a founding member set and learns the concept from
// there, so a rule firing does not put its name on a function or claim any
// particular confidence. Labels are returned in rule declaration order; one
// matching channel is enough.
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

// Concepts returns the seed vocabulary: every concept the rules can propose, in
// declaration order.
//
// It exists because the rules no longer decide anything on their own. The
// lexicon grows a concept from each seed's members, and a seed that grows
// nothing is a fact worth reporting — "this corpus has no HTTP practice" is the
// answer to a real question. Reporting it needs the full list of seeds, which
// only this package has.
func Concepts() []string {
	out := make([]string, len(patternRules))
	for i, r := range patternRules {
		out[i] = string(r.concept)
	}
	return out
}
