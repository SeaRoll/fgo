package fgo

import (
	"cmp"
	"iter"
	"slices"
)

// Stream is a lazy pipeline over a sequence of values. Intermediate operations
// such as Map, Filter, Take and DistinctBy only describe work: nothing is read
// from the source until a terminal operation such as Collect, Reduce, Find or
// Count runs. A terminal operation that can answer early stops reading the
// source at that point.
//
// Whether a Stream can be consumed more than once depends on its source and is
// not visible from the type. A Stream from ToStream is backed by a slice and can
// be consumed repeatedly; a Stream from FromSeq inherits the behaviour of the
// sequence it was given, so a channel- or generator-backed Stream yields nothing
// on a second pass. SortFunc, SortBy and Sorted always return a slice-backed,
// repeatable Stream.
type Stream[T any] struct {
	seq iter.Seq[T]
}

// ToStream returns a Stream over the elements of items.
//
// The Stream is backed by the slice rather than a copy, so it can be consumed
// repeatedly, and mutating items afterwards affects later iterations.
func ToStream[T any](items []T) *Stream[T] {
	return &Stream[T]{seq: slices.Values(items)}
}

// FromSeq returns a Stream over seq. It is the entry point for sequences from
// elsewhere, such as maps.Keys, maps.Values, or a hand-written generator.
//
// A nil seq yields an empty Stream rather than a Stream that panics on first
// use. The result inherits seq's reuse behaviour, so a single-use sequence
// yields nothing on a second pass.
func FromSeq[T any](seq iter.Seq[T]) *Stream[T] {
	if seq == nil {
		return &Stream[T]{seq: func(func(T) bool) {}}
	}
	return &Stream[T]{seq: seq}
}

// Reduce folds every element into an accumulator, starting from initial, and
// returns the final accumulator. The accumulator type U may differ from the
// element type, so it can build a sum, a string, a map, or any other aggregate.
//
// It is a terminal operation and always consumes the whole Stream.
func (s *Stream[T]) Reduce[U any](initial U, reducer func(acc U, item T) U) U {
	accumulator := initial

	// Consume the iterator
	for v := range s.seq {
		accumulator = reducer(accumulator, v)
	}

	return accumulator
}

// Map returns a lazy Stream whose elements are transform applied to each element
// of s, allowing the element type to change from T to U.
//
// transform is called during iteration, once per element actually consumed, not
// when Map is called. Use TryMap on Result when the transform can fail.
func (s *Stream[T]) Map[U any](transform func(T) U) *Stream[U] {
	newSeq := func(yield func(U) bool) {
		for v := range s.seq {
			if !yield(transform(v)) {
				return
			}
		}
	}
	return &Stream[U]{seq: newSeq}
}

// FlatMap returns a lazy Stream concatenating the Streams that transform
// produces for each element, allowing the element type to change from T to U.
// Use it to expand one element into many, such as splitting lines into words.
//
// A nil or empty Stream from transform contributes nothing and is skipped.
func (s *Stream[T]) FlatMap[U any](transform func(T) *Stream[U]) *Stream[U] {
	newSeq := func(yield func(U) bool) {
		for v := range s.seq {
			inner := transform(v)
			if inner == nil || inner.seq == nil {
				continue
			}
			for u := range inner.seq {
				if !yield(u) {
					return
				}
			}
		}
	}
	return &Stream[U]{seq: newSeq}
}

// Filter returns a lazy Stream of the elements of s that satisfy keep.
func (s *Stream[T]) Filter(keep func(T) bool) *Stream[T] {
	newSeq := func(yield func(T) bool) {
		for v := range s.seq {
			if keep(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
	return &Stream[T]{seq: newSeq}
}

// Take returns a lazy Stream of at most the first n elements of s.
//
// It stops as soon as n elements have been yielded and never reads an n+1th
// element from the source, which matters when reading has a cost or a side
// effect. A non-positive n yields an empty Stream without touching the source.
//
// Take is also what makes an unbounded source usable, since it bounds an
// otherwise infinite sequence.
func (s *Stream[T]) Take(n int) *Stream[T] {
	newSeq := func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		remaining := n
		for v := range s.seq {
			if !yield(v) {
				return
			}
			remaining--
			if remaining == 0 {
				return
			}
		}
	}
	return &Stream[T]{seq: newSeq}
}

// DistinctBy returns a lazy Stream keeping only the first element for each
// distinct key, preserving the original order.
//
// The set of seen keys is rebuilt on each iteration, so a repeatable Stream
// gives the same result every time. Memory grows with the number of distinct
// keys, so it is not suitable for an unbounded Stream of unique keys.
func (s *Stream[T]) DistinctBy[K comparable](key func(T) K) *Stream[T] {
	newSeq := func(yield func(T) bool) {
		seen := make(map[K]struct{})
		for v := range s.seq {
			k := key(v)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			if !yield(v) {
				return
			}
		}
	}
	return &Stream[T]{seq: newSeq}
}

// Collect consumes s and returns its elements as a slice. An empty Stream
// returns a nil slice rather than an empty one.
//
// It is a terminal operation.
func (s *Stream[T]) Collect() []T {
	return slices.Collect(s.seq)
}

// Find returns the first element satisfying match, or an empty Option when none
// does. It stops reading the source at the first match.
//
// It is a terminal operation.
func (s *Stream[T]) Find(match func(T) bool) Option[T] {
	for v := range s.seq {
		if match(v) {
			return Some(v)
		}
	}
	return None[T]()
}

// First returns the first element of s, or an empty Option when s is empty. It
// reads at most one element from the source.
//
// It is a terminal operation.
func (s *Stream[T]) First() Option[T] {
	for v := range s.seq {
		return Some(v)
	}
	return None[T]()
}

// Any reports whether at least one element satisfies pred, stopping at the first
// one that does. It reports false for an empty Stream.
//
// It is a terminal operation.
func (s *Stream[T]) Any(pred func(T) bool) bool {
	for v := range s.seq {
		if pred(v) {
			return true
		}
	}
	return false
}

// All reports whether every element satisfies pred, stopping at the first one
// that does not. It reports true for an empty Stream, following the usual
// convention for a vacuous truth.
//
// It is a terminal operation.
func (s *Stream[T]) All(pred func(T) bool) bool {
	for v := range s.seq {
		if !pred(v) {
			return false
		}
	}
	return true
}

// Count consumes s and returns the number of elements.
//
// It is a terminal operation.
func (s *Stream[T]) Count() int {
	count := 0
	for range s.seq {
		count++
	}
	return count
}

// MinBy returns the element whose key is smallest, or an empty Option when s is
// empty. When several elements tie, the first is returned.
//
// Keys are compared with cmp.Less, which orders NaN below every other float
// rather than making every comparison false the way the < operator does.
//
// It is a terminal operation and consumes the whole Stream.
func (s *Stream[T]) MinBy[K cmp.Ordered](key func(T) K) Option[T] {
	var best T
	var bestKey K
	found := false
	for v := range s.seq {
		k := key(v)
		if !found || cmp.Less(k, bestKey) {
			best, bestKey, found = v, k, true
		}
	}
	if !found {
		return None[T]()
	}
	return Some(best)
}

// MaxBy returns the element whose key is largest, or an empty Option when s is
// empty. When several elements tie, the first is returned.
//
// Keys are compared with cmp.Less, so a NaN key never wins unless it is the only
// element.
//
// It is a terminal operation and consumes the whole Stream.
func (s *Stream[T]) MaxBy[K cmp.Ordered](key func(T) K) Option[T] {
	var best T
	var bestKey K
	found := false
	for v := range s.seq {
		k := key(v)
		if !found || cmp.Less(bestKey, k) {
			best, bestKey, found = v, k, true
		}
	}
	if !found {
		return None[T]()
	}
	return Some(best)
}

// GroupBy consumes s and returns its elements grouped by key. Within each group
// the original order is preserved, though Go map iteration order is not defined.
//
// It is a terminal operation. Use FromSeq with maps.Keys or maps.Values to carry
// the groups back into a Stream.
func (s *Stream[T]) GroupBy[K comparable](key func(T) K) map[K][]T {
	groups := make(map[K][]T)
	for v := range s.seq {
		k := key(v)
		groups[k] = append(groups[k], v)
	}
	return groups
}

// SortFunc returns a Stream of the elements of s ordered by compare, which
// reports whether a sorts before (negative), equal to (zero), or after
// (positive) b, matching the convention of the slices package.
//
// Sorting cannot be lazy: s is consumed immediately and the result is backed by
// a slice, so the returned Stream is repeatable regardless of the source. An
// unbounded Stream cannot be sorted. The sort is not stable.
func (s *Stream[T]) SortFunc(compare func(T, T) int) *Stream[T] {
	return ToStream(slices.SortedFunc(s.seq, compare))
}

// SortBy returns a Stream of the elements of s sorted in ascending order by the
// key that key extracts, which is usually more convenient than writing a
// comparator.
//
// key is called twice per comparison, so extract a cheap field rather than
// computing an expensive value. Like SortFunc it consumes s immediately and
// returns a slice-backed Stream, and the sort is not stable.
func (s *Stream[T]) SortBy[K cmp.Ordered](key func(T) K) *Stream[T] {
	return ToStream(slices.SortedFunc(s.seq, func(a, b T) int {
		return cmp.Compare(key(a), key(b))
	}))
}

// Seq returns the underlying sequence, for handing a Stream to code that takes
// an iter.Seq directly, such as slices.Collect, slices.Sorted, or a range loop.
//
// The sequence is not consumed by this call. FromSeq is the inverse.
func (s *Stream[T]) Seq() iter.Seq[T] {
	return s.seq
}
