package ontology

import "strings"

// Entity terms: the kinds of thing Doppel talks about. Only callables are
// scored; package, receiver_type and visibility exist because relations point
// at them and every relation's Domain and Range must resolve (axiom 5).
const (
	EntEntity       TermID = "entity"
	EntCallable     TermID = "callable"
	EntFunction     TermID = "function"
	EntMethod       TermID = "method"
	EntPackage      TermID = "package"
	EntReceiverType TermID = "receiver_type"
	EntVisibility   TermID = "visibility"
)

var entityTerms = []Term{
	{ID: EntEntity, Kind: KindEntity, Abstract: true,
		Label: "Entity", Def: "Anything the analysis can name."},
	{ID: EntCallable, Kind: KindEntity, Parent: EntEntity, Abstract: true,
		Label: "Callable", Def: "A unit of code that can be invoked and therefore compared."},
	{ID: EntFunction, Kind: KindEntity, Parent: EntCallable,
		Label: "Function", Def: "A callable with no receiver."},
	{ID: EntMethod, Kind: KindEntity, Parent: EntCallable,
		Label: "Method", Def: "A callable bound to a receiver type."},
	{ID: EntPackage, Kind: KindEntity, Parent: EntEntity,
		Label: "Package", Def: "A Go package, used as a locality signal."},
	{ID: EntReceiverType, Kind: KindEntity, Parent: EntEntity,
		Label: "Receiver type", Def: "The type a method is bound to, with pointer-ness normalized away."},
	{ID: EntVisibility, Kind: KindEntity, Parent: EntEntity,
		Label: "Visibility", Def: "Whether a callable is exported from its package."},
}

// EntityKindOf classifies a callable from its receiver type.
func EntityKindOf(receiverType string) TermID {
	if NormalizeReceiver(receiverType) == "" {
		return EntFunction
	}
	return EntMethod
}

// NormalizeReceiver strips the pointer star. The parser names methods
// "*Server.Start", keeping the star, so a value-receiver and a pointer-receiver
// method on the same type arrive here as two different strings. They are bound
// to the same type and must compare equal.
func NormalizeReceiver(receiverType string) string {
	return strings.TrimPrefix(receiverType, "*")
}

// ReceiverRelatedness grades how much two callables agree on what they are
// bound to.
//
//	1.0  same normalized receiver, which also covers two plain functions
//	0.5  both methods, but on different types
//	0.0  one plain function and one method
//
// Two plain functions scoring 1.0 preserves a documented floor effect rather
// than quietly retuning it: any two of them start with that signal's weight for
// free. Lowering it is a deliberate retune, not part of introducing grading.
func ReceiverRelatedness(a, b string) float64 {
	na, nb := NormalizeReceiver(a), NormalizeReceiver(b)
	switch {
	case na == nb:
		return 1.0
	case na != "" && nb != "":
		return 0.5
	default:
		return 0.0
	}
}
