package dht

import (
	"net"
	"reflect"
	"testing"
	"time"
)

const (
	ChanSize       = 20
	MaxNeighbours  = 10
	ManagerAddress = "0.0.0.0:2024"
	PeerIP         = "192.168.1.1"
	PeerPort       = 6931
	DefaultTimeOut = 1 * time.Second
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
	manager := NewManager([]string{ManagerAddress}, DefaultTimeOut, MaxNeighbours)

	result := &TestResult{
		infoHash:  [20]byte{255},
		peerAddrs: []net.TCPAddr{{IP: net.ParseIP(PeerIP), Port: PeerPort, Zone: ""}},
	}
	outputChan := make(chan Result, ChanSize)
	manager.output = outputChan
	manager.output <- result

	receivedResult := <-outputChan
	if !reflect.DeepEqual(receivedResult, result) {
		t.Errorf("\nReceived result %v, \nExpected result %v", receivedResult, result)
	}
}
