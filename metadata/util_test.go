package metadata

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"math"
	mrand "math/rand/v2"
	"reflect"
	"testing"
	"time"

	"tgragnato.it/magnetico/bencode"
	"tgragnato.it/magnetico/metainfo"
	"tgragnato.it/magnetico/persistence"
)

func TestTotalSize(t *testing.T) {
	t.Parallel()
	positiveRand := mrand.Int64N(math.MaxInt64)

	tests := []struct {
		name    string
		files   []persistence.File
		want    uint64
		wantErr bool
	}{
		{
			name:    "No elements",
			files:   []persistence.File{},
			want:    0,
			wantErr: true,
		},
		{
			name: "Negative size",
			files: []persistence.File{
				{
					Size: -mrand.Int64N(math.MaxInt64),
					Path: "",
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "Zero size",
			files: []persistence.File{
				{
					Size: 0,
					Path: "",
				},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "Positive size",
			files: []persistence.File{
				{
					Size: positiveRand,
					Path: "",
				},
			},
			want:    uint64(positiveRand),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := totalSize(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("TotalSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TotalSize() = %v, want %v", got, tt.want)
			}
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
		{
			name:    "Empty test",
			info:    &metainfo.Info{},
			wantErr: true,
		},
		{
			name: "Pieces not %20",
			info: &metainfo.Info{
				PieceLength: 5,
				Pieces:      []byte{0, 0, 0, 0, 0},
				Name:        "",
				NameUtf8:    "",
				Length:      0,
				Private:     nil,
				Source:      "",
				Files:       []metainfo.FileInfo{},
			},
			wantErr: true,
		},
		{
			name: "Invalid file length",
			info: &metainfo.Info{
				PieceLength: 1,
				Pieces:      []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Name:        "",
				NameUtf8:    "",
				Length:      0,
				Private:     nil,
				Source:      "",
				Files:       []metainfo.FileInfo{},
			},
			wantErr: true,
		},
		{
			name: "valid info",
			info: &metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 1,
				Length:      20,
				Files:       []metainfo.FileInfo{{Length: 1, Path: []string{"file1"}}},
			},
			wantErr: false,
		},
		{
			name: "invalid pieces length",
			info: &metainfo.Info{
				Pieces:      make([]byte, 21),
				PieceLength: 1,
				Length:      20,
			},
			wantErr: true,
		},
		{
			name: "zero piece length with total length",
			info: &metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 0,
				Length:      20,
			},
			wantErr: true,
		},
		{
			name: "mismatch piece count and file lengths",
			info: &metainfo.Info{
				Pieces:      make([]byte, 20),
				PieceLength: 1,
				Length:      21,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateInfo(tt.info); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRandomDigit(t *testing.T) {
	t.Parallel()

	for i := 0; i < 100; i++ {
		digit := randomDigit()
		if digit < '0' || digit > '9' {
			t.Errorf("Expected a digit in range(0 - 9), got %c", digit)
		}
	}
}

func TestPeerId(t *testing.T) {
	t.Parallel()

	for i := 0; i < 100; i++ {
		peerID := randomID()
		lenPeerID := len(peerID)
		if lenPeerID > PeerIDLength {
			t.Errorf("peerId longer than 20 bytes: %s (%d)", peerID, lenPeerID)
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
		wantNil bool
	}{
		{"Test 1", 1, 1, false},
		{"Test 2", 256, 1, false},
		{"Test 3", 65536, 1, false},
		{"Test 4", math.MaxUint, 1, false},

		{"Test 5", 1, 2, false},
		{"Test 6", 256, 2, false},
		{"Test 7", 65536, 2, false},
		{"Test 8", math.MaxUint, 2, false},

		{"Test 9", 1, 4, false},
		{"Test 10", 256, 4, false},
		{"Test 11", 65536, 4, false},
		{"Test 12", math.MaxUint, 4, false},

		// Negative
		{"Test 13", math.MaxUint, 5, true},
		{"Test 14", math.MaxUint, -5, true},
		{"Test 15", 1, 5, true},
		{"Test 16", 2, 5, true},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if res := toBigEndian(test.i, test.n); test.wantNil && res != nil {
				t.Errorf("toBigEndian() = %v, want nil", res)
			}
		})
	}
}

func TestUnmarshalMetainfo(t *testing.T) {
	t.Parallel()

	metadataBytes := make([]byte, 100)
	if _, err := rand.Read(metadataBytes); err != nil {
		for i := 0; i < 100; i++ {
			metadataBytes[i] = byte(mrand.IntN(256))
		}
	}

	info, err := unmarshalMetainfo(metadataBytes)
	if err == nil {
		t.Error("invalid metadata but unmarshalMetainfo() error = nil")
		return
	}
	if info != nil {
		t.Error("invalid metadata but unmarshalMetainfo() Info != nil")
	}
}

func TestExtractFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		info     *metainfo.Info
		expected []persistence.File
	}{
		{
			name: "Single file",
			info: &metainfo.Info{
				Length: 100,
				Name:   "file.txt",
			},
			expected: []persistence.File{
				{
					Size: 100,
					Path: "file.txt",
				},
			},
		},
		{
			name: "Multiple files",
			info: &metainfo.Info{
				Files: []metainfo.FileInfo{
					{
						Length: 50,
						Path:   []string{"dir1", "file1.txt"},
					},
					{
						Length: 75,
						Path:   []string{"dir2", "file2.txt"},
					},
				},
			},
			expected: []persistence.File{
				{
					Size: 50,
					Path: "dir1/file1.txt",
				},
				{
					Size: 75,
					Path: "dir2/file2.txt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFiles(tt.info)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("extractFiles() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	t.Parallel()

	meta, err := bencode.Marshal(&metainfo.Info{
		PieceLength: 10,
		Pieces:      []byte{},
		Name:        "test",
		NameUtf8:    "test",
		Length:      10,
		Source:      "",
		Files: []metainfo.FileInfo{{
			Length:   0,
			Path:     []string{"test"},
			PathUtf8: []string{"test"},
		}},
	})
	if err != nil {
		t.Error(err)
	}

	infohash := sha1.Sum(meta)
	injectedTime := time.Now()

	actualMetadata, err := extractMetadata(meta, infohash, injectedTime)
	if err != nil {
		t.Errorf("extractMetadata() error = %v, want nil", err)
		return
	}

	expectedMetadata := &Metadata{
		InfoHash:     infohash[:],
		Name:         "test",
		TotalSize:    0,
		DiscoveredOn: injectedTime.Unix(),
		Files: []persistence.File{{
			Size: 0,
			Path: "test",
		}},
	}
	if !reflect.DeepEqual(actualMetadata, expectedMetadata) {
		t.Errorf("extractMetadata() = %v, want %v", actualMetadata, expectedMetadata)
	}
}
