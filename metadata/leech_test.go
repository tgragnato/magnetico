package metadata

import (
	"bytes"
	"math"
	"testing"

	"github.com/anacrolix/torrent/bencode"
)

var operationsTest_instances = []struct {
	dump    []byte
	surplus []byte
}{
	// No Surplus
	{
		dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528ee"),
		surplus: []byte(""),
	},
	// Surplus is an ASCII string
	{
		dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528eeDENEME"),
		surplus: []byte("DENEME"),
	},
	// Surplus is a bencoded dictionary
	{
		dump:    []byte("d1:md11:ut_metadatai1ee13:metadata_sizei22528eed3:inti1337ee"),
		surplus: []byte("d3:inti1337ee"),
	},
}

func TestDecoder(t *testing.T) {
	t.Parallel()
	for i, instance := range operationsTest_instances {
		buf := bytes.NewBuffer(instance.dump)
		err := bencode.NewDecoder(buf).Decode(&struct{}{})
		if err != nil {
			t.Errorf("Couldn't decode the dump #%d! %s", i+1, err.Error())
		}

		bufSurplus := buf.Bytes()
		if !bytes.Equal(bufSurplus, instance.surplus) {
			t.Errorf("Surplus #%d is not equal to what we expected! `%s`", i+1, bufSurplus)
		}
	}
}

func TestToBigEndian(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		i    uint
		n    int
		want []byte
	}{
		{"Test OneByte1", 1, 1, []byte{1}},
		{"Test OneByte2", 255, 1, []byte{255}},
		{"Test OneByte3", 65535, 1, []byte{255}},
		{"Test OneByte4", math.MaxUint64, 1, []byte{255}},

		{"Test TwoBytes 1", 1, 2, []byte{0, 1}},
		{"Test TwoBytes 2", 255, 2, []byte{0, 255}},
		{"Test TwoBytes 3", 65535, 2, []byte{255, 255}},
		{"Test TwoBytes 4", math.MaxUint64, 2, []byte{255, 255}},

		{"Test FourBytes1", 1, 4, []byte{0, 0, 0, 1}},
		{"Test FourBytes2", 255, 4, []byte{0, 0, 0, 255}},
		{"Test FourBytes3", 65535, 4, []byte{0, 0, 255, 255}},
		{"Test FourBytes4", math.MaxUint64, 4, []byte{255, 255, 255, 255}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := toBigEndian(tc.i, tc.n)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("toBigEndian(%d, %d) = %v; want %v", tc.i, tc.n, got, tc.want)
			}
		})
	}
}

func TestToBigEndianNegative(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		i       uint
		n       int
		wantErr bool
	}{
		{"Test 1", 1, 1, false},
		{"Test 2", 256, 1, false},
		{"Test 3", 65536, 1, false},
		{"Test 4", math.MaxUint64, 1, false},

		{"Test 5", 1, 2, false},
		{"Test 6", 256, 2, false},
		{"Test 7", 65536, 2, false},
		{"Test 8", math.MaxUint64, 2, false},

		{"Test 9", 1, 4, false},
		{"Test 10", 256, 4, false},
		{"Test 11", 65536, 4, false},
		{"Test 12", math.MaxUint64, 4, false},

		// Negative
		{"Test 13", math.MaxUint64, 5, true},
		{"Test 14", math.MaxUint64, -5, true},
		{"Test 15", 1, 5, true},
		{"Test 16", 2, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.wantErr {
					t.Errorf("toBigEndian() = %v, wantErr %v", r, tt.wantErr)
				}
			}()
			toBigEndian(tt.i, tt.n)
		})
	}
}
