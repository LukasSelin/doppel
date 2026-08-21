// Package bench is a corpus-agnostic golden-ranking benchmark harness.
//
// It carries no data: both inputs arrive via environment variables, so no
// reference to any particular corpus ever lives in this repository.
//
//	DOPPEL_BENCH_CORPUS  absolute path of the Go tree to analyze
//	DOPPEL_BENCH_LABELS  absolute path of a human-reviewed labels JSON file
//
// The labels file encodes a human review of a report as pair verdicts:
//
//	{
//	  "corpus": "<freeform name chosen by the label author>",
//	  "reviewed": "2026-08-21",
//	  "labels": [
//	    {"a": "pkg.FuncA", "b": "pkg.*Recv.FuncB",
//	     "class": "merge" | "refactor" | "false_positive",
//	     "note": "short rationale"}
//	  ]
//	}
//
// Pair identity is the unordered qualified-name pair, names rendered exactly
// as the reporter shows them (Package + "." + Name, receiver stars kept).
// The harness runs the pipeline's ranking-relevant stages as a library over
// the FULL population (equivalent to --tests include, since labels may span
// test and production pairs; cross test/prod pairs are dropped like the
// pipeline does), ranks with the production defaults, and scores the
// labeled pairs: each
// gets a rank or an absence reason, and three hard assertions check that
// merge findings are retrieved, that no false positive outranks a merge, and
// that no false positive sits in the top 20. Everything else is a logged
// scorecard.
//
// Run:
//
//	DOPPEL_BENCH_CORPUS=/path/to/corpus DOPPEL_BENCH_LABELS=/path/to/labels.json \
//	  go test ./internal/bench/ -v -run TestGoldenRanking
package bench
