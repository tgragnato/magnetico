package metadata

import (
	"testing"
	"time"
)

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

func TestNewSink(t *testing.T) {
	sink := NewSink(time.Second, 10)

	if sink == nil ||
		len(sink.PeerID) != 20 ||
		sink.deadline != time.Second ||
		sink.maxNLeeches != 10 ||
		sink.drain == nil ||
		sink.incomingInfoHashes == nil ||
		sink.termination == nil {
		t.Errorf("One or more fields of Sink were not initialized correctly.")
	}

}
