package bencode

import (
	"errors"
	"fmt"
)

type Bytes []byte

var (
	_ Unmarshaler = (*Bytes)(nil)
	_ Marshaler   = (*Bytes)(nil)
	_ Marshaler   = Bytes{}
)

// Unmarshals the provided byte slice into the Bytes receiver.
func (me *Bytes) UnmarshalBencode(b []byte) error {
	*me = append([]byte(nil), b...)
	return nil
}

// Marshals the Bytes receiver into a byte slice.
func (me Bytes) MarshalBencode() ([]byte, error) {
	if len(me) == 0 {
		return nil, errors.New("marshalled Bytes should not be zero-length")
	}
	return me, nil
}

// Returns a Go-syntax string representation of the Bytes receiver.
func (me Bytes) GoString() string {
	return fmt.Sprintf("bencode.Bytes(%q)", []byte(me))
}
