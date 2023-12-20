package metadata

import (
	"testing"
)

func TestRandomDigit(t *testing.T) {
	for i := 0; i < 100; i++ {
		digit := randomDigit()
		if digit < '0' || digit > '9' {
			t.Errorf("Expected a digit in range(0 - 9), got %c", digit)
		}
	}
}

func TestPeerId(t *testing.T) {
	for i := 0; i < 100; i++ {
		peerId := randomID()
		lenPeerId := len(peerId)
		if lenPeerId > 20 {
			t.Errorf("peerId longer than 20 bytes: %s (%d)", peerId, lenPeerId)
		}
	}
}
