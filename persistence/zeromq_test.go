//go:build cgo

package persistence

import (
	"context"
	"net"
	"testing"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func Test_zeromq_DHTRouting(t *testing.T) {
	t.Parallel()
	instance := &zeromq{}
	if err := instance.startDHTRouter(); err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	_, filter, err := net.ParseCIDR("127.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.ConfigureDHTRoutingTable(2, []net.IPNet{*filter}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first := net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881}
	second := net.UDPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 6882}
	if err := instance.AddDHTNodes(ctx, []net.UDPAddr{first, first, second, {IP: net.IPv4(10, 0, 0, 1), Port: 6883}}); err != nil {
		t.Fatal(err)
	}
	if empty, err := instance.IsDHTRoutingEmpty(ctx); err != nil || empty {
		t.Fatalf("IsDHTRoutingEmpty() = (%v, %v), want (false, nil)", empty, err)
	}
	if nodes, err := instance.DumpDHTNodes(ctx, true); err != nil || len(nodes) != 2 {
		t.Fatalf("DumpDHTNodes() returned %d nodes and error %v, want 2 nodes and nil", len(nodes), err)
	}
	if nodes, err := instance.GetDHTNodes(ctx); err != nil || len(nodes) != 2 {
		t.Fatalf("GetDHTNodes() returned %d nodes and error %v, want 2 nodes and nil", len(nodes), err)
	}
	if nodes, err := instance.GetDHTNodes(ctx); err != nil || len(nodes) != 1 || nodes[0].Port != first.Port {
		t.Fatalf("second GetDHTNodes() = (%v, %v), want only %v", nodes, err, first)
	}
	if err := instance.AddDHTHashes(ctx, [][20]byte{{1}, {2}}); err != nil {
		t.Fatal(err)
	}
	hashes, err := instance.GetDHTHashes(ctx)
	if err != nil || hashes[0] != [20]byte{1} || hashes[1] != [20]byte{2} {
		t.Fatalf("GetDHTHashes() = %v, %v, want stored hashes", hashes[:2], err)
	}
}

func Test_zeromq_DHTOutbox(t *testing.T) {
	t.Parallel()
	instance := &zeromq{}
	if err := instance.startDHTRouter(); err != nil {
		t.Fatal(err)
	}
	defer instance.Close()
	if err := instance.ConfigureDHTRoutingTable(2, nil); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	message := DHTOutboundMessage{
		Destination: net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881},
		Payload:     []byte("payload"),
		QueuedAt:    1,
	}
	if err := instance.EnqueueDHTMessage(ctx, message); err != nil {
		t.Fatal(err)
	}
	if err := instance.EnqueueDHTMessage(ctx, DHTOutboundMessage{
		Destination: message.Destination,
		Payload:     []byte("next"),
		QueuedAt:    2,
	}); err != nil {
		t.Fatal(err)
	}
	queued, found, err := instance.NextDHTMessage(ctx)
	if err != nil || !found || queued.ID == "" || string(queued.Payload) != "payload" {
		t.Fatalf("NextDHTMessage() = (%+v, %v, %v), want queued payload", queued, found, err)
	}
	if err := instance.ReleaseDHTMessage(ctx, queued.ID); err != nil {
		t.Fatal(err)
	}
	next, found, err := instance.NextDHTMessage(ctx)
	if err != nil || !found || string(next.Payload) != "next" {
		t.Fatalf("NextDHTMessage() after release = (%+v, %v, %v), want next message", next, found, err)
	}
	if err := instance.AckDHTMessage(ctx, next.ID); err != nil {
		t.Fatal(err)
	}
	retried, found, err := instance.NextDHTMessage(ctx)
	if err != nil || !found || retried.ID != queued.ID {
		t.Fatalf("NextDHTMessage() after next ack = (%+v, %v, %v), want released message", retried, found, err)
	}
	if err := instance.AckDHTMessage(ctx, retried.ID); err != nil {
		t.Fatal(err)
	}
	if _, found, err := instance.NextDHTMessage(ctx); err != nil || found {
		t.Fatalf("NextDHTMessage() after ack found=%v err=%v, want no message", found, err)
	}
}

func Test_zeromq_DoesTorrentExist(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}

	infoHash := []byte("exampleInfoHash")
	err = instance.AddNewTorrent(infoHash, "exampleName", []File{})
	if err != nil {
		t.Errorf("zeromq.AddNewTorrent() error = %v, want nil", err)
	}

	got, err := instance.DoesTorrentExist(infoHash)
	if err != nil {
		t.Errorf("zeromq.DoesTorrentExist() error = %v, want nil", err)
	}
	if !got {
		t.Errorf("zeromq.DoesTorrentExist() = %v, want true", got)
	}

	if err = instance.Close(); err != nil {
		t.Errorf("zeromq.Close() error = %v, want nil", err)
	}
}

func Test_zeromq_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	got, err := instance.GetNumberOfTorrents()
	if err != nil {
		t.Errorf("zeromq.GetNumberOfTorrents() error = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("zeromq.GetNumberOfTorrents() = %v, want 0", got)
	}
}

func Test_zeromq_QueryTorrents(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}

	got, err := instance.QueryTorrents(
		"example query",
		int64(1234567890),
		ByRelevance,
		true,
		uint64(10),
		nil,
		nil,
	)
	if err == nil {
		t.Error("zeromq.QueryTorrents() error = nil, want error")
	}
	if got != nil {
		t.Error("zeromq.QueryTorrents() != nil, want nil")
	}
}

func Test_zeromq_GetTorrent(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	got, err := instance.GetTorrent([]byte("infoHash"))
	if err == nil {
		t.Error("zeromq.GetTorrent() error = nil, want error")
	}
	if got != nil {
		t.Error("zeromq.GetTorrent() != nil, want nil")
	}
}

func Test_zeromq_GetFiles(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	got, err := instance.GetFiles([]byte("infoHash"))
	if err == nil {
		t.Error("zeromq.GetFiles() error = nil, , wanted error")
	}
	if got != nil {
		t.Errorf("zeromq.GetFiles() = %v, want nil", got)
	}
}

func Test_zeromq_GetStatistics(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	got, err := instance.GetStatistics("", 0)
	if err == nil {
		t.Error("zeromq.GetStatistics() error = nil, wanted error")
	}
	if got != nil {
		t.Errorf("zeromq.GetStatistics() = %v, want nil", got)
	}
}

func Test_zeromq_Engine(t *testing.T) {
	t.Parallel()

	instance := &zeromq{}
	if got := instance.Engine(); got != ZeroMQ {
		t.Errorf("zeromq.Engine() = %v, want %v", got, ZeroMQ)
	}
}

func Test_zeromq_cleanup(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}

	// Add expired torrent to cache
	expiredInfoHash := "expiredInfoHash"
	instance.cache[expiredInfoHash] = time.Now().Add(-1 * time.Minute)

	// Add valid torrent to cache
	validInfoHash := "validInfoHash"
	instance.cache[validInfoHash] = time.Now().Add(10 * time.Minute)

	instance.cleanup()

	if _, found := instance.cache[expiredInfoHash]; found {
		t.Errorf("zeromq.cleanup() did not remove expired torrent")
	}

	if _, found := instance.cache[validInfoHash]; !found {
		t.Errorf("zeromq.cleanup() removed valid torrent")
	}
}

func Test_zeromq_Export(t *testing.T) {
	t.Parallel()

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		t.Fatalf("failed to create zmq socket: %v", err)
	}
	defer socket.Close()

	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	got, err := instance.Export()
	if err == nil {
		t.Error("zeromq.Export() error = nil, want error")
	}
	if got != nil {
		t.Errorf("zeromq.Export() = %v, want nil", got)
	}
}
