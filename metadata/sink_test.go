package metadata

import (
	"net"
	"reflect"
	"testing"
	"time"
)

func TestSink_NewSink(t *testing.T) {
	t.Parallel()

	sink := NewSink(time.Second, 10, []net.IPNet{})
	if sink == nil ||
		len(sink.PeerID) != 20 ||
		sink.deadline != time.Second ||
		sink.drain == nil ||
		sink.incomingInfoHashes == nil ||
		sink.termination == nil {
		t.Error("One or more fields of Sink were not initialized correctly")
	}
}

type TestResult struct {
	infoHash  [20]byte
	peerAddrs []net.TCPAddr
}

func (tr *TestResult) InfoHash() [20]byte {
	return tr.infoHash
}

func (tr *TestResult) PeerAddrs() []net.TCPAddr {
	return tr.peerAddrs
}

func TestSink_Sink(t *testing.T) {
	t.Parallel()

	sink := NewSink(time.Minute, 2, []net.IPNet{})
	testResult := &TestResult{
		infoHash:  [20]byte{255},
		peerAddrs: []net.TCPAddr{{IP: net.ParseIP("1.0.0.1"), Port: 443}},
	}
	sink.Sink(testResult)
	if len(sink.incomingInfoHashes.infoHashes) != 0 {
		t.Error("infoHashes should be empty after sinking a result with a single peer")
	}
}

func TestSink_leechRejectsDisallowedPeers(t *testing.T) {
	t.Parallel()

	sink := NewSink(time.Minute, 2, []net.IPNet{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		sink.leech([20]byte{1}, []net.TCPAddr{
			{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			{IP: net.ParseIP("10.0.0.1"), Port: 6881},
			{IP: net.ParseIP("1.0.0.1"), Port: 80},
		})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("leech must not dial peers rejected by the address policy")
	}
	if len(sink.incomingInfoHashes.infoHashes) != 0 {
		t.Error("no peer should be queued when all are disallowed")
	}
}

func TestSink_Terminate(t *testing.T) {
	t.Parallel()

	sink := NewSink(time.Minute, 1, []net.IPNet{})
	sink.Terminate()

	if !sink.terminated {
		t.Error("terminated field of Sink has not been set to true")
	}
}

func TestSink_Drain(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("No panic while Draining an already closed Sink!")
		}
	}()

	sink := NewSink(time.Minute, 1, []net.IPNet{})
	sink.Terminate()
	sink.Drain()
}

func TestFlush(t *testing.T) {
	t.Parallel()

	sink := NewSink(time.Minute, 1, []net.IPNet{})
	testMetadata := Metadata{
		InfoHash: []byte{1, 2, 3, 4, 5, 6},
	}

	go func() {
		select {
		case result := <-sink.drain:
			if !reflect.DeepEqual(result.InfoHash, testMetadata.InfoHash) {
				t.Errorf("Expected flushed InfoHash to be %v, but got %v", testMetadata.InfoHash, result.InfoHash)
			}

		case <-time.After(1 * time.Second):
			t.Error("Timeout waiting for flush result")
		}
	}()

	sink.flush(testMetadata)

	time.Sleep(500 * time.Millisecond)

	var infoHash [20]byte
	copy(infoHash[:], testMetadata.InfoHash)
	sink.incomingInfoHashes.Lock()
	_, exists := sink.incomingInfoHashes.infoHashes[infoHash]
	sink.incomingInfoHashes.Unlock()
	if exists {
		t.Error("InfoHash was not deleted after flush")
	}
}
