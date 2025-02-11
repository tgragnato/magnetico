package mainline

import (
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"testing"
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
	if !(err != nil && strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION)) {
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
	if !(err != nil && strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION)) {
		t.Fatal(MSG_UNEXPECTED_SUFFIX)
	}
}

func TestTransport_WriteMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		throttlingRate int
		msg            *Message
		addr           *net.UDPAddr
		wantErr        bool
	}{
		{
			name:           "Nil message",
			throttlingRate: 10,
			msg:            nil,
			addr:           &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantErr:        false,
		},
		{
			name:           "Nil address",
			throttlingRate: 10,
			msg:            &Message{Q: "ping"},
			addr:           nil,
			wantErr:        false,
		},
		{
			name:           "Valid message and address",
			throttlingRate: 10,
			msg:            &Message{Q: "ping"},
			addr:           &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantErr:        false,
		},
		{
			name:           "Throttle limit reached",
			throttlingRate: 0,
			msg:            &Message{Q: "ping"},
			addr:           &net.UDPAddr{IP: net.ParseIP("::1"), Port: 8080},
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewTransport(
				net.JoinHostPort("::1", strconv.Itoa(rand.IntN(64511)+1024)),
				func(m *Message, u *net.UDPAddr) {},
				1000,
			)
			transport.throttlingRate = tt.throttlingRate
			transport.Start()
			defer transport.Terminate()

			if err := transport.WriteMessages(tt.msg, tt.addr); (err != nil) != tt.wantErr {
				t.Errorf("Transport.WriteMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
