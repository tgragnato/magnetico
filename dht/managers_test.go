package dht

import (
	"math/rand/v2"
	"net"
	"reflect"
	"strconv"
	"testing"
	"time"

	"tgragnato.it/magnetico/dht/mainline"
)

const (
	ChanSize       = 20
	MaxNeighbours  = 10
	ManagerAddress = "127.0.0.1"
	PeerIP         = "192.168.1.1"
	DefaultTimeOut = time.Second
)

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

func TestChannelOutput(t *testing.T) {
	t.Parallel()

	address := ManagerAddress + ":" + strconv.Itoa(rand.IntN(64511)+1024)
	manager := NewManager([]string{address}, MaxNeighbours, []string{"dht.tgragnato.it"}, []net.IPNet{})
	peerPort := rand.IntN(64511) + 1024

	result := &TestResult{
		infoHash: [20]byte{255},
		peerAddrs: []net.TCPAddr{{
			IP:   net.ParseIP(PeerIP),
			Port: peerPort,
		}},
	}
	outputChan := make(chan Result, ChanSize)
	manager.output = outputChan
	manager.output <- result

	receivedResult := <-outputChan
	if !reflect.DeepEqual(receivedResult, result) {
		t.Errorf("\nReceived result %v, \nExpected result %v", receivedResult, result)
	}

	manager.Terminate()
}

func TestOnIndexingResult(t *testing.T) {
	t.Parallel()

	address := ManagerAddress + ":" + strconv.Itoa(rand.IntN(64511)+1024)
	manager := NewManager([]string{address}, MaxNeighbours, []string{"dht.tgragnato.it"}, []net.IPNet{})

	result := mainline.IndexingResult{}
	outputChan := make(chan Result, ChanSize)
	manager.output = outputChan

	for i := 0; i < ChanSize; i++ {
		manager.onIndexingResult(result)
	}

	// Verify that the result is sent to the output channel
	select {
	case receivedResult := <-outputChan:
		if !reflect.DeepEqual(receivedResult, result) {
			t.Errorf("\nReceived result %v, \nExpected result %v", receivedResult, result)
		}
	default:
		t.Error("Expected result not received")
	}

	manager.Terminate()
}
