package fgo_test

import (
	"iter"
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/SeaRoll/fgo"
)

type person struct {
	name string
	age  int
}

// countingSeq reports how many elements a consumer actually pulled, which is how
// the early-exit tests below distinguish laziness from full consumption.
func countingSeq(items []int, pulled *int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for _, v := range items {
			*pulled++
			if !yield(v) {
				return
			}
		}
	}
}

// onceSeq can only be iterated a single time, like a channel or generator source.
func onceSeq(items []int) iter.Seq[int] {
	spent := false
	return func(yield func(int) bool) {
		if spent {
			return
		}
		spent = true
		for _, v := range items {
			if !yield(v) {
				return
			}
		}
	}
}

func TestToStreamCollect(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []int
	}{
		{"several items", []int{1, 2, 3}, []int{1, 2, 3}},
		{"single item", []int{7}, []int{7}},
		{"empty slice", []int{}, nil},
		{"nil slice", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fgo.ToStream(tc.items).Collect(); !slices.Equal(got, tc.want) {
				t.Errorf("Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFromSeq(t *testing.T) {
	tests := []struct {
		name string
		seq  iter.Seq[int]
		want []int
	}{
		{"wraps a sequence", slices.Values([]int{1, 2}), []int{1, 2}},
		// A nil sequence must not panic on first use.
		{"nil sequence is empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fgo.FromSeq(tc.seq).Collect(); !slices.Equal(got, tc.want) {
				t.Errorf("Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamSeqRoundTrip(t *testing.T) {
	items := []int{4, 5, 6}
	got := fgo.FromSeq(fgo.ToStream(items).Seq()).Collect()
	if !slices.Equal(got, items) {
		t.Errorf("round trip = %v, want %v", got, items)
	}
}

func TestStreamReduce(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  int
	}{
		{"sums items onto initial", []int{1, 2, 3, 4}, 110},
		{"empty returns initial", nil, 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).Reduce(100, func(acc, item int) int { return acc + item })
			if got != tc.want {
				t.Errorf("Reduce() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStreamReduceChangesType(t *testing.T) {
	got := fgo.ToStream([]int{1, 2, 3}).Reduce("", func(acc string, item int) string {
		return acc + strconv.Itoa(item)
	})
	if got != "123" {
		t.Errorf("Reduce() = %q, want %q", got, "123")
	}
}

func TestStreamMap(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []string
	}{
		{"transforms every item", []int{1, 2, 3}, []string{"1", "2", "3"}},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).Map(strconv.Itoa).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Map().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamMapIsLazy(t *testing.T) {
	calls := 0
	stream := fgo.ToStream([]int{1, 2, 3}).Map(func(v int) int {
		calls++
		return v
	})
	if calls != 0 {
		t.Fatalf("transform called %d times before consumption, want 0", calls)
	}
	stream.Collect()
	if calls != 3 {
		t.Errorf("transform called %d times after consumption, want 3", calls)
	}
}

func TestStreamFilter(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []int
	}{
		{"keeps matching", []int{1, 2, 3, 4}, []int{2, 4}},
		{"keeps none", []int{1, 3}, nil},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).Filter(func(v int) bool { return v%2 == 0 }).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Filter().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamFlatMap(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []string
	}{
		{"expands each item", []int{1, 2}, []string{"1", "1", "2", "2"}},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).FlatMap(func(v int) *fgo.Stream[string] {
				s := strconv.Itoa(v)
				return fgo.ToStream([]string{s, s})
			}).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("FlatMap().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamFlatMapDegenerateInnerStreams(t *testing.T) {
	tests := []struct {
		name  string
		inner func(int) *fgo.Stream[string]
		want  []string
	}{
		{"nil inner stream is skipped", func(int) *fgo.Stream[string] { return nil }, nil},
		{"zero value inner stream is skipped", func(int) *fgo.Stream[string] { return &fgo.Stream[string]{} }, nil},
		{"empty inner stream is skipped", func(int) *fgo.Stream[string] { return fgo.ToStream([]string{}) }, nil},
		{
			"only some items contribute",
			func(v int) *fgo.Stream[string] {
				if v%2 == 0 {
					return nil
				}
				return fgo.ToStream([]string{strconv.Itoa(v)})
			},
			[]string{"1", "3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream([]int{1, 2, 3}).FlatMap(tc.inner).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("FlatMap().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamTake(t *testing.T) {
	tests := []struct {
		name       string
		n          int
		want       []int
		wantPulled int
	}{
		{"fewer than available", 2, []int{1, 2}, 2},
		{"exactly available", 5, []int{1, 2, 3, 4, 5}, 5},
		{"more than available", 9, []int{1, 2, 3, 4, 5}, 5},
		{"zero takes nothing and pulls nothing", 0, nil, 0},
		{"negative takes nothing and pulls nothing", -3, nil, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			got := fgo.FromSeq(countingSeq([]int{1, 2, 3, 4, 5}, &pulled)).Take(tc.n).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Take(%d).Collect() = %v, want %v", tc.n, got, tc.want)
			}
			// Taking n must not pull an n+1th element from the source.
			if pulled != tc.wantPulled {
				t.Errorf("Take(%d) pulled %d elements, want %d", tc.n, pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamDistinctBy(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{"drops later duplicates", []string{"a", "bb", "c", "dd", "e"}, []string{"a", "bb"}},
		{"all distinct", []string{"a", "bb", "ccc"}, []string{"a", "bb", "ccc"}},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).DistinctBy(func(s string) int { return len(s) }).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("DistinctBy().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamDistinctByIsRepeatable(t *testing.T) {
	// The seen-set must be rebuilt per iteration, or a second pass filters everything.
	stream := fgo.ToStream([]int{1, 1, 2, 2, 3}).DistinctBy(func(v int) int { return v })
	want := []int{1, 2, 3}

	first := stream.Collect()
	second := stream.Collect()

	if !slices.Equal(first, want) {
		t.Errorf("first Collect() = %v, want %v", first, want)
	}
	if !slices.Equal(second, want) {
		t.Errorf("second Collect() = %v, want %v", second, want)
	}
}

// Each lazy combinator must propagate a downstream stop back to its source
// instead of draining it.
func TestStreamEarlyExitThroughCombinators(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		pipeline   func(*fgo.Stream[int]) []int
		want       []int
		wantPulled int
	}{
		{
			"through Map",
			[]int{1, 2, 3, 4, 5},
			func(s *fgo.Stream[int]) []int {
				return s.Map(func(v int) int { return v * 2 }).Take(2).Collect()
			},
			[]int{2, 4}, 2,
		},
		{
			"through Filter",
			[]int{1, 2, 3, 4, 5, 6},
			func(s *fgo.Stream[int]) []int {
				return s.Filter(func(v int) bool { return v%2 == 0 }).Take(2).Collect()
			},
			[]int{2, 4}, 4,
		},
		{
			"through DistinctBy",
			[]int{1, 1, 2, 2, 3},
			func(s *fgo.Stream[int]) []int {
				return s.DistinctBy(func(v int) int { return v }).Take(2).Collect()
			},
			[]int{1, 2}, 3,
		},
		{
			"through FlatMap",
			[]int{1, 2, 3, 4, 5},
			func(s *fgo.Stream[int]) []int {
				return s.FlatMap(func(v int) *fgo.Stream[int] {
					return fgo.ToStream([]int{v, v})
				}).Take(3).Collect()
			},
			[]int{1, 1, 2}, 2,
		},
		{
			"through Take into First",
			[]int{1, 2, 3, 4, 5},
			func(s *fgo.Stream[int]) []int {
				if v, ok := s.Take(4).First().Get(); ok {
					return []int{v}
				}
				return nil
			},
			[]int{1}, 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			got := tc.pipeline(fgo.FromSeq(countingSeq(tc.items, &pulled)))
			if !slices.Equal(got, tc.want) {
				t.Errorf("pipeline = %v, want %v", got, tc.want)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamFind(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		target     int
		wantValue  int
		wantValid  bool
		wantPulled int
	}{
		{"finds and stops", []int{1, 2, 3, 4}, 2, 2, true, 2},
		{"finds first element", []int{1, 2, 3, 4}, 1, 1, true, 1},
		{"no match consumes all", []int{1, 2, 3, 4}, 9, 0, false, 4},
		{"empty is none", nil, 1, 0, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			value, valid := fgo.FromSeq(countingSeq(tc.items, &pulled)).
				Find(func(v int) bool { return v == tc.target }).Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Find().Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamFirst(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		wantValue  int
		wantValid  bool
		wantPulled int
	}{
		{"returns head and stops", []int{9, 8, 7}, 9, true, 1},
		{"empty is none", nil, 0, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			value, valid := fgo.FromSeq(countingSeq(tc.items, &pulled)).First().Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("First().Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamAny(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		want       bool
		wantPulled int
	}{
		{"match short circuits", []int{1, 2, 3, 4}, true, 2},
		{"no match consumes all", []int{1, 3, 5}, false, 3},
		{"empty is false", nil, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			got := fgo.FromSeq(countingSeq(tc.items, &pulled)).Any(func(v int) bool { return v%2 == 0 })
			if got != tc.want {
				t.Errorf("Any() = %t, want %t", got, tc.want)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamAll(t *testing.T) {
	tests := []struct {
		name       string
		items      []int
		want       bool
		wantPulled int
	}{
		{"failure short circuits", []int{2, 3, 4, 6}, false, 2},
		{"all match consumes all", []int{2, 4, 6}, true, 3},
		{"empty is vacuously true", nil, true, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			got := fgo.FromSeq(countingSeq(tc.items, &pulled)).All(func(v int) bool { return v%2 == 0 })
			if got != tc.want {
				t.Errorf("All() = %t, want %t", got, tc.want)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestStreamCount(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  int
	}{
		{"counts items", []int{1, 2, 3}, 3},
		{"empty is zero", nil, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fgo.ToStream(tc.items).Count(); got != tc.want {
				t.Errorf("Count() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStreamMinByMaxBy(t *testing.T) {
	people := []person{{"ann", 30}, {"bob", 25}, {"cid", 40}}

	tests := []struct {
		name      string
		people    []person
		useMax    bool
		wantName  string
		wantValid bool
	}{
		{"min picks smallest key", people, false, "bob", true},
		{"max picks largest key", people, true, "cid", true},
		{"min of empty is none", nil, false, "", false},
		{"max of empty is none", nil, true, "", false},
		{"min of single element", people[:1], false, "ann", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := func(p person) int { return p.age }
			stream := fgo.ToStream(tc.people)

			var got fgo.Option[person]
			if tc.useMax {
				got = stream.MaxBy(key)
			} else {
				got = stream.MinBy(key)
			}

			value, valid := got.Get()
			if valid != tc.wantValid || value.name != tc.wantName {
				t.Errorf("got (%q, %t), want (%q, %t)", value.name, valid, tc.wantName, tc.wantValid)
			}
		})
	}
}

func TestStreamMinByMaxByKeepFirstOnTie(t *testing.T) {
	people := []person{{"first", 30}, {"second", 30}}
	key := func(p person) int { return p.age }

	if got, _ := fgo.ToStream(people).MinBy(key).Get(); got.name != "first" {
		t.Errorf("MinBy tie = %q, want %q", got.name, "first")
	}
	if got, _ := fgo.ToStream(people).MaxBy(key).Get(); got.name != "first" {
		t.Errorf("MaxBy tie = %q, want %q", got.name, "first")
	}
}

func TestStreamMinByMaxByNaNKeys(t *testing.T) {
	// cmp.Less orders NaN below every other float, unlike the < operator.
	items := []float64{3, math.NaN(), 1}
	key := func(v float64) float64 { return v }

	minValue, _ := fgo.ToStream(items).MinBy(key).Get()
	if !math.IsNaN(minValue) {
		t.Errorf("MinBy() = %v, want NaN", minValue)
	}

	maxValue, _ := fgo.ToStream(items).MaxBy(key).Get()
	if maxValue != 3 {
		t.Errorf("MaxBy() = %v, want 3", maxValue)
	}
}

func TestStreamGroupBy(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  map[int][]string
	}{
		{
			"groups by key",
			[]string{"a", "bb", "c", "dd"},
			map[int][]string{1: {"a", "c"}, 2: {"bb", "dd"}},
		},
		{"empty yields empty map", nil, map[int][]string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).GroupBy(func(s string) int { return len(s) })
			if len(got) != len(tc.want) {
				t.Fatalf("GroupBy() = %v, want %v", got, tc.want)
			}
			for k, wantGroup := range tc.want {
				if !slices.Equal(got[k], wantGroup) {
					t.Errorf("GroupBy()[%d] = %v, want %v", k, got[k], wantGroup)
				}
			}
		})
	}
}

func TestStreamSortFunc(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []int
	}{
		{"sorts ascending", []int{3, 1, 2}, []int{1, 2, 3}},
		{"already sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.items).SortFunc(func(a, b int) int { return a - b }).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("SortFunc().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStreamSortBy(t *testing.T) {
	tests := []struct {
		name   string
		people []person
		want   []string
	}{
		{
			"sorts by extracted key",
			[]person{{"ann", 30}, {"bob", 25}, {"cid", 40}},
			[]string{"bob", "ann", "cid"},
		},
		{"empty stays empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.ToStream(tc.people).
				SortBy(func(p person) int { return p.age }).
				Map(func(p person) string { return p.name }).
				Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("SortBy() names = %v, want %v", got, tc.want)
			}
		})
	}
}

// Documents current behaviour: whether a stream survives a second pass depends
// on how it was built, which is not visible from its type.
func TestStreamReuseSemantics(t *testing.T) {
	tests := []struct {
		name       string
		build      func() *fgo.Stream[int]
		wantFirst  []int
		wantSecond []int
	}{
		{
			"slice backed stream is repeatable",
			func() *fgo.Stream[int] {
				return fgo.ToStream([]int{1, 2, 3}).Map(func(v int) int { return v * 2 })
			},
			[]int{2, 4, 6}, []int{2, 4, 6},
		},
		{
			"single use source is exhausted after one pass",
			func() *fgo.Stream[int] { return fgo.FromSeq(onceSeq([]int{1, 2, 3})) },
			[]int{1, 2, 3}, nil,
		},
		{
			"sorting launders a single use source into a repeatable one",
			func() *fgo.Stream[int] {
				return fgo.FromSeq(onceSeq([]int{3, 1, 2})).SortBy(func(v int) int { return v })
			},
			[]int{1, 2, 3}, []int{1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stream := tc.build()
			if got := stream.Collect(); !slices.Equal(got, tc.wantFirst) {
				t.Errorf("first Collect() = %v, want %v", got, tc.wantFirst)
			}
			if got := stream.Collect(); !slices.Equal(got, tc.wantSecond) {
				t.Errorf("second Collect() = %v, want %v", got, tc.wantSecond)
			}
		})
	}
}

func TestStreamChaining(t *testing.T) {
	people := []person{
		{"ann", 30}, {"bob", 25}, {"cid", 40}, {"dan", 25}, {"eve", 35},
	}

	got := fgo.ToStream(people).
		Filter(func(p person) bool { return p.age >= 25 }).
		DistinctBy(func(p person) int { return p.age }).
		SortBy(func(p person) int { return p.age }).
		Take(2).
		Map(func(p person) string { return p.name }).
		Collect()

	if want := []string{"bob", "ann"}; !slices.Equal(got, want) {
		t.Errorf("chain = %v, want %v", got, want)
	}
}
