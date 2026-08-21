// Package bench is the measurement harness: a golden-ranking scorer, a ladder
// of pinned public Go corpora, per-stage performance benchmarks over them, and
// the generator behind the reports in examples/.
//
// # Golden ranking
//
// A labels file encodes a human review of a report as pair verdicts:
//
//	{
//	  "corpus": "<freeform name chosen by the label author>",
//	  "reviewed": "2026-08-21",
//	  "population": "include" | "exclude" | "only",   // which --tests view the labels describe; default include
//	  "labels": [
//	    {"a": "pkg.FuncA", "b": "pkg.*Recv.FuncB",
//	     "class": "merge" | "refactor" | "false_positive",
//	     "note": "short rationale"}
//	  ]
//	}
//
// Pair identity is the unordered qualified-name pair, names rendered exactly
// as the reporter shows them (Package + "." + Name, receiver stars kept).
// scoreLabels runs the pipeline's ranking-relevant stages as a library over
// the declared population (cross test/prod pairs are dropped like the pipeline
// does), ranks with the production defaults, and scores the labeled pairs:
// each gets a rank or an absence reason, and three hard assertions check that
// merge findings are retrieved, that no false positive outranks a merge, and
// that no false positive sits in the top 20. Everything else is a logged
// scorecard. A partial review is fine — only labeled pairs are scored, so a
// pair whose verdict is genuinely contested is better left out than guessed.
//
// Two tests call it:
//
//   - TestGoldenCorpora scores the committed reviews in examples/labels/
//     against the matching rung of the public ladder. Public, pinned, and
//     checkable by any reader; skipped for corpora that are not fetched.
//
//   - TestGoldenRanking scores a PRIVATE review. Both inputs arrive by
//     environment variable, so no reference to that corpus ever lives in this
//     repository — no names, no paths, no labeled pairs:
//
//     DOPPEL_BENCH_CORPUS  absolute path of the Go tree to analyze
//     DOPPEL_BENCH_LABELS  absolute path of a human-reviewed labels JSON file
//
//     DOPPEL_BENCH_CORPUS=/path/to/corpus DOPPEL_BENCH_LABELS=/path/to/labels.json \
//     go test ./internal/bench/ -v -run TestGoldenRanking
//
// # The corpus ladder
//
// Corpora in corpora.go pins seven public Go repositories at release tags,
// ordered old-and-complex to new-and-narrow. Only coordinates are committed;
// Fetch shallow-clones a rung into Root() ($DOPPEL_CORPORA, else a user-cache
// directory) and verifies the checked-out commit against the manifest. The
// trees themselves never enter the repository and are never analyzed by
// `doppel analyze .` on this one, because Root() is outside the working tree.
//
//	task corpora    # fetch (network, a few hundred MB)
//	task bench      # per-stage benchmarks over whatever is fetched
//	task golden     # score the committed labels
//	task examples   # regenerate examples/
package bench
