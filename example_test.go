package fgo_test

import (
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/SeaRoll/fgo"
)

func ExampleOption() {
	settings := map[string]string{"port": "8080", "timeout": "0"}

	// Look up a setting, parse it, and reject values outside the valid range,
	// falling back to a default if any step comes up empty.
	read := func(key string) int {
		raw, ok := settings[key]
		return fgo.OptionFromTuple(raw, ok).
			FlatMap(func(s string) fgo.Option[int] {
				n, err := strconv.Atoi(s)
				return fgo.OptionFromTuple(n, err == nil)
			}).
			Filter(func(n int) bool { return n > 0 }).
			UnwrapOr(-1)
	}

	fmt.Println(read("port"))    // parses and passes the filter
	fmt.Println(read("timeout")) // parses but fails the filter
	fmt.Println(read("host"))    // missing entirely
	// Output:
	// 8080
	// -1
	// -1
}

func ExampleSome() {
	fmt.Println(fgo.Some(42).UnwrapOr(0))

	// A falsy value is still a present value.
	fmt.Println(fgo.Some(false).UnwrapOr(true))
	// Output:
	// 42
	// false
}

func ExampleNone() {
	absent := fgo.None[string]()

	fmt.Println(absent.Exists())
	fmt.Println(absent.UnwrapOr("default"))
	// Output:
	// false
	// default
}

func ExampleOptionFromTuple() {
	ages := map[string]int{"ann": 30}

	value, ok := ages["ann"]
	found := fgo.OptionFromTuple(value, ok)

	value, ok = ages["bob"]
	missing := fgo.OptionFromTuple(value, ok)

	fmt.Println(found.UnwrapOr(-1))
	fmt.Println(missing.UnwrapOr(-1))
	// Output:
	// 30
	// -1
}

func ExampleOption_Get() {
	present := fgo.Some(0)
	absent := fgo.None[int]()

	value, ok := present.Get()
	fmt.Println(value, ok)

	value, ok = absent.Get()
	fmt.Println(value, ok)

	// UnwrapOr cannot tell a present zero from an absent one; Get can.
	fmt.Println(present.UnwrapOr(-1), absent.UnwrapOr(-1))
	// Output:
	// 0 true
	// 0 false
	// 0 -1
}

func ExampleOption_Exists() {
	// An empty string is present, so Exists reports true.
	fmt.Println(fgo.Some("").Exists())
	fmt.Println(fgo.None[string]().Exists())

	// The zero value of Option behaves as None.
	var unset fgo.Option[string]
	fmt.Println(unset.Exists())
	// Output:
	// true
	// false
	// false
}

func ExampleOption_UnwrapOr() {
	fmt.Println(fgo.Some(10).UnwrapOr(-1))
	fmt.Println(fgo.None[int]().UnwrapOr(-1))
	// Output:
	// 10
	// -1
}

func ExampleOption_UnwrapOrElse() {
	expensive := func() int {
		fmt.Println("computing default")
		return -1
	}

	// The fallback is skipped entirely when a value is present.
	fmt.Println(fgo.Some(42).UnwrapOrElse(expensive))
	fmt.Println(fgo.None[int]().UnwrapOrElse(expensive))
	// Output:
	// 42
	// computing default
	// -1
}

func ExampleOption_Map() {
	// Map can change the element type, here from string to int.
	length := func(s string) int { return len(s) }

	fmt.Println(fgo.Some("hello").Map(length).UnwrapOr(0))
	fmt.Println(fgo.None[string]().Map(length).UnwrapOr(0))
	// Output:
	// 5
	// 0
}

func ExampleOption_FlatMap() {
	managers := map[string]string{"ann": "bob"}
	titles := map[string]string{"bob": "director"}

	lookup := func(m map[string]string) func(string) fgo.Option[string] {
		return func(key string) fgo.Option[string] {
			value, ok := m[key]
			return fgo.OptionFromTuple(value, ok)
		}
	}

	// Each step can come up empty, so the transform returns an Option itself.
	fmt.Println(lookup(managers)("ann").FlatMap(lookup(titles)).UnwrapOr("unknown"))

	// cid has no manager, so the chain stops before the second lookup.
	fmt.Println(lookup(managers)("cid").FlatMap(lookup(titles)).UnwrapOr("unknown"))
	// Output:
	// director
	// unknown
}

func ExampleOption_Filter() {
	positive := func(v int) bool { return v > 0 }

	fmt.Println(fgo.Some(5).Filter(positive).Exists())  // kept
	fmt.Println(fgo.Some(-5).Filter(positive).Exists()) // present but rejected
	fmt.Println(fgo.None[int]().Filter(positive).Exists())
	// Output:
	// true
	// false
	// false
}

func ExampleOption_OkOr() {
	errNotFound := errors.New("not found")

	// A present value ignores the error entirely.
	fmt.Println(fgo.Some(7).OkOr(errNotFound).Tuple())

	// An absent value becomes a failure worth reporting.
	fmt.Println(fgo.None[int]().OkOr(errNotFound).Tuple())
	// Output:
	// 7 <nil>
	// 0 not found
}

func ExampleResult() {
	// Parse a port, annotate any parse failure, then validate the range.
	parsePort := func(raw string) fgo.Result[int] {
		n, err := strconv.Atoi(raw)
		return fgo.ResultFromTuple(n, err).
			MapErr(func(err error) error { return fmt.Errorf("parsing port: %w", err) }).
			FlatMap(func(n int) fgo.Result[int] {
				if n <= 1024 {
					return fgo.Err[int](errors.New("port must be above 1024"))
				}
				return fgo.Ok(n)
			})
	}

	fmt.Println(parsePort("8080").Tuple())
	fmt.Println(parsePort("80").Tuple())
	fmt.Println(parsePort("abc").Tuple())
	// Output:
	// 8080 <nil>
	// 0 port must be above 1024
	// 0 parsing port: strconv.Atoi: parsing "abc": invalid syntax
}

func ExampleOk() {
	fmt.Println(fgo.Ok(42).Tuple())

	// A zero value is still a success.
	fmt.Println(fgo.Ok(0).Err())
	// Output:
	// 42 <nil>
	// <nil>
}

func ExampleErr() {
	failed := fgo.Err[int](errors.New("boom"))

	fmt.Println(failed.Err())
	fmt.Println(failed.UnwrapOr(-1))
	// Output:
	// boom
	// -1
}

func ExampleResultFromTuple() {
	// Wraps the return of any function with an (T, error) signature.
	n, err := strconv.Atoi("42")
	fmt.Println(fgo.ResultFromTuple(n, err).Tuple())

	// The value is discarded when the error is non-nil.
	n, err = strconv.Atoi("nope")
	fmt.Println(fgo.ResultFromTuple(n, err).UnwrapOr(-1))
	// Output:
	// 42 <nil>
	// -1
}

func ExampleResult_Err() {
	fmt.Println(fgo.Ok("data").Err())
	fmt.Println(fgo.Err[string](errors.New("disk full")).Err())
	// Output:
	// <nil>
	// disk full
}

func ExampleResult_Tuple() {
	// Tuple hands a Result back to ordinary Go error handling.
	value, err := fgo.Ok("config").Tuple()
	fmt.Printf("%q %v\n", value, err)

	value, err = fgo.Err[string](errors.New("missing")).Tuple()
	fmt.Printf("%q %v\n", value, err)
	// Output:
	// "config" <nil>
	// "" missing
}

func ExampleResult_UnwrapOr() {
	fmt.Println(fgo.Ok(10).UnwrapOr(-1))
	fmt.Println(fgo.Err[int](errors.New("timeout")).UnwrapOr(-1))
	// Output:
	// 10
	// -1
}

func ExampleResult_UnwrapOrElse() {
	onErr := func(err error) int {
		fmt.Println("recovering from:", err)
		return -1
	}

	// The fallback receives the error, and is skipped on success.
	fmt.Println(fgo.Ok(42).UnwrapOrElse(onErr))
	fmt.Println(fgo.Err[int](errors.New("timeout")).UnwrapOrElse(onErr))
	// Output:
	// 42
	// recovering from: timeout
	// -1
}

func ExampleResult_Map() {
	double := func(v int) string { return strconv.Itoa(v * 2) }

	value, err := fgo.Ok(21).Map(double).Tuple()
	fmt.Printf("%q %v\n", value, err)

	// The original error survives the change of element type.
	value, err = fgo.Err[int](errors.New("no input")).Map(double).Tuple()
	fmt.Printf("%q %v\n", value, err)
	// Output:
	// "42" <nil>
	// "" no input
}

func ExampleResult_TryMap() {
	// TryMap accepts any func(T) (U, error), which most Go functions already are.
	fmt.Println(fgo.Ok("42").TryMap(strconv.Atoi).Tuple())
	fmt.Println(fgo.Ok("abc").TryMap(strconv.Atoi).Tuple())
	// Output:
	// 42 <nil>
	// 0 strconv.Atoi: parsing "abc": invalid syntax
}

func ExampleResult_FlatMap() {
	// The transform decides its own error, so it returns a Result.
	checkPositive := func(v int) fgo.Result[int] {
		if v <= 0 {
			return fgo.Err[int](errors.New("must be positive"))
		}
		return fgo.Ok(v)
	}

	fmt.Println(fgo.Ok(5).FlatMap(checkPositive).Tuple())
	fmt.Println(fgo.Ok(-5).FlatMap(checkPositive).Tuple())
	// Output:
	// 5 <nil>
	// 0 must be positive
}

func ExampleResult_MapErr() {
	errNotFound := errors.New("not found")

	wrapped := fgo.Err[int](errNotFound).
		MapErr(func(err error) error { return fmt.Errorf("loading user: %w", err) })

	fmt.Println(wrapped.Err())

	// Wrapping with %w keeps errors.Is working against the original.
	fmt.Println(errors.Is(wrapped.Err(), errNotFound))

	// A successful Result is left untouched.
	fmt.Println(fgo.Ok(1).MapErr(func(error) error { return errNotFound }).Tuple())
	// Output:
	// loading user: not found
	// true
	// 1 <nil>
}

func ExampleResult_Ok() {
	// Ok converts to an Option, discarding why the failure happened.
	fmt.Println(fgo.Ok(7).Ok().Get())
	fmt.Println(fgo.Err[int](errors.New("failed")).Ok().Get())
	// Output:
	// 7 true
	// 0 false
}

func ExampleStream() {
	type order struct {
		customer string
		total    int
	}
	orders := []order{
		{"ann", 120}, {"bob", 80}, {"ann", 200}, {"cid", 150}, {"bob", 60},
	}

	// The two biggest orders, one per customer.
	top := fgo.ToStream(orders).
		Filter(func(o order) bool { return o.total >= 80 }).
		SortBy(func(o order) int { return -o.total }).
		DistinctBy(func(o order) string { return o.customer }).
		Take(2).
		Map(func(o order) string { return fmt.Sprintf("%s:%d", o.customer, o.total) }).
		Collect()

	fmt.Println(top)
	// Output:
	// [ann:200 cid:150]
}

func ExampleToStream() {
	stream := fgo.ToStream([]int{1, 2, 3})

	// A slice-backed Stream can be consumed more than once.
	fmt.Println(stream.Collect())
	fmt.Println(stream.Count())
	// Output:
	// [1 2 3]
	// 3
}

func ExampleFromSeq() {
	ages := map[string]int{"ann": 30, "bob": 25}

	// Bridges any iter.Seq, such as the standard library's map iterators.
	fmt.Println(fgo.Sorted(fgo.FromSeq(maps.Keys(ages))).Collect())

	// A nil sequence is an empty Stream rather than a panic.
	fmt.Println(fgo.FromSeq[int](nil).Collect())
	// Output:
	// [ann bob]
	// []
}

func ExampleStream_Reduce() {
	fmt.Println(fgo.ToStream([]int{1, 2, 3, 4}).Reduce(0, func(acc, item int) int {
		return acc + item
	}))

	// The accumulator type may differ from the element type.
	fmt.Println(fgo.ToStream([]int{1, 2, 3}).Reduce("", func(acc string, item int) string {
		return acc + strconv.Itoa(item)
	}))
	// Output:
	// 10
	// 123
}

func ExampleStream_Map() {
	lengths := fgo.ToStream([]string{"a", "bb", "ccc"}).
		Map(func(s string) int { return len(s) }).
		Collect()

	fmt.Println(lengths)
	// Output:
	// [1 2 3]
}

func ExampleStream_FlatMap() {
	// Expand each line into its words.
	words := fgo.ToStream([]string{"the quick brown", "fox"}).
		FlatMap(func(line string) *fgo.Stream[string] {
			return fgo.ToStream(strings.Fields(line))
		}).
		Collect()

	fmt.Println(words)
	// Output:
	// [the quick brown fox]
}

func ExampleStream_Filter() {
	evens := fgo.ToStream([]int{1, 2, 3, 4, 5, 6}).
		Filter(func(v int) bool { return v%2 == 0 }).
		Collect()

	fmt.Println(evens)
	// Output:
	// [2 4 6]
}

func ExampleStream_Take() {
	// Take bounds an otherwise infinite sequence.
	var naturals iter.Seq[int] = func(yield func(int) bool) {
		for n := 1; ; n++ {
			if !yield(n) {
				return
			}
		}
	}

	fmt.Println(fgo.FromSeq(naturals).Take(5).Collect())

	// A non-positive n yields nothing.
	fmt.Println(fgo.ToStream([]int{1, 2, 3}).Take(0).Collect())
	// Output:
	// [1 2 3 4 5]
	// []
}

func ExampleStream_DistinctBy() {
	// Keeps the first fruit for each starting letter.
	firsts := fgo.ToStream([]string{"apple", "avocado", "banana", "blueberry"}).
		DistinctBy(func(s string) byte { return s[0] }).
		Collect()

	fmt.Println(firsts)
	// Output:
	// [apple banana]
}

func ExampleStream_Collect() {
	fmt.Println(fgo.ToStream([]int{1, 2, 3}).Collect())

	// An empty Stream collects to a nil slice.
	fmt.Println(fgo.ToStream([]int{}).Collect() == nil)
	// Output:
	// [1 2 3]
	// true
}

func ExampleStream_Find() {
	fmt.Println(fgo.ToStream([]int{1, 2, 3, 4}).Find(func(v int) bool { return v > 2 }).Get())
	fmt.Println(fgo.ToStream([]int{1, 2}).Find(func(v int) bool { return v > 9 }).Get())
	// Output:
	// 3 true
	// 0 false
}

func ExampleStream_First() {
	fmt.Println(fgo.ToStream([]string{"a", "b"}).First().UnwrapOr("none"))
	fmt.Println(fgo.ToStream([]string{}).First().UnwrapOr("none"))
	// Output:
	// a
	// none
}

func ExampleStream_Any() {
	even := func(v int) bool { return v%2 == 0 }

	fmt.Println(fgo.ToStream([]int{1, 3, 4}).Any(even))
	fmt.Println(fgo.ToStream([]int{1, 3}).Any(even))

	// Any is false for an empty Stream.
	fmt.Println(fgo.ToStream([]int{}).Any(even))
	// Output:
	// true
	// false
	// false
}

func ExampleStream_All() {
	even := func(v int) bool { return v%2 == 0 }

	fmt.Println(fgo.ToStream([]int{2, 4}).All(even))
	fmt.Println(fgo.ToStream([]int{2, 3}).All(even))

	// All is vacuously true for an empty Stream.
	fmt.Println(fgo.ToStream([]int{}).All(even))
	// Output:
	// true
	// false
	// true
}

func ExampleStream_Count() {
	fmt.Println(fgo.ToStream([]int{1, 2, 3}).Count())
	fmt.Println(fgo.ToStream([]int{1, 2, 3, 4}).
		Filter(func(v int) bool { return v%2 == 0 }).
		Count())
	// Output:
	// 3
	// 2
}

func ExampleStream_MinBy() {
	type product struct {
		name  string
		price int
	}
	products := []product{{"pen", 300}, {"book", 1200}, {"mug", 800}}

	cheapest, ok := fgo.ToStream(products).MinBy(func(p product) int { return p.price }).Get()
	fmt.Println(cheapest.name, ok)

	// An empty Stream has no minimum.
	_, ok = fgo.ToStream([]product{}).MinBy(func(p product) int { return p.price }).Get()
	fmt.Println(ok)
	// Output:
	// pen true
	// false
}

func ExampleStream_MaxBy() {
	type product struct {
		name  string
		price int
	}
	products := []product{{"pen", 300}, {"book", 1200}, {"mug", 800}}

	priciest, ok := fgo.ToStream(products).MaxBy(func(p product) int { return p.price }).Get()
	fmt.Println(priciest.name, ok)
	// Output:
	// book true
}

func ExampleStream_GroupBy() {
	groups := fgo.ToStream([]string{"apple", "avocado", "banana"}).
		GroupBy(func(s string) string { return s[:1] })

	// Map iteration order is undefined, so read specific keys.
	fmt.Println(groups["a"])
	fmt.Println(groups["b"])
	// Output:
	// [apple avocado]
	// [banana]
}

func ExampleStream_SortFunc() {
	// Descending, using the slices package comparator convention.
	descending := fgo.ToStream([]int{3, 1, 2}).
		SortFunc(func(a, b int) int { return b - a }).
		Collect()

	fmt.Println(descending)
	// Output:
	// [3 2 1]
}

func ExampleStream_SortBy() {
	type employee struct {
		name string
		age  int
	}
	staff := []employee{{"ann", 30}, {"bob", 25}, {"cid", 40}}

	byAge := fgo.ToStream(staff).
		SortBy(func(e employee) int { return e.age }).
		Map(func(e employee) string { return e.name }).
		Collect()

	fmt.Println(byAge)
	// Output:
	// [bob ann cid]
}

func ExampleStream_Seq() {
	seq := fgo.ToStream([]int{1, 2, 3}).
		Map(func(v int) int { return v * v }).
		Seq()

	// The sequence works anywhere an iter.Seq is accepted.
	for v := range seq {
		fmt.Println(v)
	}
	fmt.Println(slices.Collect(seq))
	// Output:
	// 1
	// 4
	// 9
	// [1 4 9]
}

func ExampleSorted() {
	fmt.Println(fgo.Sorted(fgo.ToStream([]int{3, 1, 2})).Collect())
	fmt.Println(fgo.Sorted(fgo.ToStream([]string{"pear", "apple"})).Collect())
	// Output:
	// [1 2 3]
	// [apple pear]
}

func ExampleCollectResult() {
	parseAll := func(inputs []string) fgo.Result[[]int] {
		return fgo.CollectResult(fgo.ToStream(inputs).Map(func(s string) fgo.Result[int] {
			n, err := strconv.Atoi(s)
			return fgo.ResultFromTuple(n, err)
		}))
	}

	fmt.Println(parseAll([]string{"1", "2", "3"}).Tuple())

	// The first failure wins, and the values gathered before it are discarded.
	fmt.Println(parseAll([]string{"1", "nope", "3"}).Tuple())
	// Output:
	// [1 2 3] <nil>
	// [] strconv.Atoi: parsing "nope": invalid syntax
}
