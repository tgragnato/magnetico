//go:build cgo
// +build cgo

package persistence

import (
	"testing"
	"time"

	zmq "gopkg.in/zeromq/goczmq.v4"
)

func Test_zeromq_DoesTorrentExist(t *testing.T) {
	t.Parallel()

	instance := &zeromq{
		context: zmq.NewSock(zmq.Pub),
		cache:   map[string]time.Time{},
	}

	infoHash := []byte("exampleInfoHash")
	err := instance.AddNewTorrent(infoHash, "exampleName", []File{})
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
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

	instance := &zeromq{
		context: &zmq.Sock{},
		cache:   map[string]time.Time{},
	}
	got, err := instance.Export()
	if err == nil {
		t.Error("zeromq.Export() error = nil, want error")
	}
	if got != nil {
		t.Errorf("zeromq.Export() = %v, want nil", got)
	}
}
