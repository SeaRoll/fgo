package fgo_test

import (
	"encoding/json"
	"testing"

	"github.com/SeaRoll/fgo"
)

type profile struct {
	Name     string             `json:"name"`
	Nickname fgo.Option[string] `json:"nickname"`
	Age      fgo.Option[int]    `json:"age"`
}

type sparseProfile struct {
	Name     string             `json:"name"`
	Nickname fgo.Option[string] `json:"nickname,omitzero"`
	Age      fgo.Option[int]    `json:"age,omitzero"`
}

func TestOptionMarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		option fgo.Option[int]
		want   string
	}{
		{"some encodes the value", fgo.Some(42), "42"},
		{"some of zero encodes as zero, not null", fgo.Some(0), "0"},
		{"some of negative", fgo.Some(-7), "-7"},
		{"none encodes as null", fgo.None[int](), "null"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.option)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestOptionMarshalJSONOtherTypes(t *testing.T) {
	tests := []struct {
		name   string
		option any
		want   string
	}{
		{"some string", fgo.Some("hi"), `"hi"`},
		{"some empty string is not null", fgo.Some(""), `""`},
		{"none string", fgo.None[string](), "null"},
		{"some false is not null", fgo.Some(false), "false"},
		{"some slice", fgo.Some([]int{1, 2}), "[1,2]"},
		{"none slice", fgo.None[[]int](), "null"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.option)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestOptionUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue int
		wantValid bool
		wantErr   bool
	}{
		{"number decodes as present", "42", 42, true, false},
		{"zero decodes as present", "0", 0, true, false},
		{"null decodes as absent", "null", 0, false, false},
		{"string is a type error", `"abc"`, 0, false, true},
		{"object is a type error", "{}", 0, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got fgo.Option[int]
			err := json.Unmarshal([]byte(tc.input), &got)

			if (err != nil) != tc.wantErr {
				t.Fatalf("Unmarshal(%s) error = %v, wantErr %t", tc.input, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			value, valid := got.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

func TestOptionUnmarshalJSONOverwritesPriorValue(t *testing.T) {
	tests := []struct {
		name      string
		start     fgo.Option[int]
		input     string
		wantValue int
		wantValid bool
	}{
		{"null clears a present value", fgo.Some(1), "null", 0, false},
		{"value replaces a present value", fgo.Some(1), "2", 2, true},
		{"value fills an absent one", fgo.None[int](), "3", 3, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.start
			if err := json.Unmarshal([]byte(tc.input), &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			value, valid := got.Get()
			if value != tc.wantValue || valid != tc.wantValid {
				t.Errorf("Get() = (%d, %t), want (%d, %t)", value, valid, tc.wantValue, tc.wantValid)
			}
		})
	}
}

func TestOptionIsZero(t *testing.T) {
	var unset fgo.Option[int]

	tests := []struct {
		name   string
		option fgo.Option[int]
		want   bool
	}{
		{"some is not zero", fgo.Some(1), false},
		{"some of zero is not zero", fgo.Some(0), false},
		{"none is zero", fgo.None[int](), true},
		{"zero value is zero", unset, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.option.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestOptionStructMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value profile
		want  string
	}{
		{
			"all fields present",
			profile{Name: "ann", Nickname: fgo.Some("annie"), Age: fgo.Some(30)},
			`{"name":"ann","nickname":"annie","age":30}`,
		},
		{
			// Absent fields still appear, because omitempty does not apply to structs.
			"absent fields become null",
			profile{Name: "bob", Nickname: fgo.None[string](), Age: fgo.None[int]()},
			`{"name":"bob","nickname":null,"age":null}`,
		},
		{
			"unset fields behave as absent",
			profile{Name: "cid"},
			`{"name":"cid","nickname":null,"age":null}`,
		},
		{
			"present but falsy fields are kept",
			profile{Name: "dan", Nickname: fgo.Some(""), Age: fgo.Some(0)},
			`{"name":"dan","nickname":"","age":0}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestOptionStructMarshalJSONOmitzero(t *testing.T) {
	tests := []struct {
		name  string
		value sparseProfile
		want  string
	}{
		{
			"present fields are kept",
			sparseProfile{Name: "ann", Nickname: fgo.Some("annie"), Age: fgo.Some(30)},
			`{"name":"ann","nickname":"annie","age":30}`,
		},
		{
			// omitzero consults IsZero, so absent fields disappear entirely.
			"absent fields are omitted",
			sparseProfile{Name: "bob"},
			`{"name":"bob"}`,
		},
		{
			"present but falsy fields survive omitzero",
			sparseProfile{Name: "cid", Nickname: fgo.Some(""), Age: fgo.Some(0)},
			`{"name":"cid","nickname":"","age":0}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("Marshal() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestOptionStructUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantNickname string
		wantHasNick  bool
		wantAge      int
		wantHasAge   bool
	}{
		{
			"all fields present",
			`{"name":"ann","nickname":"annie","age":30}`,
			"annie", true, 30, true,
		},
		{
			"explicit nulls decode as absent",
			`{"name":"bob","nickname":null,"age":null}`,
			"", false, 0, false,
		},
		{
			// A missing key never calls UnmarshalJSON, leaving the zero Option.
			"missing keys decode as absent",
			`{"name":"cid"}`,
			"", false, 0, false,
		},
		{
			"falsy values decode as present",
			`{"name":"dan","nickname":"","age":0}`,
			"", true, 0, true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got profile
			if err := json.Unmarshal([]byte(tc.input), &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			nickname, hasNick := got.Nickname.Get()
			if nickname != tc.wantNickname || hasNick != tc.wantHasNick {
				t.Errorf("Nickname.Get() = (%q, %t), want (%q, %t)",
					nickname, hasNick, tc.wantNickname, tc.wantHasNick)
			}

			age, hasAge := got.Age.Get()
			if age != tc.wantAge || hasAge != tc.wantHasAge {
				t.Errorf("Age.Get() = (%d, %t), want (%d, %t)", age, hasAge, tc.wantAge, tc.wantHasAge)
			}
		})
	}
}

func TestOptionJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value profile
	}{
		{"all present", profile{Name: "ann", Nickname: fgo.Some("annie"), Age: fgo.Some(30)}},
		{"all absent", profile{Name: "bob"}},
		{"mixed", profile{Name: "cid", Nickname: fgo.Some("c"), Age: fgo.None[int]()}},
		{"present but falsy", profile{Name: "dan", Nickname: fgo.Some(""), Age: fgo.Some(0)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var decoded profile
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if decoded != tc.value {
				t.Errorf("round trip = %+v, want %+v", decoded, tc.value)
			}
		})
	}
}
