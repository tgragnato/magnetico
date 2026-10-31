package dht

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"tgragnato.it/magnetico/v2/dht/mainline"
	"tgragnato.it/magnetico/v2/persistence"
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

type outboundDatagram struct {
	payload []byte
	address net.UDPAddr
}

type outboundTestService struct {
	datagrams chan outboundDatagram
}

func (s *outboundTestService) Start() {}

func (s *outboundTestService) Terminate() {}

func (s *outboundTestService) SendDatagram(payload []byte, address *net.UDPAddr) error {
	s.datagrams <- outboundDatagram{payload: append([]byte(nil), payload...), address: *address}
	return nil
}

func TestOutboundDispatcherUsesGlobalQueueAndAnyTransport(t *testing.T) {
	t.Parallel()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	database, err := persistence.MakeDatabase(fmt.Sprintf("sqlite3:///%s?cache=shared&mode=memory", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	first := &outboundTestService{datagrams: make(chan outboundDatagram, 1)}
	second := &outboundTestService{datagrams: make(chan outboundDatagram, 1)}
	manager := &Manager{indexingServices: []Service{first, second}, outboundDone: make(chan struct{})}
	queue := mainline.NewOutboundQueueState(10)
	ctx, cancel := context.WithCancel(context.Background())
	go manager.dispatchOutbound(ctx, database, mainline.NewOutboundRateLimiter(0), queue)
	defer func() {
		cancel()
		<-manager.outboundDone
	}()

	address := net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881}
	for _, payload := range [][]byte{[]byte("first"), []byte("second")} {
		if err := database.EnqueueDHTMessage(ctx, persistence.DHTOutboundMessage{Destination: address, Payload: payload}); err != nil {
			t.Fatal(err)
		}
	}

	for index, service := range []*outboundTestService{first, second} {
		select {
		case datagram := <-service.datagrams:
			wantPayload := []byte("first")
			if index == 1 {
				wantPayload = []byte("second")
			}
			if string(datagram.payload) != string(wantPayload) || datagram.address.Port != address.Port {
				t.Fatalf("transport %d received (%q, %v), want (%q, %v)", index, datagram.payload, datagram.address, wantPayload, address)
			}
		case <-time.After(time.Second):
			t.Fatalf("transport %d did not receive an outbound datagram", index)
		}
	}

	deadline := time.Now().Add(time.Second)
	for {
		if _, found, err := database.NextDHTMessage(ctx); err != nil {
			t.Fatal(err)
		} else if !found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("outbound queue still contained acknowledged datagrams")
		}
		time.Sleep(time.Millisecond)
	}
}

func (tr *TestResult) InfoHash() [20]byte {
	return tr.infoHash
}

func (tr *TestResult) PeerAddrs() []net.TCPAddr {
	return tr.peerAddrs
}

func TestChannelOutput(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())

	address := ManagerAddress + ":" + strconv.Itoa(rand.IntN(64511)+1024)
	database, err := persistence.MakeDatabase(fmt.Sprintf("sqlite3:///%s?cache=shared&mode=memory", strings.ReplaceAll(t.Name(), "/", "_")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})
	manager, err := NewManager(ctx, []string{address}, MaxNeighbours, []string{"dht.tgragnato.it"}, []net.IPNet{}, database, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		manager.Terminate()
	})
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
