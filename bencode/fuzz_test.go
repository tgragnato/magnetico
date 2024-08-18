package bencode

import (
	"reflect"
	"testing"
)

func Fuzz(f *testing.F) {
	for _, ret := range random_encode_tests {
		f.Add([]byte(ret.expected))
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		t.Parallel()

		var d interface{}
		err := Unmarshal(b, &d)
		if err != nil {
			t.Skip(err)
		}
		b0, err := Marshal(d)
		if err != nil {
			t.Errorf("Marshal error: %v", err)
		}
		var d0 interface{}
		err = Unmarshal(b0, &d0)
		if err != nil {
			t.Errorf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(d0, d) {
			t.Errorf("Unmarshaled value does not match original value")
		}
	})
}

func FuzzInterfaceRoundTrip(f *testing.F) {
	for _, ret := range random_encode_tests {
		f.Add([]byte(ret.expected))
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		t.Parallel()

		var d interface{}
		err := Unmarshal(b, &d)
		if err != nil {
			t.Skip()
		}
		b0, err := Marshal(d)
		if err != nil {
			t.Errorf("Marshal error: %v", err)
		}
		var d0 interface{}
		err = Unmarshal(b0, &d0)
		if err != nil {
			t.Errorf("Unmarshal error: %v", err)
		}
		if !reflect.DeepEqual(d0, d) {
			t.Errorf("Unmarshaled value does not match original value")
		}
	})
}
