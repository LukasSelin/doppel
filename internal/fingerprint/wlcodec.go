// This file's encoders and Snapshot are the two halves of one design, so the
// rationale lives here rather than split across packages.
//
// A Weisfeiler-Lehman label is an FNV-1a hash: essentially a uniformly
// random 64-bit value, with no structure a general-purpose compressor or a
// naive delta encoding can exploit. Measured on the public moby corpus
// (7,644 functions, ~85 labels/function): delta-uvarint over one function's
// own sorted raw labels costs ~9.5 bytes/entry, because the gap between two
// uniformly random 64-bit values drawn from the same function's small bag is
// itself close to the full 64-bit range — sorting does not cluster values
// that were never clustered. Storing every unit's bag this way would have
// put the moby --format json payload at just over 5x its schema-5 size,
// over the budget this feature is measured against.
//
// EncodeWLBagIndexed is the fix, and it is not a cleverer bit-packing of the
// same data — it exploits a fact about *this corpus* rather than about
// 64-bit integers: the same handful of structural labels recur across many
// functions (that recurrence is the entire premise of the corpus IDF
// weighting elsewhere in this package), so the set of *distinct* labels
// across a whole run is far smaller than the sum of every unit's bag size —
// 131,350 distinct labels behind 648,305 total entries on moby, a 4.9x
// reduction before a single byte is counted. internal/snapshot.Build
// collects that distinct set once per run (Snapshot.Labels, via
// EncodeLabelDict — still delta-uvarint over raw hashes, since a dictionary
// entry is the one thing in this design that cannot avoid paying full
// 64-bit-domain cost, but it pays it exactly once instead of once per
// occurrence) and every Unit.WL then stores small integers — positions in
// that shared dictionary — which delta-encode far more tightly than raw
// hashes because the domain they range over is the dictionary's size, not
// 2^64. Measured end to end on moby: schema-6 --format json is 5,877,513
// bytes against a schema-5 baseline of 2,077,641 — 2.83x, inside the <3x
// budget the round-trip feature is measured against.

package fingerprint

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"slices"
)

// wlCodecEncoding is the base64 alphabet every encoder in this file uses to
// turn a byte string into one JSON string field.
//
// Raw (unpadded) standard encoding, chosen once and fixed forever: none of
// these strings travel in a URL or a filename, so the URL-safe alphabet buys
// nothing, and the padding characters RawStdEncoding drops are pure bytes a
// snapshot never needs.
var wlCodecEncoding = base64.RawStdEncoding

// EncodeWLBag serializes a Weisfeiler-Lehman label bag (see WLBag) — sorted
// ascending by Label, no label repeated — into one compact string: the
// sorted labels delta-uvarint, interleaved with a plain uvarint of each
// label's count, then base64.
//
// It is a general-purpose, dictionary-free primitive — useful standalone,
// and what this package's own tests round-trip against real corpus bags —
// but it is not what internal/snapshot stores a run's units under. Delta
// encoding raw 64-bit hashes only pays off when the values happen to
// cluster, which a single function's own bag does not (see the package
// doc): EncodeWLBagIndexed is the corpus-scale form, and the one a run's
// Snapshot actually uses.
//
// An empty or nil bag encodes to "", which DecodeWLBag reads back as nil —
// the same "no body" convention Fingerprint's zero value and Digest's empty
// string already use.
func EncodeWLBag(bag []LabelCount) string {
	if len(bag) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(bag)*4)
	var tmp [binary.MaxVarintLen64]byte
	var prev uint64
	for _, lc := range bag {
		n := binary.PutUvarint(tmp[:], lc.Label-prev)
		buf = append(buf, tmp[:n]...)
		n = binary.PutUvarint(tmp[:], uint64(lc.Count))
		buf = append(buf, tmp[:n]...)
		prev = lc.Label
	}
	return wlCodecEncoding.EncodeToString(buf)
}

// DecodeWLBag is EncodeWLBag's inverse: the exact []LabelCount WLBag built,
// same labels, same counts, same ascending order. "" decodes to nil.
//
// It reconstructs each label as the running sum of deltas mod 2^64, which is
// exactly what Go's unsigned subtraction and addition already do — so this
// is correct even over a bag EncodeWLBag was not asked to sort first,
// though every caller in this codebase only ever hands it WLBag's own
// output.
func DecodeWLBag(s string) ([]LabelCount, error) {
	if s == "" {
		return nil, nil
	}
	buf, err := wlCodecEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("fingerprint: decode WL bag: %w", err)
	}
	var out []LabelCount
	var prev uint64
	for len(buf) > 0 {
		delta, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("fingerprint: decode WL bag: malformed label varint")
		}
		buf = buf[n:]
		count, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("fingerprint: decode WL bag: malformed count varint")
		}
		buf = buf[n:]
		label := prev + delta
		out = append(out, LabelCount{Label: label, Count: int(count)})
		prev = label
	}
	return out, nil
}

// EncodeLabelDict serializes a sorted, deduped set of Weisfeiler-Lehman
// labels — the corpus-wide dictionary a Snapshot stores once per run, never
// once per unit — as delta-uvarint over the sorted raw values, then base64.
// The wire shape is identical to EncodeWLBag's; what differs is how often a
// caller pays for it. "" for an empty dictionary, decoded back to nil.
func EncodeLabelDict(labels []uint64) string {
	if len(labels) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(labels)*7)
	var tmp [binary.MaxVarintLen64]byte
	var prev uint64
	for _, label := range labels {
		n := binary.PutUvarint(tmp[:], label-prev)
		buf = append(buf, tmp[:n]...)
		prev = label
	}
	return wlCodecEncoding.EncodeToString(buf)
}

// DecodeLabelDict is EncodeLabelDict's inverse: the exact sorted []uint64
// it was built from. "" decodes to nil.
func DecodeLabelDict(s string) ([]uint64, error) {
	if s == "" {
		return nil, nil
	}
	buf, err := wlCodecEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("fingerprint: decode label dictionary: %w", err)
	}
	var out []uint64
	var prev uint64
	for len(buf) > 0 {
		delta, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("fingerprint: decode label dictionary: malformed varint")
		}
		buf = buf[n:]
		label := prev + delta
		out = append(out, label)
		prev = label
	}
	return out, nil
}

// EncodeWLBagIndexed serializes a bag against a corpus-wide label dictionary
// (see EncodeLabelDict): dict must be sorted ascending and, in every actual
// caller, is built as the union of every unit's own bag before any of them
// is encoded — internal/snapshot.Build's contract, which is what guarantees
// dict contains every label a bag can carry. Each label becomes its
// position in dict rather than its raw value, and positions delta-encode far
// more tightly than raw hashes: they range over len(dict), not 2^64 (see the
// package doc for the measurement that motivates this over EncodeWLBag).
//
// A label absent from dict is silently dropped rather than erroring. That
// case does not arise from the one caller this package has — dict is always
// built from exactly these bags — and a codec persisting a corpus statistic
// should not be able to fail an entire run's snapshot over one entry;
// DecodeWLBagIndexed is what still catches a genuine mismatch, by refusing
// an out-of-range index rather than guessing.
func EncodeWLBagIndexed(bag []LabelCount, dict []uint64) string {
	if len(bag) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(bag)*3)
	var tmp [binary.MaxVarintLen64]byte
	var prev uint64
	any := false
	for _, lc := range bag {
		idx, found := slices.BinarySearch(dict, lc.Label)
		if !found {
			continue
		}
		pos := uint64(idx)
		n := binary.PutUvarint(tmp[:], pos-prev)
		buf = append(buf, tmp[:n]...)
		n = binary.PutUvarint(tmp[:], uint64(lc.Count))
		buf = append(buf, tmp[:n]...)
		prev = pos
		any = true
	}
	if !any {
		return ""
	}
	return wlCodecEncoding.EncodeToString(buf)
}

// DecodeWLBagIndexed is EncodeWLBagIndexed's inverse. dict must be the exact
// dictionary the bag was encoded against — in practice, Snapshot.Labels
// decoded once via DecodeLabelDict and reused across every unit, since
// re-decoding it per unit would undo the whole point of storing it once. An
// index at or past len(dict) means the string was not produced against this
// dictionary — a schema, build or file mismatch upstream of this call — and
// is reported as an error rather than silently returning a wrong label.
func DecodeWLBagIndexed(s string, dict []uint64) ([]LabelCount, error) {
	if s == "" {
		return nil, nil
	}
	buf, err := wlCodecEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("fingerprint: decode indexed WL bag: %w", err)
	}
	var out []LabelCount
	var prev uint64
	for len(buf) > 0 {
		delta, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("fingerprint: decode indexed WL bag: malformed index varint")
		}
		buf = buf[n:]
		count, n := binary.Uvarint(buf)
		if n <= 0 {
			return nil, fmt.Errorf("fingerprint: decode indexed WL bag: malformed count varint")
		}
		buf = buf[n:]
		pos := prev + delta
		if pos >= uint64(len(dict)) {
			return nil, fmt.Errorf("fingerprint: decode indexed WL bag: index %d outside dictionary of %d labels", pos, len(dict))
		}
		out = append(out, LabelCount{Label: dict[pos], Count: int(count)})
		prev = pos
	}
	return out, nil
}

// WLOverlap exposes wlOverlap to a caller that has only bags and a LabelIDF,
// not a full Fingerprint — the position a Snapshot consumer is in, since it
// stores the bag and not Flow/Depth/Types. It is the same math Similarity's
// 0.60 component runs, so recomputing through it (rather than through a
// hand-built partial Fingerprint) is what lets a round trip claim
// bit-identical results rather than merely "close": jaccard is the WL
// component of Breakdown, containment is Breakdown.Containment.
func WLOverlap(a, b []LabelCount, idf *LabelIDF) (jaccard, containment float64) {
	return wlOverlap(a, b, idf)
}
