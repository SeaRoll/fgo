package fgo

// Result is a container that either holds a value of type T or holds the error
// that prevented producing one. It carries a failure through a chain of
// operations so each step does not have to check for one.
//
// Unlike Option, the zero value of Result has a nil error and therefore reads
// as a successful zero value rather than a failure.
type Result[T any] struct {
	value T
	err   error
}

// ResultFromTuple adapts the idiomatic Go (value, error) pair into a Result,
// for wrapping the return of an ordinary function.
//
// When err is non-nil the resulting Result holds the error and discards val.
// Result.Get performs the reverse conversion.
func ResultFromTuple[T any](val T, err error) Result[T] {
	if err != nil {
		return Err[T](err)
	}
	return Ok(val)
}

// Ok returns a successful Result holding v. A zero, empty, or false v is still
// a success.
func Ok[T any](v T) Result[T] {
	return Result[T]{value: v, err: nil}
}

// Err returns a failed Result holding err. The type argument is required
// because there is no value to infer T from.
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// Err returns the error that caused the failure, or nil when r holds a value.
func (r Result[T]) Err() error {
	return r.err
}

// Get returns the contained value and error as the idiomatic Go pair, for
// handing a Result back to ordinary error handling at the edge of a chain.
//
// The value is only meaningful when the error is nil. It is the inverse of
// ResultFromTuple, and mirrors Option.Get.
func (r Result[T]) Get() (T, error) {
	return r.value, r.err
}

// UnwrapOr returns the contained value, or fallback when r holds an error. The
// error is discarded; use UnwrapOrElse or Err to inspect it.
func (r Result[T]) UnwrapOr(fallback T) T {
	if r.err != nil {
		return fallback
	}
	return r.value
}

// UnwrapOrElse returns the contained value, or the result of calling fallback
// with the error when r holds one.
//
// Because fallback receives the error, it can log or derive a default from the
// specific failure. It is not called when a value is present.
func (r Result[T]) UnwrapOrElse(fallback func(error) T) T {
	if r.err != nil {
		return fallback(r.err)
	}
	return r.value
}

// Map applies transform to the contained value and returns the result as a new
// Result, allowing the element type to change from T to U.
//
// When r holds an error, transform is not called and the original error is
// carried through unchanged. Use TryMap when transform can itself fail.
func (r Result[T]) Map[U any](transform func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(transform(r.value))
}

// TryMap applies a transform that may fail, accepting the (U, error) shape that
// most Go functions already return, and allowing the element type to change
// from T to U.
//
// A non-nil error from transform becomes the failure and its value is
// discarded. When r already holds an error, transform is not called and the
// original error is carried through unchanged.
func (r Result[T]) TryMap[U any](transform func(T) (U, error)) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return ResultFromTuple(transform(r.value))
}

// FlatMap applies transform to the contained value and returns its Result
// directly rather than nesting it, allowing the element type to change from T
// to U.
//
// Use it when transform already returns a Result, such as a validation step
// that decides its own error. When r holds an error, transform is not called
// and the original error is carried through unchanged.
func (r Result[T]) FlatMap[U any](transform func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return transform(r.value)
}

// MapErr replaces the contained error with the result of transform, leaving a
// successful Result untouched. It is the hook for annotating a failure with
// context as it travels up a chain.
//
// Wrap with %w so that errors.Is and errors.As keep working against the
// original error. transform is not called when r holds a value.
func (r Result[T]) MapErr(transform func(error) error) Result[T] {
	if r.err != nil {
		return Err[T](transform(r.err))
	}
	return r
}

// Ok converts r into an Option, discarding the error. It is the bridge from
// "failed" to "missing", for the point where why something failed stops
// mattering.
//
// Option.OkOr performs the reverse conversion, supplying an error.
func (r Result[T]) Ok() Option[T] {
	if r.err != nil {
		return None[T]()
	}
	return Some(r.value)
}
