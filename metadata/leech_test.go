package metadata

import (
	"bytes"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"math"
	"testing"
)

func TestDecoder(t *testing.T) {
	t.Parallel()

	var operationInstances = []struct {
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

	for i, instance := range operationInstances {
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
	tests := []struct {
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
	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := toBigEndian(testCase.i, testCase.n)
			if !bytes.Equal(got, testCase.want) {
				t.Errorf("toBigEndian(%d, %d) = %v; want %v", testCase.i, testCase.n, got, testCase.want)
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
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); (r != nil) != test.wantErr {
					t.Errorf("toBigEndian() = %v, wantErr %v", r, test.wantErr)
				}
			}()
			toBigEndian(test.i, test.n)
		})
	}
}

func TestValidateInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		info    *metainfo.Info
		wantErr bool
	}{
		{"valid info",
			&metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 1,
				Length:      20,
				Files:       []metainfo.FileInfo{{Length: 1, Path: []string{"file1"}}},
			},
			false,
		}, {"invalid pieces length",
			&metainfo.Info{
				Pieces:      make([]byte, 21),
				PieceLength: 1,
				Length:      20,
			},
			true,
		}, {"zero piece length with total length",
			&metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 0,
				Length:      20,
			},
			true,
		}, {"mismatch piece count and file lengths",
			&metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 1,
				Length:      21,
			},
			true,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := validateInfo(test.info); (err != nil) != test.wantErr {
				t.Errorf("validateInfo() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
