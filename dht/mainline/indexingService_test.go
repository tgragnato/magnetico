package mainline

import (
	"testing"
)

func TestUint16BE(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    uint16
		want [2]byte
	}{
		{"zero", 0, [2]byte{0, 0}},
		{"one", 1, [2]byte{0, 1}},
		{"two", 2, [2]byte{0, 2}},
		{"max", 0xFFFF, [2]byte{0xFF, 0xFF}},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := toBigEndianBytes(test.v)
			if got != test.want {
				t.Errorf("toBigEndianBytes(%v) = %v, want %v", test.v, got, test.want)
			}
		})
	}
}
