package ontology

// Concept terms: what a function is doing, as intent rather than shape.
//
// The nine leaves are exactly the tags tagger.Tag emits, with byte-identical
// IDs, so introducing the taxonomy changes no output. Everything above them is
// new and abstract, and is the entire point: without interior nodes http_call
// and db_access compare exactly equal to two tags with nothing in common, and a
// real near-miss is invisible.
//
//	concept
//	├── io_operation
//	│   ├── remote_io
//	│   │   └── http_call
//	│   └── data_store_access
//	│       ├── db_access
//	│       ├── caching
//	│       └── transaction
//	├── data_transformation
//	│   ├── mapping
//	│   └── validation
//	├── control_flow
//	│   ├── concurrency
//	│   └── fault_tolerance
//	│       └── retry
//	└── error_handling
//	    └── error_wrapping
//
// remote_io and fault_tolerance are deliberately unary. A node with one child
// adds no discriminative power and costs its leaf depth, which under Wu-Palmer
// lowers that leaf relatedness to everything: http_call to db_access is 0.33
// with remote_io in place and would be 0.40 without it. They are kept because
// they name the slot a future sibling goes in (grpc_call, circuit_breaker) and
// because the pinned relatedness values below are documented against them.
// Removing them is a scoring change, not a simplification.
//
// Relatedness for every pair of leaves, as a review aid. The shape of the tree
// is silently a weight table, so the numbers should be readable here without
// having to derive them:
//
//	http_call   db_access    0.33     db_access   caching      0.67
//	http_call   caching      0.33     db_access   transaction  0.67
//	http_call   transaction  0.33     caching     transaction  0.67
//	mapping     validation   0.50     concurrency retry        0.40
//	every other leaf pair     0.00    (least common ancestor is the root)
const (
	ConConcept            TermID = "concept"
	ConIOOperation        TermID = "io_operation"
	ConRemoteIO           TermID = "remote_io"
	ConHTTPCall           TermID = "http_call"
	ConDataStoreAccess    TermID = "data_store_access"
	ConDBAccess           TermID = "db_access"
	ConCaching            TermID = "caching"
	ConTransaction        TermID = "transaction"
	ConDataTransformation TermID = "data_transformation"
	ConMapping            TermID = "mapping"
	ConValidation         TermID = "validation"
	ConControlFlow        TermID = "control_flow"
	ConConcurrency        TermID = "concurrency"
	ConFaultTolerance     TermID = "fault_tolerance"
	ConRetry              TermID = "retry"
	ConErrorHandling      TermID = "error_handling"
	ConErrorWrapping      TermID = "error_wrapping"
)

var conceptTerms = []Term{
	{ID: ConConcept, Kind: KindConcept, Abstract: true,
		Label: "Concept", Def: "A kind of work a function performs."},

	{ID: ConIOOperation, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "I/O operation", Def: "Work that crosses the process boundary."},
	{ID: ConRemoteIO, Kind: KindConcept, Parent: ConIOOperation, Abstract: true,
		Label: "Remote I/O", Def: "I/O against a service reached over the network."},
	{ID: ConHTTPCall, Kind: KindConcept, Parent: ConRemoteIO,
		Label: "HTTP call", Def: "Issues an outbound HTTP request."},
	{ID: ConDataStoreAccess, Kind: KindConcept, Parent: ConIOOperation, Abstract: true,
		Label: "Data store access", Def: "I/O against something that holds state."},
	{ID: ConDBAccess, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Database access", Def: "Reads or writes a database."},
	{ID: ConCaching, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Caching", Def: "Reads or writes a cache."},
	{ID: ConTransaction, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Transaction", Def: "Delimits a unit of work that commits or rolls back."},

	{ID: ConDataTransformation, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Data transformation", Def: "Work that reshapes or inspects values in memory."},
	{ID: ConMapping, Kind: KindConcept, Parent: ConDataTransformation,
		Label: "Mapping", Def: "Converts a value from one representation to another."},
	{ID: ConValidation, Kind: KindConcept, Parent: ConDataTransformation,
		Label: "Validation", Def: "Checks a value against expectations."},

	{ID: ConControlFlow, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Control flow", Def: "Work that governs when and how other work runs."},
	{ID: ConConcurrency, Kind: KindConcept, Parent: ConControlFlow,
		Label: "Concurrency", Def: "Coordinates work across goroutines."},
	{ID: ConFaultTolerance, Kind: KindConcept, Parent: ConControlFlow, Abstract: true,
		Label: "Fault tolerance", Def: "Control flow that exists to survive failure."},
	{ID: ConRetry, Kind: KindConcept, Parent: ConFaultTolerance,
		Label: "Retry", Def: "Re-attempts an operation after failure."},

	{ID: ConErrorHandling, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Error handling", Def: "Work that carries or enriches failure information."},
	{ID: ConErrorWrapping, Kind: KindConcept, Parent: ConErrorHandling,
		Label: "Error wrapping", Def: "Adds context to an error before returning it."},
}
