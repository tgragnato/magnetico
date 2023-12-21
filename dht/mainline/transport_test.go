package mainline

import (
	"net"
	"strings"
	"testing"
)

const DEFAULT_IP = "0.0.0.0:0"
const MSG_SKIP_ERR = "Skipping due to an error during initialization!"
const MSG_UNEXPECTED_SUFFIX = "Unexpected suffix in the error message!"
const MSG_CLOSED_CONNECTION = "use of closed network connection"

func TestReadFromOnClosedConn(t *testing.T) {
	t.Parallel()
	// Initialization
	laddr, err := net.ResolveUDPAddr("udp", DEFAULT_IP)
	if err != nil {
		t.Skipf(MSG_SKIP_ERR)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Skipf(MSG_SKIP_ERR)
	}

	buffer := make([]byte, 65536)

	// Setting Up
	conn.Close()

	// Testing
	_, _, err = conn.ReadFrom(buffer)
	if !(err != nil && strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION)) {
		t.Fatalf(MSG_UNEXPECTED_SUFFIX)
	}
}

func TestWriteToOnClosedConn(t *testing.T) {
	t.Parallel()
	// Initialization
	laddr, err := net.ResolveUDPAddr("udp", DEFAULT_IP)
	if err != nil {
		t.Skipf(MSG_SKIP_ERR)
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		t.Skipf(MSG_SKIP_ERR)
	}

	// Setting Up
	conn.Close()

	// Testing
	_, err = conn.WriteTo([]byte("estarabim"), laddr)
	if !(err != nil && strings.HasSuffix(err.Error(), MSG_CLOSED_CONNECTION)) {
		t.Fatalf(MSG_UNEXPECTED_SUFFIX)
	}
}
