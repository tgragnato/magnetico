package dht

import (
	"math/rand/v2"
	"net"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"tgragnato.it/magnetico/v2/dht/mainline"
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
	manager.mu.Lock()
	manager.output = outputChan
	manager.mu.Unlock()
	manager.output <- result

	receivedResult := <-outputChan
	if !reflect.DeepEqual(receivedResult, result) {
		t.Errorf("\nReceived result %v, \nExpected result %v", receivedResult, result)
	}

	manager.Terminate()
}

func TestOnIndexingResultDrainsFullChannelWithoutBlockingOrDropping(t *testing.T) {
	t.Parallel()

	oldOutput := make(chan Result, ChanSize)
	manager := &Manager{output: oldOutput}
	expected := make([]Result, 0, ChanSize+1)

	for range ChanSize {
		result := &TestResult{infoHash: [20]byte{byte(len(expected) + 1)}}
		expected = append(expected, result)
		oldOutput <- result
	}
	newResult := mainline.IndexingResult{}
	expected = append(expected, newResult)

	done := make(chan struct{})
	go func() {
		manager.onIndexingResult(newResult)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(DefaultTimeOut):
		t.Fatal("draining full output channel blocked")
	}

	if _, ok := <-oldOutput; ok {
		t.Fatal("old output channel was not closed after draining")
	}

	output := manager.Output()
	for i, want := range expected {
		select {
		case got := <-output:
			if !reflect.DeepEqual(got, want) {
				t.Errorf("result %d = %v, want %v", i, got, want)
			}
		case <-time.After(DefaultTimeOut):
			t.Fatalf("result %d was lost during channel swap", i)
		}
	}
}

func TestOnIndexingResultConcurrentProducersDoNotDropResults(t *testing.T) {
	t.Parallel()

	const resultCount = 64
	manager := &Manager{output: make(chan Result, 1)}
	var producers sync.WaitGroup
	producers.Add(resultCount)
	for range resultCount {
		go func() {
			defer producers.Done()
			manager.onIndexingResult(mainline.IndexingResult{})
		}()
	}
	producers.Wait()

	output := manager.Output()
	for range resultCount {
		select {
		case result := <-output:
			if result == nil {
				t.Fatal("received nil result")
			}
		case <-time.After(DefaultTimeOut):
			t.Fatal("a concurrent result was lost")
		}
	}
}
