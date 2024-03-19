package mainline

import (
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestUint16BE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    uint16
		want [2]byte
	}{
		{"zero", 0, [2]byte{0, 0}},
		{"one", 1, [2]byte{0, 1}},
		{"two", 2, [2]byte{0, 2}},
		{"max", 0xFFFF, [2]byte{0xFF, 0xFF}},
	}

	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := toBigEndianBytes(test.v)
			if got != test.want {
				t.Errorf("toBigEndianBytes(%v) = %v, want %v", test.v, got, test.want)
			}
		})
	}
}

func TestBasicIndexingService(t *testing.T) {
	t.Parallel()

	randomPort := rand.Intn(64511) + 1024
	tests := []struct {
		name          string
		laddr         string
		interval      time.Duration
		maxNeighbors  uint
		eventHandlers IndexingServiceEventHandlers
	}{
		{
			name:          "Loopback Random IPv4",
			laddr:         net.JoinHostPort("127.0.0.1", strconv.Itoa(randomPort)),
			interval:      500 * time.Millisecond,
			maxNeighbors:  0,
			eventHandlers: IndexingServiceEventHandlers{},
		},
		{
			name:          "Loopback Random IPv6",
			laddr:         net.JoinHostPort("::1", strconv.Itoa(randomPort)),
			interval:      500 * time.Millisecond,
			maxNeighbors:  0,
			eventHandlers: IndexingServiceEventHandlers{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := NewIndexingService(tt.laddr, tt.interval, tt.maxNeighbors, tt.eventHandlers, []string{"dht.tgragnato.it"})
			if is == nil {
				t.Error("NewIndexingService() = nil, wanted != nil")
			}
			is.Start()
			time.Sleep(time.Second)
			is.findNeighbors()
			time.Sleep(time.Second)
			is.Terminate()
		})
	}
}
