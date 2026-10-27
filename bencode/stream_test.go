package bencode

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type streamPairs struct {
	keys []string
	n    int64
}

func (p *streamPairs) UnmarshalBencodeStream(s *Stream) error {
	if err := s.EnterDict(); err != nil {
		return err
	}
	for {
		key, ok, err := s.NextKey()
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		p.keys = append(p.keys, key)
		if err := s.Decode(&p.n); err != nil {
			return err
		}
	}
}

func TestStreamUnmarshaler(t *testing.T) {
	t.Parallel()

	var got struct {
		A streamPairs `bencode:"a"`
		B int         `bencode:"b"`
	}
	if err := Unmarshal([]byte("d1:ad1:xi1e1:yi2ee1:bi3ee"), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.A.keys) != 2 || got.A.keys[0] != "x" || got.A.keys[1] != "y" || got.A.n != 2 || got.B != 3 {
		t.Fatalf("unexpected result %+v", got)
	}
}

func TestStreamUnmarshalerErrors(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "i1e", "d", "d1:x", "d2:xi1ee", "d01:xi1ee", "d1:xi1e", "d1:xi1eee"} {
		if err := Unmarshal([]byte(in), new(streamPairs)); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}

	var ue *UnmarshalerError
	if err := Unmarshal([]byte("i1e"), new(streamPairs)); !errors.As(err, &ue) {
		t.Errorf("expected UnmarshalerError, got %v", err)
	}
}

func TestStreamLengthAndKeyErrors(t *testing.T) {
	t.Parallel()

	for _, in := range []string{
		"dxe",                    // key not a string
		"d1x",                    // invalid length byte
		"d12",                    // EOF inside length
		"d99999999999999999999:", // length overflow
		"d5:ab",                  // truncated key
	} {
		if err := Unmarshal([]byte(in), new(streamPairs)); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}

	dec := NewDecoder(strings.NewReader("d4:abcdi1ee"))
	dec.MaxStrLen = 3
	if err := dec.Decode(new(streamPairs)); err == nil {
		t.Error("expected error for key exceeding MaxStrLen")
	}
}

type streamTwice struct{}

func (streamTwice) UnmarshalBencodeStream(s *Stream) error {
	if err := s.EnterDict(); err != nil {
		return err
	}
	return s.EnterDict()
}

func TestStreamEnterDictEOF(t *testing.T) {
	t.Parallel()

	if err := Unmarshal([]byte("d"), new(streamTwice)); err == nil || !strings.Contains(err.Error(), io.ErrUnexpectedEOF.Error()) {
		t.Errorf("expected unexpected EOF, got %v", err)
	}
}

func TestStreamUnmarshalerInContainers(t *testing.T) {
	t.Parallel()

	var list []streamPairs
	if err := Unmarshal([]byte("ld1:xi1eed1:yi2eee"), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[1].keys[0] != "y" {
		t.Fatalf("unexpected list %+v", list)
	}

	var byValue struct{ A streamPairs }
	if err := Unmarshal([]byte("d1:Ad1:xi1eee"), &byValue); err != nil {
		t.Fatal(err)
	}
}
