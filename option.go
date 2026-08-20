package fgo

// Option is a container that either holds a value of type T or holds nothing.
// It makes absence explicit for types whose zero value is a legitimate value,
// where a bare T cannot distinguish "zero" from "missing".
//
// The zero value of Option is equivalent to None, so an unset struct field
// reads as an absent value rather than a present zero.
type Option[T any] struct {
	value T
	valid bool
}

// OptionFromTuple adapts the idiomatic Go comma-ok pair into an Option, for
// wrapping results such as a map lookup or type assertion.
//
// When ok is false the resulting Option holds nothing, regardless of val.
func OptionFromTuple[T any](val T, ok bool) Option[T] {
	return Option[T]{value: val, valid: ok}
}

// Some returns an Option holding v. A zero, empty, or false v still counts as
// a present value.
func Some[T any](v T) Option[T] {
	return Option[T]{value: v, valid: true}
}

// None returns an Option holding nothing. The type argument is required because
// there is no value to infer T from.
func None[T any]() Option[T] {
	return Option[T]{valid: false}
}

// Get returns the contained value and whether it is present, following Go's
// comma-ok convention. The first result is only meaningful when the second is
// true, and callers should ignore it otherwise.
//
// This is the only accessor that can distinguish Some of a zero value from
// None, since UnwrapOr cannot tell a present zero from an absent one.
func (o Option[T]) Get() (T, bool) {
	return o.value, o.valid
}

// Exists reports whether o holds a value.
func (o Option[T]) Exists() bool {
	return o.valid
}

// UnwrapOr returns the contained value, or fallback when o holds nothing.
//
// Note that a present zero value is returned as-is; fallback is only used for
// absence. Use Get when the two need to be told apart.
func (o Option[T]) UnwrapOr(fallback T) T {
	if o.valid {
		return o.value
	}
	return fallback
}

// UnwrapOrElse returns the contained value, or the result of calling fallback
// when o holds nothing.
//
// fallback is not called when a value is present, which makes this the right
// choice over UnwrapOr when computing the default is expensive or has side
// effects.
func (o Option[T]) UnwrapOrElse(fallback func() T) T {
	if o.valid {
		return o.value
	}
	return fallback()
}

// Map applies transform to the contained value and returns the result as a new
// Option, allowing the element type to change from T to U.
//
// When o holds nothing, transform is not called and an empty Option[U] is
// returned.
func (o Option[T]) Map[U any](transform func(T) U) Option[U] {
	if o.valid {
		return Some(transform(o.value))
	}
	return None[U]()
}

// FlatMap applies transform to the contained value and returns its Option
// directly rather than nesting it, allowing the element type to change from T
// to U.
//
// Use it instead of Map when transform may itself produce no value, such as a
// chain of lookups where any step can come up empty. When o holds nothing,
// transform is not called and an empty Option[U] is returned.
func (o Option[T]) FlatMap[U any](transform func(T) Option[U]) Option[U] {
	if o.valid {
		return transform(o.value)
	}
	return None[U]()
}

// Filter returns o unchanged when it holds a value that satisfies keep, and an
// empty Option otherwise. It turns a present-but-unwanted value into absence.
//
// When o holds nothing, keep is not called.
func (o Option[T]) Filter(keep func(T) bool) Option[T] {
	if o.valid && keep(o.value) {
		return o
	}
	return None[T]()
}

// OkOr converts o into a Result, supplying err as the failure when o holds
// nothing. It is the bridge from "missing" to "failed", for the point where an
// absent value becomes an error worth reporting.
//
// err is ignored when a value is present, so it is safe to build eagerly.
// Result.Ok performs the reverse conversion, discarding the error.
func (o Option[T]) OkOr(err error) Result[T] {
	if !o.valid {
		return Err[T](err)
	}
	return Ok(o.value)
}
