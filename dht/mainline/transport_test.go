package mainline

import (
	"context"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	DEFAULT_IP            = "0.0.0.0:0"
	MSG_SKIP_ERR          = "Skipping due to an error during initialization!"
	MSG_UNEXPECTED_SUFFIX = "Unexpected suffix in the error message!"
	MSG_CLOSED_CONNECTION = "use of closed network connection"
)

func TestReadFromOnClosedConn(t *testing.T) {
	t.Parallel()
	// Initialization
	laddr, err := net.ResolveUDPAddr("udp", DEFAULT_IP)
	if err != nil {
		t.Skip(MSG_SKIP_ERR)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Skip(MSG_SKIP_ERR)
	}

	buffer := make([]byte, 65536)

	// Setting Up
	conn.Close()

	// Testing
	_, _, err = conn.ReadFrom(buffer)
	if err == nil || !strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION) {
		t.Fatal(MSG_UNEXPECTED_SUFFIX)
	}
}

func TestWriteToOnClosedConn(t *testing.T) {
	t.Parallel()
	// Initialization
	laddr, err := net.ResolveUDPAddr("udp", DEFAULT_IP)
	if err != nil {
		t.Skip(MSG_SKIP_ERR)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Skip(MSG_SKIP_ERR)
	}

	// Setting Up
	conn.Close()

	// Testing
	_, err = conn.WriteTo([]byte("estarabim"), laddr)
	if err == nil || !strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION) {
		t.Fatal(MSG_UNEXPECTED_SUFFIX)
	}
}

func TestTransport_WriteMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     *Message
		addr    *net.UDPAddr
		wantErr bool
	}{
		{
			name:    "Nil message",
			msg:     nil,
			addr:    &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantErr: false,
		},
		{
			name:    "Nil address",
			msg:     &Message{Q: "ping"},
			addr:    nil,
			wantErr: false,
		},
		{
			name:    "Valid message and address",
			msg:     &Message{Q: "ping"},
			addr:    &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewTransport(
				net.JoinHostPort("::1", strconv.Itoa(rand.IntN(64511)+1024)),
				func(m *Message, u *net.UDPAddr) {},
				1000,
			)
			transport.Start(context.Background())
			defer transport.Terminate()

			if err := transport.WriteMessages(tt.msg, tt.addr); (err != nil) != tt.wantErr {
				t.Errorf("Transport.WriteMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOutboundRateLimiterSpacesMessages(t *testing.T) {
	t.Parallel()
	limiter := NewOutboundRateLimiter(100)
	started := time.Now()
	for range 3 {
		if err := limiter.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("three sends took %v, want at least 15ms at 100 RPS", elapsed)
	}
}

func TestOutboundRateLimiterSharesBudgetAcrossWorkers(t *testing.T) {
	t.Parallel()
	limiter := NewOutboundRateLimiter(50)
	started := time.Now()
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := limiter.Wait(context.Background()); err != nil {
				t.Errorf("rate limiter wait: %v", err)
			}
		}()
	}
	workers.Wait()
	if elapsed := time.Since(started); elapsed < 45*time.Millisecond {
		t.Fatalf("four concurrent sends took %v, want at least 45ms at shared 50 RPS", elapsed)
	}
}

func TestTransportEnqueuesDatagramGlobally(t *testing.T) {
	t.Parallel()
	database := newRoutingDatabase(t, 10, nil)
	queue := NewOutboundQueueState(10)
	transport := NewTransport("127.0.0.1:0", func(*Message, *net.UDPAddr) {}, 10)
	transport.ConfigureOutboundQueue(database, queue)
	ctx, cancel := context.WithCancel(context.Background())
	transport.Start(ctx)
	t.Cleanup(func() {
		cancel()
		transport.Terminate()
	})

	message := &Message{
		Y: "q",
		T: []byte("aa"),
		Q: "ping",
		A: QueryArguments{ID: make([]byte, 20)},
	}
	destination := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6881}
	if err := transport.WriteMessages(message, destination); err != nil {
		t.Fatal(err)
	}
	queued, found, err := database.NextDHTMessage(ctx)
	if err != nil || !found || string(queued.Payload) == "" {
		t.Fatalf("NextDHTMessage() = (%+v, %v, %v), want a globally queued payload", queued, found, err)
	}
	if queued.Destination.IP.String() != destination.IP.String() || queued.Destination.Port != destination.Port {
		t.Fatalf("queued destination = %v, want %v", queued.Destination, destination)
	}
	if err := database.AckDHTMessage(ctx, queued.ID); err != nil {
		t.Fatal(err)
	}
	queue.Acknowledge()
}

func TestTransportFull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		queuedCommunications uint64
		maxNeighbors         uint
		want                 bool
	}{
		{
			name:                 "Transport not full",
			queuedCommunications: 5,
			maxNeighbors:         10,
			want:                 false,
		},
		{
			name:                 "Transport full",
			queuedCommunications: 10,
			maxNeighbors:         10,
			want:                 true,
		},
		{
			name:                 "Transport over capacity",
			queuedCommunications: 15,
			maxNeighbors:         10,
			want:                 true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &Transport{
				queuedCommunications: tt.queuedCommunications,
				maxNeighbors:         tt.maxNeighbors,
			}
			if got := transport.Full(); got != tt.want {
				t.Errorf("Transport.Full() = %v, want %v", got, tt.want)
			}
		})
	}
}
