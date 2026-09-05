package retriever

import "sync"

// pairMemo memoizes a deterministic function of an unordered pair, safely for
// concurrent use.
//
// Retrieval computes two such functions — the exact fingerprint Breakdown and
// the shared concept information — and computes each pair twice by
// construction: a pair (a,b) is reached from a's admission turn and again from
// b's, and once more when the union is evaluated. A plain map halved that work
// and made the loops unparallelisable; this keeps the halving and allows the
// fan-out.
//
// # Why sharding and not one lock
//
// The memo is hit once per candidate neighbour, millions of times on a large
// corpus, so a single mutex would serialise exactly the loops being spread out.
// Sharding by the key means workers on unrelated pairs never touch the same
// lock.
//
// # Why a racing compute is fine
//
// Two workers can miss the same key and both compute it. The stored value is
// the same either way — every function memoized here is pure in the pair and
// the corpus statistics, both fixed by this point — so the duplicate costs one
// wasted computation and nothing else. The alternative, holding the shard lock
// across the computation, would serialise the expensive part rather than the
// lookup. Nothing iterates a memo, so map order never reaches an output.
type pairMemo[V any] struct {
	shards [memoShards]memoShard[V]
}

// memoShards is a power of two so the index is a mask rather than a division.
// 64 is well above the worker count on any machine this runs on, which is what
// keeps contention negligible without making the empty memo large.
const memoShards = 64

type memoShard[V any] struct {
	mu sync.RWMutex
	m  map[pairKey]V
	// Pad the shard out past a cache line so two shards' mutexes never share
	// one, which would reintroduce the contention the sharding removes.
	_ [40]byte
}

func newPairMemo[V any]() *pairMemo[V] {
	p := &pairMemo[V]{}
	for i := range p.shards {
		p.shards[i].m = make(map[pairKey]V)
	}
	return p
}

// get returns the memoized value for k, calling compute if it is absent.
func (p *pairMemo[V]) get(k pairKey, compute func() V) V {
	s := &p.shards[(uint(k[0])*31+uint(k[1]))&(memoShards-1)]

	s.mu.RLock()
	v, ok := s.m[k]
	s.mu.RUnlock()
	if ok {
		return v
	}

	v = compute()

	s.mu.Lock()
	// Another worker may have stored it while this one computed. Keep the
	// stored value rather than overwriting, so every caller of a given key sees
	// one identical value even in the presence of a future non-pure computer.
	if prev, ok := s.m[k]; ok {
		s.mu.Unlock()
		return prev
	}
	s.m[k] = v
	s.mu.Unlock()
	return v
}
