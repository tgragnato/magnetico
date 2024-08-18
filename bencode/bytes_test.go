package bencode

import (
	"reflect"
	"testing"
)

func TestBytesMarshalNil(t *testing.T) {
	t.Parallel()

	var b Bytes
	Marshal(b)
}

type structWithBytes struct {
	A Bytes
	B Bytes
}

func TestMarshalNilStructBytes(t *testing.T) {
	t.Parallel()

	t.Run("Nil test", func(t *testing.T) {
		_, err := Marshal(structWithBytes{A: Bytes("i42e")})
		if err == nil {
			t.Error("Marshal was expected to fail")
		}
	})
}

type structWithOmitEmptyBytes struct {
	A Bytes `bencode:",omitempty"`
	B Bytes `bencode:",omitempty"`
}

func TestMarshalNilStructBytesOmitEmpty(t *testing.T) {
	t.Parallel()

	t.Run("Marshal-Unmarshal test", func(t *testing.T) {
		b, err := Marshal(structWithOmitEmptyBytes{A: Bytes("i42e")})
		if err != nil {
			t.Error("Marshal failed with error:", err.Error())
		}
		t.Logf("%q", b)

		var s structWithBytes
		err = Unmarshal(b, &s)
		if err != nil {
			t.Error("Unmarshal failed with error:", err.Error())
		}
		if reflect.DeepEqual(s.B, Bytes("i42e")) {
			t.Error("Unmarshal failed to preserve marshaled bytes")
		}
	})
}
