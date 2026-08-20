// Package fgo provides functional primitives for Go: [Option] for a value that
// may be absent, [Result] for a value that may have failed, and [Stream] for a
// lazy pipeline over a sequence.
//
// Each type is a small wrapper with methods that chain, so a sequence of
// transformations reads top to bottom without an error check or a nil check
// between every step.
//
// # Requirements
//
// fgo requires Go 1.27 or later. Methods carrying their own type parameters,
// such as [Option.Map] and [Stream.FlatMap], appear throughout. Before generic
// methods these had to be package-level functions, which meant they could not be
// chained; the chaining style this package is built around depends on them.
//
// # Option
//
// Option makes absence explicit for types whose zero value is a legitimate
// value, where a bare T cannot distinguish "zero" from "missing":
//
//	value, ok := ages["bob"]
//	age := fgo.OptionFromTuple(value, ok).
//		Filter(func(n int) bool { return n >= 18 }).
//		UnwrapOr(-1)
//
// [Option.Get] is the only accessor that can tell Some of a zero value apart
// from None, since UnwrapOr cannot distinguish a present zero from an absent
// one.
//
// # Result
//
// Result carries a failure through a chain so each step does not have to check
// for one. [ResultFromTuple] and [Result.Tuple] convert to and from the ordinary
// Go (value, error) pair, so a chain can start and end in idiomatic code:
//
//	n, err := strconv.Atoi(raw)
//	port, err := fgo.ResultFromTuple(n, err).
//		MapErr(func(err error) error { return fmt.Errorf("parsing port: %w", err) }).
//		Tuple()
//
// [Result.TryMap] accepts the (U, error) shape that most Go functions already
// return, and [Result.MapErr] is the hook for annotating a failure with context
// as it travels up a chain.
//
// # Stream
//
// Stream describes a pipeline over a sequence and does no work until a terminal
// operation runs:
//
//	top := fgo.ToStream(orders).
//		Filter(func(o order) bool { return o.total >= 80 }).
//		SortBy(func(o order) int { return -o.total }).
//		Take(2).
//		Collect()
//
// [ToStream] starts from a slice and [FromSeq] from any iter.Seq, including the
// standard library's map and slice iterators or a hand-written generator.
// [Stream.Seq] hands the sequence back out again.
//
// # Converting between the types
//
// The three types meet at defined points. [Option.OkOr] turns an absent value
// into a failure by supplying an error, and [Result.Ok] goes the other way,
// discarding why the failure happened. Stream's short-circuiting operations,
// such as [Stream.Find] and [Stream.MinBy], return an Option so an empty Stream
// needs no sentinel, and [CollectResult] collapses a Stream of Results into a
// single Result holding either every value or the first error.
//
// # Laziness and reuse
//
// Stream's intermediate operations are lazy, and a terminal operation that can
// answer early stops reading the source at that point: [Stream.Take] never reads
// an element beyond its limit, which is what makes an unbounded source usable.
//
// Sorting is the exception. [Stream.SortBy], [Stream.SortFunc] and [Sorted] must
// see every element, so they consume the source immediately and return a
// slice-backed Stream.
//
// Whether a Stream can be consumed more than once depends on its source and is
// not visible from the type. One from [ToStream] is slice-backed and repeatable;
// one from [FromSeq] inherits the behaviour of the sequence given, so a channel-
// or generator-backed Stream yields nothing on a second pass.
package fgo
