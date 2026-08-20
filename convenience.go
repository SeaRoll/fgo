package fgo

import (
	"cmp"
	"slices"
)

// Sorted returns a Stream of the elements of s in ascending order.
//
// It is a function rather than a method because it constrains T to cmp.Ordered,
// and a method cannot narrow the type parameter of its receiver. Stream.SortBy
// and Stream.SortFunc sort by a key or a comparator and do chain.
//
// Like the other sorts, it consumes s immediately and returns a slice-backed,
// repeatable Stream, so an unbounded Stream cannot be sorted.
func Sorted[T cmp.Ordered](s *Stream[T]) *Stream[T] {
	return ToStream(slices.Sorted(s.seq))
}

// CollectResult consumes a Stream of Results and returns a single Result holding
// every value, or the first error encountered. It is the terminal operation for
// a pipeline whose steps can fail.
//
// It fails fast: at the first error it stops reading the Stream and discards the
// values gathered so far, so the result is either every value or none. An empty
// Stream yields Ok of a nil slice.
//
// It is a function rather than a method because it applies only to
// Stream[Result[T]], and a method cannot target a single instantiation of its
// receiver.
func CollectResult[T any](s *Stream[Result[T]]) Result[[]T] {
	var items []T
	for r := range s.seq {
		if r.err != nil {
			return Err[[]T](r.err)
		}
		items = append(items, r.value)
	}
	return Ok(items)
}
