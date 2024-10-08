package bencode

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

type random_encode_test struct {
	value    interface{}
	expected string
}

type random_struct struct {
	ABC         int    `bencode:"abc"`
	SkipThisOne string `bencode:"-"`
	CDE         string
}

type dummy struct {
	a, b, c int
}

func (d *dummy) MarshalBencode() ([]byte, error) {
	var b bytes.Buffer
	_, err := fmt.Fprintf(&b, "i%dei%dei%de", d.a+1, d.b+1, d.c+1)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

var random_encode_tests = []random_encode_test{
	{int(10), "i10e"},
	{uint(10), "i10e"},
	{"hello, world", "12:hello, world"},
	{true, "i1e"},
	{false, "i0e"},
	{int8(-8), "i-8e"},
	{int16(-16), "i-16e"},
	{int32(32), "i32e"},
	{int64(-64), "i-64e"},
	{uint8(8), "i8e"},
	{uint16(16), "i16e"},
	{uint32(32), "i32e"},
	{uint64(64), "i64e"},
	{random_struct{123, "nono", "hello"}, "d3:CDE5:hello3:abci123ee"},
	{map[string]string{"a": "b", "c": "d"}, "d1:a1:b1:c1:de"},
	{[]byte{1, 2, 3, 4}, "4:\x01\x02\x03\x04"},
	{&[4]byte{1, 2, 3, 4}, "4:\x01\x02\x03\x04"},
	{nil, ""},
	{[]byte{}, "0:"},
	{[]byte(nil), "0:"},
	{"", "0:"},
	{[]int{}, "le"},
	{map[string]int{}, "de"},
	{&dummy{1, 2, 3}, "i2ei3ei4e"},
	{struct {
		A *string
	}{nil}, "d1:A0:e"},
	{struct {
		A *string
	}{new(string)}, "d1:A0:e"},
	{struct {
		A *string `bencode:",omitempty"`
	}{nil}, "de"},
	{struct {
		A *string `bencode:",omitempty"`
	}{new(string)}, "d1:A0:e"},
	{bigIntFromString("62208002200000000000"), "i62208002200000000000e"},
	{*bigIntFromString("62208002200000000000"), "i62208002200000000000e"},
}

func bigIntFromString(s string) *big.Int {
	bi, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return bi
}

func TestRandomEncode(t *testing.T) {
	t.Parallel()

	for _, test := range random_encode_tests {
		data, err := Marshal(test.value)
		if err != nil {
			t.Errorf("Error encoding value: %v", err)
		}
		if string(data) != test.expected {
			t.Errorf("Unexpected encoded value. Got: %s, Expected: %s", string(data), test.expected)
		}
	}
}

func TestIsEmptyValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value    interface{}
		expected bool
	}{
		{nil, true},
		{0, true},
		{0.0, true},
		{false, true},
		{"", true},
		{[]int{}, false},
		{[2]int{0, 0}, true},
		{[2]int{0, 1}, false},
		{map[string]string{}, false},
		{struct{}{}, true},
		{random_struct{123, "nono", "hello"}, false},
		{&struct{}{}, false},
		{make(chan int), false},
		{func() {}, false},
	}

	for _, tc := range testCases {
		v := reflect.ValueOf(tc.value)
		isEmpty := isEmptyValue(v)
		if isEmpty != tc.expected {
			t.Errorf("Unexpected result for value %v. Got: %v, Expected: %v", tc.value, isEmpty, tc.expected)
		}
	}
}
