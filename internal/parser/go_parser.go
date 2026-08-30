package parser

import (
	"strings"

	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/syntax"
)

func init() { register(goFrontend{}) }

// goFrontend adapts internal/gofront to the Frontend interface. The adapter
// lives here rather than in gofront because the interface belongs to the
// registry's owner, and gofront must stay free of any dependency on this
// package — a frontend depending on the thing that dispatches to it is the
// cycle this split exists to avoid.
type goFrontend struct{}

func (goFrontend) Lang() string { return gofront.Lang }

func (goFrontend) Extensions() []string { return []string{".go"} }

func (goFrontend) Parse(path string, src []byte) (*syntax.File, error) {
	return gofront.Parse(path, src)
}

// IsTestFile keys on Go's compiler-recognized suffix, not a naming heuristic:
// the toolchain itself treats _test.go as a separate build unit.
func (goFrontend) IsTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}
