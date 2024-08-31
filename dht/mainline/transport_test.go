package mainline

import (
	"math/rand"
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

func TestWriteMessages(t *testing.T) {
	t.Parallel()

	transport := NewTransport(
		net.JoinHostPort("::1", strconv.Itoa(rand.Intn(64511)+1024)),
		func(m *Message, u *net.UDPAddr) {},
	)
	transport.throttlingRate = 10
	transport.Start()

	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
	}{
		{
			name:    "Nil message",
			msg:     nil,
			wantErr: false,
		},
		{
			name:    "Empty message",
			msg:     nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := transport.WriteMessages(tt.msg, transport.laddr); (err != nil) != tt.wantErr {
				t.Errorf("Transport.WriteMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	transport.Terminate()
}
