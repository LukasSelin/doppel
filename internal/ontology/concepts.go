package ontology

// Concept terms: what a function is doing, as intent rather than shape.
//
// The fourteen leaves are exactly the tags tagger.Tag emits, with byte-identical
// IDs. The original nine were introduced so the taxonomy changed no output;
// the five added later (grpc_call, circuit_breaker, serialization, file_io,
// logging) filled the two reserved unary slots and widened the coverage —
// every unmatched intent scores zero, so more leaves means more genuinely
// related pairs land above zero. Everything above the leaves is abstract, and
// is the entire point: without interior nodes http_call and db_access compare
// exactly equal to two tags with nothing in common, and a real near-miss is
// invisible.
//
//	concept
//	├── io_operation
//	│   ├── remote_io
//	│   │   ├── http_call
//	│   │   └── grpc_call
//	│   ├── data_store_access
//	│   │   ├── db_access
//	│   │   ├── caching
//	│   │   └── transaction
//	│   ├── file_io
//	│   └── logging
//	├── data_transformation
//	│   ├── mapping
//	│   ├── validation
//	│   └── serialization
//	├── control_flow
//	│   ├── concurrency
//	│   └── fault_tolerance
//	│       ├── retry
//	│       └── circuit_breaker
//	└── error_handling
//	    └── error_wrapping
//
// remote_io and fault_tolerance were deliberately unary for a while — kept as
// the named slot a future sibling goes in, at the cost of a depth level that
// lowers their leaf's relatedness to everything (http_call to db_access is
// 0.33 with remote_io in place and would be 0.40 without it). grpc_call and
// circuit_breaker are those siblings, arrived. file_io and logging sit
// directly under io_operation: neither is remote and neither is a data store,
// and inventing an interior node for each would repeat the unary-slot cost
// with no future sibling in sight.
//
// Relatedness for every non-zero pair of leaves, as a review aid. The shape of
// the tree is silently a weight table, so the numbers should be readable here
// without having to derive them:
//
//	http_call   grpc_call    0.67     db_access   caching      0.67
//	retry       circuit_brk  0.67     db_access   transaction  0.67
//	http/grpc   db/cach/tx   0.33     caching     transaction  0.67
//	http/grpc   file_io|log  0.40     db/cach/tx  file_io|log  0.40
//	file_io     logging      0.50     mapping     validation   0.50
//	mapping     serialization 0.50    validation  serialization 0.50
//	concurrency retry|brk    0.40
//	every other leaf pair     0.00    (least common ancestor is the root)
const (
	ConConcept            TermID = "concept"
	ConIOOperation        TermID = "io_operation"
	ConRemoteIO           TermID = "remote_io"
	ConHTTPCall           TermID = "http_call"
	ConGRPCCall           TermID = "grpc_call"
	ConDataStoreAccess    TermID = "data_store_access"
	ConDBAccess           TermID = "db_access"
	ConCaching            TermID = "caching"
	ConTransaction        TermID = "transaction"
	ConFileIO             TermID = "file_io"
	ConLogging            TermID = "logging"
	ConDataTransformation TermID = "data_transformation"
	ConMapping            TermID = "mapping"
	ConValidation         TermID = "validation"
	ConSerialization      TermID = "serialization"
	ConControlFlow        TermID = "control_flow"
	ConConcurrency        TermID = "concurrency"
	ConFaultTolerance     TermID = "fault_tolerance"
	ConRetry              TermID = "retry"
	ConCircuitBreaker     TermID = "circuit_breaker"
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
	{ID: ConGRPCCall, Kind: KindConcept, Parent: ConRemoteIO,
		Label: "gRPC call", Def: "Issues an outbound gRPC request."},
	{ID: ConDataStoreAccess, Kind: KindConcept, Parent: ConIOOperation, Abstract: true,
		Label: "Data store access", Def: "I/O against something that holds state."},
	{ID: ConDBAccess, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Database access", Def: "Reads or writes a database."},
	{ID: ConCaching, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Caching", Def: "Reads or writes a cache."},
	{ID: ConTransaction, Kind: KindConcept, Parent: ConDataStoreAccess,
		Label: "Transaction", Def: "Delimits a unit of work that commits or rolls back."},
	{ID: ConFileIO, Kind: KindConcept, Parent: ConIOOperation,
		Label: "File I/O", Def: "Reads, writes or traverses the filesystem."},
	{ID: ConLogging, Kind: KindConcept, Parent: ConIOOperation,
		Label: "Logging", Def: "Emits diagnostic output to a log sink."},

	{ID: ConDataTransformation, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Data transformation", Def: "Work that reshapes or inspects values in memory."},
	{ID: ConMapping, Kind: KindConcept, Parent: ConDataTransformation,
		Label: "Mapping", Def: "Converts a value from one representation to another."},
	{ID: ConValidation, Kind: KindConcept, Parent: ConDataTransformation,
		Label: "Validation", Def: "Checks a value against expectations."},
	{ID: ConSerialization, Kind: KindConcept, Parent: ConDataTransformation,
		Label: "Serialization", Def: "Encodes or decodes a value against a wire or storage format."},

	{ID: ConControlFlow, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Control flow", Def: "Work that governs when and how other work runs."},
	{ID: ConConcurrency, Kind: KindConcept, Parent: ConControlFlow,
		Label: "Concurrency", Def: "Coordinates work across goroutines."},
	{ID: ConFaultTolerance, Kind: KindConcept, Parent: ConControlFlow, Abstract: true,
		Label: "Fault tolerance", Def: "Control flow that exists to survive failure."},
	{ID: ConRetry, Kind: KindConcept, Parent: ConFaultTolerance,
		Label: "Retry", Def: "Re-attempts an operation after failure."},
	{ID: ConCircuitBreaker, Kind: KindConcept, Parent: ConFaultTolerance,
		Label: "Circuit breaker", Def: "Stops issuing an operation while it keeps failing."},

	{ID: ConErrorHandling, Kind: KindConcept, Parent: ConConcept, Abstract: true,
		Label: "Error handling", Def: "Work that carries or enriches failure information."},
	{ID: ConErrorWrapping, Kind: KindConcept, Parent: ConErrorHandling,
		Label: "Error wrapping", Def: "Adds context to an error before returning it."},
}
