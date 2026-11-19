package lexfront

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/syntax"
)

func funcNamed(t *testing.T, f *syntax.File, name string) syntax.Func {
	t.Helper()
	for _, fn := range f.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("no function %q in %v", name, names(f))
	return syntax.Func{}
}

// The lexical frontend reads no types, ever, so every function it produces
// says so. Scoring two of its empty type sets as agreement was worth 0.15 of
// code-shape on every pair, the larger part of the 0.40 floor this flag
// removes.
func TestEveryLexicalFunctionIsUntyped(t *testing.T) {
	src := "function add(a, b) {\n  return a + b;\n}\nfunction none() {\n  return 1;\n}\n"
	f, _ := Parse(specFor(t, "javascript"), "a.js", []byte(src))
	for _, fn := range f.Funcs {
		if !fn.Untyped {
			t.Errorf("%s: Untyped = false; the lexical frontend reads no types", fn.Name)
		}
	}
}

// Nesting is lost exactly where a control-flow body is not brace-delimited:
// a Python suite, or a braceless C-family statement. A switch arm owns no
// body in any grammar, so an arm does not count; and a body with no control
// flow at all has nothing to lose.
func TestUnnestedMarksLostBodies(t *testing.T) {
	py := `
def guarded(x):
    if x:
        return 1
    return 0

def straight(x):
    y = x + 1
    return y
`
	f, _ := Parse(specFor(t, "python"), "a.py", []byte(py))
	if !funcNamed(t, f, "guarded").Unnested {
		t.Errorf("python if suite: Unnested = false, want true")
	}
	if funcNamed(t, f, "straight").Unnested {
		t.Errorf("python body with no control flow: Unnested = true, want false")
	}

	js := `
function braced(x) {
  if (x) {
    return 1;
  }
  for (let i = 0; i < 3; i++) {
    x++;
  }
  return 0;
}

function braceless(x) {
  if (x) return 1;
  return 0;
}

function arms(x) {
  switch (x) {
    case 1:
      return "one";
    default:
      return "many";
  }
}
`
	g, _ := Parse(specFor(t, "javascript"), "a.js", []byte(js))
	if funcNamed(t, g, "braced").Unnested {
		t.Errorf("braced if/for: Unnested = true, want false")
	}
	if !funcNamed(t, g, "braceless").Unnested {
		t.Errorf("braceless if: Unnested = false, want true")
	}
	if funcNamed(t, g, "arms").Unnested {
		t.Errorf("switch with case arms: Unnested = true, want false")
	}
}
