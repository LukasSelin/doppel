package parser

import (
	"github.com/LukasSelin/doppel/internal/lexfront"
	"github.com/LukasSelin/doppel/internal/syntax"
)

func init() {
	for _, sp := range lexfront.Specs {
		register(lexFrontend{sp})
	}
}

// lexFrontend adapts one language-table entry of internal/lexfront to the
// Frontend interface. There is one instance per language rather than a single
// "lexical" frontend claiming everything, and that is load-bearing: Lang() is
// what the cross-language rule compares, so a Python function and a
// TypeScript one must not share a tag or they would become merge candidates
// for each other.
type lexFrontend struct{ sp lexfront.Spec }

func (f lexFrontend) Lang() string { return f.sp.Lang }

func (f lexFrontend) Extensions() []string { return f.sp.Exts }

func (f lexFrontend) Parse(path string, src []byte) (*syntax.File, error) {
	return lexfront.Parse(f.sp, path, src)
}

func (f lexFrontend) IsTestFile(path string) bool { return f.sp.IsTestFile(path) }
