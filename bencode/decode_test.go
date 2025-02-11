package bencode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

type random_decode_test struct {
	data     string
	expected interface{}
}

var random_decode_tests = []random_decode_test{
	{"i57e", int64(57)},
	{"i-9223372036854775808e", int64(-9223372036854775808)},
	{"5:hello", "hello"},
	{"29:unicode test проверка", "unicode test проверка"},
	{"d1:ai5e1:b5:helloe", map[string]interface{}{"a": int64(5), "b": "hello"}},
	{
		"li5ei10ei15ei20e7:bencodee",
		[]interface{}{int64(5), int64(10), int64(15), int64(20), "bencode"},
	},
	{"ldedee", []interface{}{map[string]interface{}{}, map[string]interface{}{}}},
	{"le", []interface{}{}},
	{"i604919719469385652980544193299329427705624352086e", func() *big.Int {
		ret, _ := big.NewInt(-1).SetString("604919719469385652980544193299329427705624352086", 10)
		return ret
	}()},
	{"d1:rd6:\xd4/\xe2F\x00\x01i42ee1:t3:\x9a\x87\x011:v4:TR%=1:y1:re", map[string]interface{}{
		"r": map[string]interface{}{"\xd4/\xe2F\x00\x01": int64(42)},
		"t": "\x9a\x87\x01",
		"v": "TR%=",
		"y": "r",
	}},
	{"d0:i420ee", map[string]interface{}{"": int64(420)}},
}

func TestRandomDecode(t *testing.T) {
	t.Parallel()

	for _, test := range random_decode_tests {
		var value interface{}
		err := Unmarshal([]byte(test.data), &value)
		if err != nil {
			t.Error(err, test.data)
			continue
		}
		if !reflect.DeepEqual(test.expected, value) {
			t.Errorf("Test failed: expected %v, got %v", test.expected, value)
		}
	}
}

func TestLoneE(t *testing.T) {
	t.Parallel()

	var v int
	err := Unmarshal([]byte("e"), &v)
	se := err.(*SyntaxError)
	if se.Offset != 0 {
		t.Errorf("Expected offset of 0, got %d", se.Offset)
	}
}

func TestDecoderConsecutive(t *testing.T) {
	t.Parallel()

	d := NewDecoder(bytes.NewReader([]byte("i1ei2e")))
	var i int
	err := d.Decode(&i)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if i != 1 {
		t.Errorf("Expected value 1 for i, got %v", i)
	}
	err = d.Decode(&i)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if i != 2 {
		t.Errorf("Expected value 2 for i, got %v", i)
	}
	err = d.Decode(&i)
	if err != io.EOF {
		t.Errorf("Test failed: expected EOF, got %v", err)
	}
}

func TestDecoderConsecutiveDicts(t *testing.T) {
	t.Parallel()

	bb := bytes.NewBufferString("d4:herp4:derped3:wat1:ke17:oh baby a triple!")

	d := NewDecoder(bb)
	if bb.String() != "d4:herp4:derped3:wat1:ke17:oh baby a triple!" {
		t.Errorf("Unexpected value for bb.String()")
	}
	if d.Offset != 0 {
		t.Errorf("Unexpected value for d.Offset")
	}

	var m map[string]interface{}

	err := d.Decode(&m)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(m) != 1 {
		t.Errorf("Expected map length of 1, got %d", len(m))
	}
	if m["herp"] != "derp" {
		t.Errorf("Expected value 'derp' for key 'herp', got %v", m["herp"])
	}
	if bb.String() != "d3:wat1:ke17:oh baby a triple!" {
		t.Errorf("Unexpected value for bb.String()")
	}
	if d.Offset != 14 {
		t.Errorf("Expected offset of 14, got %d", d.Offset)
	}

	err = d.Decode(&m)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if m["wat"] != "k" {
		t.Errorf("Expected value 'k' for key 'wat', got %v", m["wat"])
	}
	if bb.String() != "17:oh baby a triple!" {
		t.Errorf("Unexpected value for bb.String()")
	}
	if d.Offset != 24 {
		t.Errorf("Expected offset of 24, got %d", d.Offset)
	}

	var s string
	err = d.Decode(&s)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if s != "oh baby a triple!" {
		t.Errorf("Expected value 'oh baby a triple!', got %v", s)
	}
	if d.Offset != 44 {
		t.Errorf("Expected offset of 44, got %d", d.Offset)
	}
}

func check_error(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func assert_equal(t *testing.T, x, y interface{}) {
	if !reflect.DeepEqual(x, y) {
		t.Errorf("got: %v (%T), expected: %v (%T)\n", x, x, y, y)
	}
}

type unmarshalerInt struct {
	x int
}

func (me *unmarshalerInt) UnmarshalBencode(data []byte) error {
	return Unmarshal(data, &me.x)
}

type unmarshalerString struct {
	x string
}

func (me *unmarshalerString) UnmarshalBencode(data []byte) error {
	me.x = string(data)
	return nil
}

func TestUnmarshalerBencode(t *testing.T) {
	t.Parallel()

	var i unmarshalerInt
	var ss []unmarshalerString
	check_error(t, Unmarshal([]byte("i71e"), &i))
	assert_equal(t, i.x, 71)
	check_error(t, Unmarshal([]byte("l5:hello5:fruit3:waye"), &ss))
	assert_equal(t, ss[0].x, "5:hello")
	assert_equal(t, ss[1].x, "5:fruit")
	assert_equal(t, ss[2].x, "3:way")
}

func TestIgnoreUnmarshalTypeError(t *testing.T) {
	t.Parallel()

	s := struct {
		Ignore int `bencode:",ignore_unmarshal_type_error"`
		Normal int
	}{}
	err := Unmarshal([]byte("d6:Normal5:helloe"), &s)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
	err = Unmarshal([]byte("d6:Ignore5:helloe"), &s)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if err := Unmarshal([]byte("d6:Ignorei42ee"), &s); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if s.Ignore != 42 {
		t.Errorf("Expected value 42 for Ignore, got %v", s.Ignore)
	}
}

// Test unmarshalling []byte into something that has the same kind but
// different type.
func TestDecodeCustomSlice(t *testing.T) {
	t.Parallel()

	type flag byte
	var fs3, fs2 []flag
	// We do a longer slice then a shorter slice to see if the buffers are
	// shared.
	d := NewDecoder(bytes.NewBufferString("3:\x01\x10\xff2:\x04\x0f"))
	err := d.Decode(&fs3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	err = d.Decode(&fs2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual([]flag{1, 16, 255}, fs3) {
		t.Errorf("Expected value %v for fs3, got %v", []flag{1, 16, 255}, fs3)
	}
	if !reflect.DeepEqual([]flag{4, 15}, fs2) {
		t.Errorf("Expected value %v for fs2, got %v", []flag{4, 15}, fs2)
	}
}

func TestUnmarshalUnusedBytes(t *testing.T) {
	t.Parallel()

	var i int
	err := Unmarshal([]byte("i42ee"), &i)
	if err != nil {
		if _, ok := err.(ErrUnusedTrailingBytes); ok {
			if err.(ErrUnusedTrailingBytes).NumUnusedBytes != 1 {
				t.Errorf("Expected 1 unused trailing byte, got %d", err.(ErrUnusedTrailingBytes).NumUnusedBytes)
			}
		} else {
			t.Errorf("Unexpected error: %v", err)
		}
	}
	if i != 42 {
		t.Errorf("Expected value 42 for i, got %v", i)
	}
}

func TestUnmarshalByteArray(t *testing.T) {
	t.Parallel()

	var ba [2]byte
	err := Unmarshal([]byte("2:hi"), &ba)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(ba[:]) != "hi" {
		t.Errorf("Expected value 'hi' for ba, got %s", string(ba[:]))
	}
}

func TestDecodeDictIntoUnsupported(t *testing.T) {
	t.Parallel()

	// Any type that a dict shouldn't be unmarshallable into.
	var i int
	err := Unmarshal([]byte("d1:a1:be"), &i)
	if err == nil {
		t.Errorf("An error was expected")
	}
}

func TestUnmarshalDictKeyNotString(t *testing.T) {
	t.Parallel()

	// Any type that a dict shouldn't be unmarshallable into.
	var i int
	err := Unmarshal([]byte("di42e3:yese"), &i)
	if err == nil {
		t.Errorf("An error was expected")
	}
}

type arbitraryReader struct{}

func (arbitraryReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func decodeHugeString(strLen int64, header, tail string, v interface{}, maxStrLen MaxStrLen) error {
	r, w := io.Pipe()
	go func() {
		fmt.Fprintf(w, header, strLen)
		if _, err := io.CopyN(w, arbitraryReader{}, strLen); err != nil {
			panic(err)
		}
		if _, err := w.Write([]byte(tail)); err != nil {
			panic(err)
		}
		w.Close()
	}()
	d := NewDecoder(r)
	d.MaxStrLen = maxStrLen
	return d.Decode(v)
}

// Ensure that bencode strings in various places obey the Decoder.MaxStrLen field.
func TestDecodeMaxStrLen(t *testing.T) {
	t.Parallel()

	test := func(t *testing.T, header, tail string, v interface{}, maxStrLen MaxStrLen) {
		strLen := maxStrLen
		if strLen == 0 {
			strLen = DefaultDecodeMaxStrLen
		}
		if err := decodeHugeString(strLen, header, tail, v, maxStrLen); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if err := decodeHugeString(strLen+1, header, tail, v, maxStrLen); err == nil {
			t.Errorf("An error was expected")
		}
	}
	test(t, "d%d:", "i0ee", new(interface{}), 0)
	test(t, "%d:", "", new(interface{}), DefaultDecodeMaxStrLen)
	test(t, "%d:", "", new([]byte), 1)
	test(t, "d3:420%d:", "e", new(struct {
		Hi []byte `bencode:"420"`
	}), 69)
}

// This is for the "tgragnato.it/magnetico/metainfo".Info.Private field.
func TestDecodeStringIntoBoolPtr(t *testing.T) {
	t.Parallel()

	var m struct {
		Private *bool `bencode:"private,omitempty"`
	}
	check := func(t *testing.T, msg string, expectNil, expectTrue bool) {
		m.Private = nil
		err := Unmarshal([]byte(msg), &m)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if expectNil {
			if m.Private != nil {
				t.Errorf("Expected nil value for m.Private, got %v", m.Private)
			}
		} else {
			if m.Private == nil {
				t.Errorf("Expected non-nil value for m.Private")
			} else if *m.Private != expectTrue {
				t.Errorf("Expected value %v for m.Private, got %v", expectTrue, *m.Private)
			}
		}
	}
	check(t, "d7:privatei1ee", false, true)
	check(t, "d7:privatei0ee", false, false)
	check(t, "d7:privatei42ee", false, true)
	// This is a weird case. We could not allocate the bool to indicate it was bad (maybe a bad
	// serializer which isn't uncommon), but that requires reworking the decoder to handle
	// automatically. I think if we cared enough we'd create a custom Unmarshaler. Also if we were
	// worried enough about performance I'd completely rewrite this package.
	check(t, "d7:private0:e", false, false)
	check(t, "d7:private1:te", false, true)
	check(t, "d7:private5:falsee", false, false)
	check(t, "d7:private1:Fe", false, false)
	check(t, "d7:private11:bunnyfoofooe", false, true)
}

// To set expectations about how our Decoder should work.
func TestJsonDecoderBehaviour(t *testing.T) {
	t.Parallel()

	test := func(t *testing.T, input string, items int, finalErr error) {
		d := json.NewDecoder(strings.NewReader(input))
		actualItems := 0
		var firstErr error
		for {
			var discard any
			firstErr = d.Decode(&discard)
			if firstErr != nil {
				break
			}
			actualItems++
		}
		if firstErr != finalErr {
			t.Errorf("Expected error %v, got %v", finalErr, firstErr)
		}
		if actualItems != items {
			t.Errorf("Expected %d items, got %d", items, actualItems)
		}
	}
	test(t, "", 0, io.EOF)
	test(t, "{}", 1, io.EOF)
	test(t, "{} {", 1, io.ErrUnexpectedEOF)
}
