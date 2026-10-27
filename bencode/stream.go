package bencode

import (
	"fmt"
	"io"
	"math"
	"reflect"
)

// Keys up to this length are read with a single exact allocation.
const smallKeyLen = 512

// StreamUnmarshaler is an optional alternative to Unmarshaler. The Decoder hands the type a Stream
// positioned at the start of its value, so it can be decoded in place without first buffering the
// whole value. It takes precedence over Unmarshaler.
type StreamUnmarshaler interface {
	UnmarshalBencodeStream(*Stream) error
}

// Stream gives a StreamUnmarshaler error-returning, reflection-free access to the Decoder input.
type Stream struct {
	d *Decoder
}

// EnterDict consumes the start of a dict.
func (s *Stream) EnterDict() error {
	offset := s.d.Offset
	b, err := s.d.nextByte()
	if err != nil {
		return err
	}
	if b != 'd' {
		return &SyntaxError{Offset: offset, What: fmt.Errorf("expected dict, got %+q", b)}
	}
	return nil
}

// NextKey reads the next key of the current dict. When the dict ends the end marker is consumed
// and ok is false.
func (s *Stream) NextKey() (key string, ok bool, err error) {
	d := s.d
	first, err := d.nextByte()
	if err != nil {
		return "", false, err
	}
	if first == 'e' {
		return "", false, nil
	}
	length, err := d.readStringLength(first)
	if err != nil {
		return "", false, err
	}
	// Small keys skip ReadAll's 512-byte initial buffer; large declared lengths stay incremental.
	if length <= smallKeyLen {
		buf := make([]byte, length)
		n, err := io.ReadFull(d.r, buf)
		d.Offset += int64(n)
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return "", false, err
		}
		return string(buf), true, nil
	}
	b, err := io.ReadAll(io.LimitReader(d.r, int64(length)))
	d.Offset += int64(len(b))
	if err != nil {
		return "", false, err
	}
	if len(b) != length {
		return "", false, io.ErrUnexpectedEOF
	}
	return string(b), true, nil
}

// Decode decodes the next value into v, as Decoder.Decode does.
func (s *Stream) Decode(v any) error {
	return s.d.Decode(v)
}

func (d *Decoder) nextByte() (byte, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	d.Offset++
	return b, nil
}

// Reads a string length whose first digit was already consumed, up to and including the ':'.
func (d *Decoder) readStringLength(first byte) (int, error) {
	start := d.Offset - 1
	if first < '0' || first > '9' {
		return 0, &SyntaxError{Offset: start, What: fmt.Errorf("expected string, got %+q", first)}
	}
	maxLen := d.getMaxStrLen()
	n := int64(first - '0')
	for {
		b, err := d.nextByte()
		if err != nil {
			return 0, err
		}
		if b == ':' {
			if n > maxLen {
				return 0, &SyntaxError{Offset: start, What: fmt.Errorf("string length exceeds limit (%v)", maxLen)}
			}
			return int(n), nil
		}
		if b < '0' || b > '9' || first == '0' {
			return 0, &SyntaxError{Offset: d.Offset - 1, What: fmt.Errorf("invalid string length byte %+q", b)}
		}
		if n > (math.MaxInt64-9)/10 {
			return 0, &SyntaxError{Offset: start, What: fmt.Errorf("string length overflows")}
		}
		n = n*10 + int64(b-'0')
		if n > maxLen {
			return 0, &SyntaxError{Offset: start, What: fmt.Errorf("string length exceeds limit (%v)", maxLen)}
		}
	}
}

func asStreamUnmarshaler(v reflect.Value) (StreamUnmarshaler, bool) {
	if v.CanInterface() {
		if s, ok := v.Interface().(StreamUnmarshaler); ok {
			return s, true
		}
	}
	if v.CanAddr() {
		if s, ok := v.Addr().Interface().(StreamUnmarshaler); ok {
			return s, true
		}
	}
	return nil, false
}

// Same contract as parseValue: false means the end marker was found and nothing was decoded.
func (d *Decoder) parseStream(s StreamUnmarshaler, v reflect.Value) (bool, error) {
	b, err := d.r.ReadByte()
	if err != nil {
		return false, err
	}
	if b == 'e' {
		d.Offset++
		return false, nil
	}
	if err := d.r.UnreadByte(); err != nil {
		return false, err
	}
	if err := s.UnmarshalBencodeStream(&Stream{d}); err != nil {
		return false, &UnmarshalerError{v.Type(), err}
	}
	return true, nil
}
