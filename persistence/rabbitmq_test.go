package persistence

import (
	"sync"
	"testing"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

func Test_rabbitmq_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	got, err := r.GetNumberOfTorrents()
	if err != nil {
		t.Errorf("rabbitmq.GetNumberOfTorrents() error = %v, want nil", err)
	}
	if got != 0 {
		t.Errorf("rabbitmq.GetNumberOfTorrents() = %v, want 0", got)
	}
}

func Test_rabbitmq_QueryTorrents(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}

	got, err := r.QueryTorrents(
		"example query",
		int64(1234567890),
		ByRelevance,
		true,
		uint64(10),
		nil,
		nil,
	)
	if err == nil {
		t.Error("rabbitmq.QueryTorrents() error = nil, want error")
	}
	if got != nil {
		t.Error("rabbitmq.QueryTorrents() != nil, want nil")
	}
}

func Test_rabbitmq_GetTorrent(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	got, err := r.GetTorrent([]byte("infoHash"))
	if err == nil {
		t.Error("rabbitmq.GetTorrent() error = nil, want error")
	}
	if got != nil {
		t.Error("rabbitmq.GetTorrent() != nil, want nil")
	}
}

func Test_rabbitmq_GetFiles(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	got, err := r.GetFiles([]byte("infoHash"))
	if err == nil {
		t.Error("rabbitmq.GetFiles() error = nil, , wanted error")
	}
	if got != nil {
		t.Errorf("rabbitmq.GetFiles() = %v, want nil", got)
	}
}

func Test_rabbitmq_GetStatistics(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	got, err := r.GetStatistics("", 0)
	if err == nil {
		t.Error("rabbitmq.GetStatistics() error = nil, wanted error")
	}
	if got != nil {
		t.Errorf("rabbitmq.GetStatistics() = %v, want nil", got)
	}
}

func Test_rabbitmq_Engine(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	if got := r.Engine(); got != RabbitMQ {
		t.Errorf("rabbitmq.Engine() = %v, want %v", got, RabbitMQ)
	}
}

func Test_rabbitmq_cleanup(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}

	// Add expired torrent to cache
	expiredInfoHash := "expiredInfoHash"
	r.cache[expiredInfoHash] = time.Now().Add(-1 * time.Minute)

	// Add valid torrent to cache
	validInfoHash := "validInfoHash"
	r.cache[validInfoHash] = time.Now().Add(10 * time.Minute)

	r.cleanup()

	if _, found := r.cache[expiredInfoHash]; found {
		t.Errorf("rabbitmq.cleanup() did not remove expired torrent")
	}

	if _, found := r.cache[validInfoHash]; !found {
		t.Errorf("rabbitmq.cleanup() removed valid torrent")
	}
}

func Test_rabbitmq_DoesTorrentExist(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "",
		conn:      nil,
		ch:        nil,
		dataQueue: nil,
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}

	infoHash := []byte("testhash")

	exists, err := r.DoesTorrentExist(infoHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected torrent to not exist")
	}

	r.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)

	exists, err = r.DoesTorrentExist(infoHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected torrent to exist")
	}
}

func Test_rabbitmq_connect(t *testing.T) {
	t.Parallel()

	r := &rabbitMQ{
		url:       "amqp://guest:guest@localhost:55672/",
		conn:      &amqp091.Connection{},
		ch:        &amqp091.Channel{},
		dataQueue: &amqp091.Queue{},
		cache:     map[string]time.Time{},
	}

	if err := r.connect(); err == nil {
		t.Error("rabbitmq.connect() error = nil, want error")
	}
}

func Test_rabbitmq_AddNewTorrent(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Test did not panick!")
		}
	}()

	r := &rabbitMQ{
		url:       "",
		conn:      &amqp091.Connection{},
		ch:        &amqp091.Channel{},
		dataQueue: &amqp091.Queue{},
		cache:     map[string]time.Time{},
		Mutex:     sync.Mutex{},
	}
	err := r.AddNewTorrent([]byte("exampleInfoHash"), "exampleName", []File{})
	if err == nil {
		t.Error("rabbitmq.AddNewTorrent() error = nil, want error")
	}
}
