package fgo

import "encoding/json"

// MarshalJSON encodes o as its contained value, or as JSON null when it holds
// nothing. It implements [encoding/json.Marshaler].
//
// The method is declared on the value receiver, so an Option works as a struct
// field without needing a pointer.
//
// Note that encoding/json's omitempty option has no effect on a struct, so an
// absent value still appears as null. Use omitzero instead, which consults
// [Option.IsZero] and leaves the field out entirely.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

// UnmarshalJSON decodes JSON null as an absent value and anything else as a
// present one. It implements [encoding/json.Unmarshaler].
//
// A missing field leaves o untouched rather than clearing it, which is how
// encoding/json behaves for any type, so decoding into a reused Option should
// start from its zero value.
//
// The round trip is lossy for a value type that itself marshals to null, such as
// an Option[*int] holding a nil pointer: it encodes as null and decodes back as
// absent.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*o = Some(value)
	return nil
}

// IsZero reports whether o holds nothing. It is the negation of [Option.Exists],
// and carries this name because it is what encoding/json's omitzero option looks
// for when deciding to leave a field out:
//
//	type user struct {
//		Nickname Option[string] `json:"nickname,omitzero"`
//	}
func (o Option[T]) IsZero() bool {
	return !o.valid
}
