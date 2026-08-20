package fgo_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/SeaRoll/fgo"
)

var errOther = errors.New("other error")

func TestResultFromTuple(t *testing.T) {
	tests := []struct {
		name      string
		val       int
		err       error
		wantValue int
		wantErr   error
	}{
		{"nil error keeps value", 7, nil, 7, nil},
		// Unlike OptionFromTuple, a non-nil error discards the value.
		{"non-nil error discards value", 7, errTest, 0, errTest},
		{"nil error with zero value", 0, nil, 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := fgo.ResultFromTuple(tc.val, tc.err).Get()
			if value != tc.wantValue {
				t.Errorf("ResultFromTuple(%d, %v) value = %d, want %d", tc.val, tc.err, value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ResultFromTuple(%d, %v) err = %v, want %v", tc.val, tc.err, err, tc.wantErr)
			}
		})
	}
}

func TestResultGet(t *testing.T) {
	var zeroValue fgo.Result[int]

	tests := []struct {
		name      string
		result    fgo.Result[int]
		wantValue int
		wantErr   error
	}{
		{"ok", fgo.Ok(42), 42, nil},
		{"err", fgo.Err[int](errTest), 0, errTest},
		{"ok of zero is not err", fgo.Ok(0), 0, nil},
		{"zero value result behaves as ok of zero", zeroValue, 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.result.Get()
			if value != tc.wantValue {
				t.Errorf("Get() value = %d, want %d", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Get() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestResultErr(t *testing.T) {
	tests := []struct {
		name    string
		result  fgo.Result[string]
		wantErr error
	}{
		{"ok has nil error", fgo.Ok("x"), nil},
		{"err exposes the error", fgo.Err[string](errTest), errTest},
		{"ok of empty string has nil error", fgo.Ok(""), nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.Err(); !errors.Is(got, tc.wantErr) {
				t.Errorf("Err() = %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestResultUnwrapOr(t *testing.T) {
	tests := []struct {
		name     string
		result   fgo.Result[int]
		fallback int
		want     int
	}{
		{"ok returns value", fgo.Ok(3), 99, 3},
		{"err returns fallback", fgo.Err[int](errTest), 99, 99},
		{"ok of zero returns zero not fallback", fgo.Ok(0), 99, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.UnwrapOr(tc.fallback); got != tc.want {
				t.Errorf("UnwrapOr(%d) = %d, want %d", tc.fallback, got, tc.want)
			}
		})
	}
}

func TestResultUnwrapOrElse(t *testing.T) {
	tests := []struct {
		name        string
		result      fgo.Result[int]
		want        int
		wantCalls   int
		wantSeenErr error
	}{
		{"ok skips fallback", fgo.Ok(3), 3, 0, nil},
		{"err passes error to fallback", fgo.Err[int](errTest), 99, 1, errTest},
		{"ok of zero skips fallback", fgo.Ok(0), 0, 0, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var seenErr error
			got := tc.result.UnwrapOrElse(func(err error) int {
				calls++
				seenErr = err
				return 99
			})
			if got != tc.want {
				t.Errorf("UnwrapOrElse() = %d, want %d", got, tc.want)
			}
			if calls != tc.wantCalls {
				t.Errorf("fallback called %d times, want %d", calls, tc.wantCalls)
			}
			if !errors.Is(seenErr, tc.wantSeenErr) {
				t.Errorf("fallback received %v, want %v", seenErr, tc.wantSeenErr)
			}
		})
	}
}

func TestResultMap(t *testing.T) {
	tests := []struct {
		name      string
		result    fgo.Result[int]
		wantValue string
		wantErr   error
		wantCalls int
	}{
		{"ok applies transform", fgo.Ok(12), "12", nil, 1},
		// The original error survives the change of type parameter.
		{"err skips transform and keeps error", fgo.Err[int](errTest), "", errTest, 0},
		{"ok of zero applies transform", fgo.Ok(0), "0", nil, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			value, err := tc.result.Map(func(v int) string {
				calls++
				return strconv.Itoa(v)
			}).Get()
			if value != tc.wantValue {
				t.Errorf("Map() value = %q, want %q", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Map() err = %v, want %v", err, tc.wantErr)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestResultTryMap(t *testing.T) {
	tests := []struct {
		name        string
		result      fgo.Result[int]
		returnValue string
		returnErr   error
		wantValue   string
		wantErr     error
		wantCalls   int
	}{
		{"ok and transform succeeds", fgo.Ok(5), "5", nil, "5", nil, 1},
		// A failing transform discards whatever value it returned alongside the error.
		{"ok and transform fails", fgo.Ok(5), "discarded", errOther, "", errOther, 1},
		{"err short circuits", fgo.Err[int](errTest), "unused", nil, "", errTest, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			value, err := tc.result.TryMap(func(int) (string, error) {
				calls++
				return tc.returnValue, tc.returnErr
			}).Get()
			if value != tc.wantValue {
				t.Errorf("TryMap() value = %q, want %q", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("TryMap() err = %v, want %v", err, tc.wantErr)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestResultFlatMap(t *testing.T) {
	tests := []struct {
		name      string
		result    fgo.Result[int]
		returns   fgo.Result[string]
		wantValue string
		wantErr   error
		wantCalls int
	}{
		{"ok to ok", fgo.Ok(1), fgo.Ok("one"), "one", nil, 1},
		{"ok to err", fgo.Ok(1), fgo.Err[string](errOther), "", errOther, 1},
		{"err short circuits", fgo.Err[int](errTest), fgo.Ok("unused"), "", errTest, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			value, err := tc.result.FlatMap(func(int) fgo.Result[string] {
				calls++
				return tc.returns
			}).Get()
			if value != tc.wantValue {
				t.Errorf("FlatMap() value = %q, want %q", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("FlatMap() err = %v, want %v", err, tc.wantErr)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestResultMapErr(t *testing.T) {
	tests := []struct {
		name      string
		result    fgo.Result[int]
		wantValue int
		wantErr   error
		wantMsg   string
		wantCalls int
	}{
		// Wrapping with %w must keep errors.Is working against the original.
		{"err is wrapped", fgo.Err[int](errTest), 0, errTest, "wrapped: test error", 1},
		{"ok is untouched", fgo.Ok(4), 4, nil, "", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			value, err := tc.result.MapErr(func(err error) error {
				calls++
				return fmt.Errorf("wrapped: %w", err)
			}).Get()
			if value != tc.wantValue {
				t.Errorf("MapErr() value = %d, want %d", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("MapErr() err = %v, want it to match %v", err, tc.wantErr)
			}
			if tc.wantMsg != "" && err.Error() != tc.wantMsg {
				t.Errorf("MapErr() err.Error() = %q, want %q", err.Error(), tc.wantMsg)
			}
			if calls != tc.wantCalls {
				t.Errorf("transform called %d times, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func TestResultOk(t *testing.T) {
	tests := []struct {
		name      string
		result    fgo.Result[int]
		wantValue int
		wantValid bool
	}{
		{"ok becomes some", fgo.Ok(8), 8, true},
		// The error is dropped on the way to Option.
		{"err becomes none", fgo.Err[int](errTest), 0, false},
		{"ok of zero becomes some of zero", fgo.Ok(0), 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, valid := tc.result.Ok().Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Ok().Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

func TestResultChaining(t *testing.T) {
	tests := []struct {
		name      string
		result    fgo.Result[string]
		wantValue int
		wantErr   error
	}{
		{"parses and doubles", fgo.Ok("21"), 42, nil},
		{"propagates parse failure", fgo.Ok("nope"), 0, strconv.ErrSyntax},
		{"propagates original error", fgo.Err[string](errTest), 0, errTest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.result.
				TryMap(strconv.Atoi).
				Map(func(v int) int { return v * 2 }).
				MapErr(func(err error) error { return fmt.Errorf("chain: %w", err) }).
				Get()
			if value != tc.wantValue {
				t.Errorf("chain value = %d, want %d", value, tc.wantValue)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("chain err = %v, want it to match %v", err, tc.wantErr)
			}
		})
	}
}
