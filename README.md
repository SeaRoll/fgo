# fgo

Functional primitives for Go: `Option` for a value that may be absent, `Result` for a value that may have failed, and `Stream` for a lazy pipeline over a sequence.

Each type is a small wrapper whose methods chain, so a sequence of transformations reads top to bottom without an error check or a nil check between every step.

## Requirements

**Go 1.27 or later.** Methods carrying their own type parameters — `Option.Map`, `Result.TryMap`, `Stream.FlatMap`, `Stream.SortBy` — appear throughout. Before generic methods these had to be package-level functions, which meant they could not be chained; the style this package is built around depends on them.

## Install

```sh
go get github.com/SeaRoll/fgo
```

## Option

Makes absence explicit for types whose zero value is a legitimate value, where a bare `T` cannot distinguish "zero" from "missing".

```go
value, ok := ages["bob"]

age := fgo.OptionFromTuple(value, ok).
    Filter(func(n int) bool { return n >= 18 }).
    UnwrapOr(-1)
```

`Get` is the only accessor that can tell `Some(0)` from `None[int]()` — `UnwrapOr` cannot distinguish a present zero from an absent one:

```go
present, absent := fgo.Some(0), fgo.None[int]()

present.Get()        // 0, true
absent.Get()         // 0, false
present.UnwrapOr(-1) // 0
absent.UnwrapOr(-1)  // -1
```

`Option` implements `json.Marshaler` and `json.Unmarshaler`, so it works as a struct field directly. A present value encodes as itself, an absent one as `null`, and `null` or a missing key decodes back as absent:

```go
type user struct {
    Name     string             `json:"name"`
    Nickname fgo.Option[string] `json:"nickname"`
}
// {"name":"ann","nickname":"annie"}   — Some("annie")
// {"name":"bob","nickname":null}      — None[string]()
```

`omitempty` has no effect on a struct, so an absent value still appears as `null`. Use `omitzero` (Go 1.24 and later) to leave the field out entirely — it consults `IsZero`:

```go
Nickname fgo.Option[string] `json:"nickname,omitzero"`
// {"name":"bob"}   — the key disappears
```

| Constructors                      | Methods                                                                         | JSON                                        |
| --------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------- |
| `Some`, `None`, `OptionFromTuple` | `Get`, `Exists`, `UnwrapOr`, `UnwrapOrElse`, `Map`, `FlatMap`, `Filter`, `OkOr` | `MarshalJSON`, `UnmarshalJSON`, `IsZero` |

## Result

Carries a failure through a chain so each step does not have to check for one. `ResultFromTuple` and `Tuple` convert to and from the ordinary Go `(value, error)` pair, so a chain can start and end in idiomatic code.

```go
n, err := strconv.Atoi(raw)

port, err := fgo.ResultFromTuple(n, err).
    MapErr(func(err error) error { return fmt.Errorf("parsing port: %w", err) }).
    FlatMap(func(n int) fgo.Result[int] {
        if n <= 1024 {
            return fgo.Err[int](errors.New("port must be above 1024"))
        }
        return fgo.Ok(n)
    }).
    Tuple()
```

`TryMap` accepts the `(U, error)` shape that most Go functions already return, so no wrapping is needed:

```go
fgo.Ok("42").TryMap(strconv.Atoi).Tuple()  // 42, nil
fgo.Ok("abc").TryMap(strconv.Atoi).Tuple() // 0, strconv.Atoi: parsing "abc": invalid syntax
```

Wrap with `%w` in `MapErr` so `errors.Is` and `errors.As` keep working against the original error.

| Constructors                   | Methods                                                                                |
| ------------------------------ | -------------------------------------------------------------------------------------- |
| `Ok`, `Err`, `ResultFromTuple` | `Err`, `Tuple`, `UnwrapOr`, `UnwrapOrElse`, `Map`, `TryMap`, `FlatMap`, `MapErr`, `Ok` |

## Stream

Describes a pipeline over a sequence and does no work until a terminal operation runs.

```go
top := fgo.ToStream(orders).
    Filter(func(o order) bool { return o.total >= 80 }).
    SortBy(func(o order) int { return -o.total }).
    DistinctBy(func(o order) string { return o.customer }).
    Take(2).
    Collect()
```

`ToStream` starts from a slice, `FromSeq` from any `iter.Seq` — including the standard library's iterators or a hand-written generator — and `Seq` hands the sequence back out:

```go
fgo.Sorted(fgo.FromSeq(maps.Keys(ages))).Collect()

for v := range fgo.ToStream(items).Map(square).Seq() {
    // ...
}
```

|                            |                                                  |
| -------------------------- | ------------------------------------------------ |
| Sources                    | `ToStream`, `FromSeq`                            |
| Lazy                       | `Map`, `FlatMap`, `Filter`, `Take`, `DistinctBy` |
| Eager, slice-backed result | `SortBy`, `SortFunc`, `Sorted`                   |
| Terminal                   | `Collect`, `Reduce`, `Count`, `GroupBy`, `Seq`   |
| Terminal, short-circuiting | `Find`, `First`, `Any`, `All`                    |
| Terminal, returns `Option` | `Find`, `First`, `MinBy`, `MaxBy`                |

## Converting between the types

The three types meet at defined points:

```go
opt.OkOr(err)          // Option -> Result, supplying the error
res.Ok()               // Result -> Option, discarding the error
stream.Find(pred)      // Stream  -> Option, empty needs no sentinel
fgo.CollectResult(s)   // Stream[Result[T]] -> Result[[]T], fail-fast
```

`CollectResult` stops at the first error and discards the values gathered before it, so you get either every value or none:

```go
results := fgo.ToStream(inputs).Map(parse)

fgo.CollectResult(results).Tuple() // [1 2 3], nil  — or  [], first error
```

## Notes

**Take never over-reads.** Taking `n` elements reads exactly `n` from the source, never an `n+1`th, which matters when reading has a cost or a side effect. It is also what makes an unbounded source usable.

**Sorting cannot be lazy.** `SortBy`, `SortFunc` and `Sorted` must see every element, so they consume the source immediately. An unbounded Stream cannot be sorted. Neither sort is stable, and `SortBy` calls its key function twice per comparison — extract a cheap field rather than computing an expensive value.

**Reuse depends on the source, and is not visible from the type.** A Stream from `ToStream` is slice-backed and can be consumed repeatedly. A Stream from `FromSeq` inherits the behaviour of the sequence given, so a channel- or generator-backed Stream yields nothing on a second pass. The sorts always return a repeatable Stream regardless of the source.

**A few operations are functions, not methods.** `Sorted` constrains `T` to `cmp.Ordered` and `CollectResult` applies only to `Stream[Result[T]]`. Go does not let a method narrow its receiver's type parameter or target a single instantiation, so neither can chain. `SortBy` and `SortFunc` are the chainable alternatives to `Sorted`.

**Zero values differ between the two containers.** A zero `Option` reads as `None`. A zero `Result` has a nil error and so reads as a successful zero value — worth knowing for an unset struct field.

## Tests

```sh
go test ./...
go test -cover ./...
```
