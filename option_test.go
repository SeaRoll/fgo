package fgo_test

import (
	"errors"
	"strconv"
	"testing"

	"github.com/SeaRoll/fgo"
)

var errTest = errors.New("test error")

func TestOptionFromTuple(t *testing.T) {
	tests := []struct {
		name      string
		val       int
		ok        bool
		wantValue int
		wantValid bool
	}{
		{"ok keeps value", 7, true, 7, true},
		{"not ok discards value", 7, false, 7, false},
		{"ok with zero value", 0, true, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := fgo.OptionFromTuple(tc.val, tc.ok).Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("OptionFromTuple(%d, %t).Get() = (%d, %t), want (%d, %t)",
					tc.val, tc.ok, value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

func TestOptionGet(t *testing.T) {
	var zeroValue fgo.Option[int]

	tests := []struct {
		name      string
		option    fgo.Option[int]
		wantValue int
		wantValid bool
	}{
		{"some", fgo.Some(42), 42, true},
		{"none", fgo.None[int](), 0, false},
		// Some(0) and None[int]() carry the same value; only validity separates them.
		{"some of zero is not none", fgo.Some(0), 0, true},
		{"zero value option behaves as none", zeroValue, 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := tc.option.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

// A falsy value is still a present value; only Get can tell the two apart.
func TestOptionGetBool(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[bool]
		wantValue bool
		wantValid bool
	}{
		{"some of false", fgo.Some(false), false, true},
		{"some of true", fgo.Some(true), true, true},
		{"none", fgo.None[bool](), false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := tc.option.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Get() = (%t, %t), want (%t, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

func TestOptionExists(t *testing.T) {
	tests := []struct {
		name   string
		option fgo.Option[string]
		want   bool
	}{
		{"some", fgo.Some("x"), true},
		{"some of empty string", fgo.Some(""), true},
		{"none", fgo.None[string](), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.option.Exists(); got != tc.want {
				t.Errorf("Exists() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestOptionUnwrapOr(t *testing.T) {
	tests := []struct {
		name     string
		option   fgo.Option[int]
		fallback int
		want     int
	}{
		{"some returns value", fgo.Some(3), 99, 3},
		{"none returns fallback", fgo.None[int](), 99, 99},
		{"some of zero returns zero not fallback", fgo.Some(0), 99, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.option.UnwrapOr(tc.fallback); got != tc.want {
				t.Errorf("UnwrapOr(%d) = %d, want %d", tc.fallback, got, tc.want)
			}
		})
	}
}

func TestOptionUnwrapOrElse(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		want      int
		wantCalls int
	}{
		{"some skips fallback", fgo.Some(3), 3, 0},
		{"none evaluates fallback", fgo.None[int](), 99, 1},
		{"some of zero skips fallback", fgo.Some(0), 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := tc.option.UnwrapOrElse(func() int {
				calls++
				return 99
			})
			if got != tc.want {
				t.Errorf("UnwrapOrElse() = %d, want %d", got, tc.want)
			}
			if calls != tc.wantCalls {
				t.Errorf("fallback called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestOptionMap(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		wantValue string
		wantValid bool
		wantCalls int
	}{
		{"some applies transform", fgo.Some(12), "12", true, 1},
		{"none skips transform", fgo.None[int](), "", false, 0},
		{"some of zero applies transform", fgo.Some(0), "0", true, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := tc.option.Map(func(v int) string {
				calls++
				return strconv.Itoa(v)
			})
			value, valid := got.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Map() = (%q, %t), want (%q, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestOptionFlatMap(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		returns   fgo.Option[string]
		wantValue string
		wantValid bool
		wantCalls int
	}{
		{"some to some", fgo.Some(1), fgo.Some("one"), "one", true, 1},
		{"some to none", fgo.Some(1), fgo.None[string](), "", false, 1},
		{"none short circuits", fgo.None[int](), fgo.Some("unused"), "", false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := tc.option.FlatMap(func(int) fgo.Option[string] {
				calls++
				return tc.returns
			})
			value, valid := got.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("FlatMap() = (%q, %t), want (%q, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestOptionFilter(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		keep      bool
		wantValue int
		wantValid bool
		wantCalls int
	}{
		{"some kept", fgo.Some(5), true, 5, true, 1},
		{"some dropped", fgo.Some(5), false, 0, false, 1},
		{"none skips predicate", fgo.None[int](), true, 0, false, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			got := tc.option.Filter(func(int) bool {
				calls++
				return tc.keep
			})
			value, valid := got.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Filter() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
			if calls != tc.wantCalls {
				t.Errorf("predicate called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestOptionOkOr(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		err       error
		wantValue int
		wantErr   error
	}{
		{"some becomes ok", fgo.Some(8), errTest, 8, nil},
		{"none becomes err", fgo.None[int](), errTest, 0, errTest},
		{"some of zero becomes ok", fgo.Some(0), errTest, 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.option.OkOr(tc.err).Get()
			if value != tc.wantValue {
				t.Errorf("OkOr().Get() value = %d, want %d", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("OkOr().Get() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestOptionChaining(t *testing.T) {
	tests := []struct {
		name      string
		option    fgo.Option[int]
		wantValue string
		wantValid bool
	}{
		{"passes filter", fgo.Some(4), "8", true},
		{"fails filter", fgo.Some(3), "", false},
		{"starts none", fgo.None[int](), "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := tc.option.
				Filter(func(v int) bool { return v%2 == 0 }).
				Map(func(v int) int { return v * 2 }).
				FlatMap(func(v int) fgo.Option[string] { return fgo.Some(strconv.Itoa(v)) }).
				Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("chain = (%q, %t), want (%q, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}
