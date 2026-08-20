package fgo_test

import (
	"errors"
	"iter"
	"slices"
	"strconv"
	"testing"

	"github.com/SeaRoll/fgo"
)

// countingResultSeq mirrors countingSeq for Result elements, so CollectResult's
// fail-fast behaviour can be asserted by how much of the source it consumed.
func countingResultSeq(items []fgo.Result[int], pulled *int) iter.Seq[fgo.Result[int]] {
	return func(yield func(fgo.Result[int]) bool) {
		for _, v := range items {
			*pulled++
			if !yield(v) {
				return
			}
		}
	}
}

func TestSorted(t *testing.T) {
	tests := []struct {
		name  string
		items []int
		want  []int
	}{
		{"unsorted", []int{3, 1, 2}, []int{1, 2, 3}},
		{"already sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"reverse sorted", []int{3, 2, 1}, []int{1, 2, 3}},
		{"keeps duplicates", []int{2, 1, 2, 1}, []int{1, 1, 2, 2}},
		{"negative values", []int{0, -2, 5, -1}, []int{-2, -1, 0, 5}},
		{"single item", []int{7}, []int{7}},
		{"empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.Sorted(fgo.ToStream(tc.items)).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Sorted().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSortedStrings(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{"lexicographic order", []string{"pear", "apple", "fig"}, []string{"apple", "fig", "pear"}},
		{"empty string sorts first", []string{"b", "", "a"}, []string{"", "a", "b"}},
		{"uppercase before lowercase", []string{"b", "A"}, []string{"A", "b"}},
		{"empty", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fgo.Sorted(fgo.ToStream(tc.items)).Collect()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Sorted().Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

// Sorted must drain its source to sort, so it always hands back a slice-backed
// stream that survives a second pass.
func TestSortedIsRepeatable(t *testing.T) {
	tests := []struct {
		name  string
		build func() *fgo.Stream[int]
		want  []int
	}{
		{
			"slice backed source",
			func() *fgo.Stream[int] { return fgo.ToStream([]int{3, 1, 2}) },
			[]int{1, 2, 3},
		},
		{
			"single use source",
			func() *fgo.Stream[int] { return fgo.FromSeq(onceSeq([]int{3, 1, 2})) },
			[]int{1, 2, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stream := fgo.Sorted(tc.build())
			if got := stream.Collect(); !slices.Equal(got, tc.want) {
				t.Errorf("first Collect() = %v, want %v", got, tc.want)
			}
			if got := stream.Collect(); !slices.Equal(got, tc.want) {
				t.Errorf("second Collect() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCollectResult(t *testing.T) {
	tests := []struct {
		name       string
		items      []fgo.Result[int]
		wantValues []int
		wantErr    error
		wantPulled int
	}{
		{
			"all ok collects every value",
			[]fgo.Result[int]{fgo.Ok(1), fgo.Ok(2), fgo.Ok(3)},
			[]int{1, 2, 3}, nil, 3,
		},
		{
			// Values seen before the error are discarded, not returned partially.
			"first element errors and stops immediately",
			[]fgo.Result[int]{fgo.Err[int](errTest), fgo.Ok(2), fgo.Ok(3)},
			nil, errTest, 1,
		},
		{
			"middle element errors and stops there",
			[]fgo.Result[int]{fgo.Ok(1), fgo.Err[int](errTest), fgo.Ok(3)},
			nil, errTest, 2,
		},
		{
			"last element errors after full consumption",
			[]fgo.Result[int]{fgo.Ok(1), fgo.Ok(2), fgo.Err[int](errTest)},
			nil, errTest, 3,
		},
		{
			// Fail-fast means the earliest error wins, not the last.
			"earliest of several errors wins",
			[]fgo.Result[int]{fgo.Ok(1), fgo.Err[int](errTest), fgo.Err[int](errOther)},
			nil, errTest, 2,
		},
		{
			"empty is ok with no items",
			nil,
			nil, nil, 0,
		},
		{
			"ok of zero values is not an error",
			[]fgo.Result[int]{fgo.Ok(0), fgo.Ok(0)},
			[]int{0, 0}, nil, 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pulled := 0
			stream := fgo.FromSeq(countingResultSeq(tc.items, &pulled))

			values, err := fgo.CollectResult(stream).Tuple()
			if !slices.Equal(values, tc.wantValues) {
				t.Errorf("CollectResult() values = %v, want %v", values, tc.wantValues)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("CollectResult() err = %v, want %v", err, tc.wantErr)
			}
			if pulled != tc.wantPulled {
				t.Errorf("pulled %d elements, want %d", pulled, tc.wantPulled)
			}
		})
	}
}

func TestCollectResultFromMappedStream(t *testing.T) {
	tests := []struct {
		name       string
		items      []string
		wantValues []int
		wantErr    error
	}{
		{"all parse", []string{"1", "2", "3"}, []int{1, 2, 3}, nil},
		{"one fails", []string{"1", "nope", "3"}, nil, errTest},
		{"empty", nil, nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := fgo.ToStream(tc.items).Map(func(s string) fgo.Result[int] {
				n, err := strconv.Atoi(s)
				if err != nil {
					return fgo.Err[int](errTest)
				}
				return fgo.Ok(n)
			})

			values, err := fgo.CollectResult(results).Tuple()
			if !slices.Equal(values, tc.wantValues) {
				t.Errorf("CollectResult() values = %v, want %v", values, tc.wantValues)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("CollectResult() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}
