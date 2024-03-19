package mainline

import (
	"net"
	"testing"
)

func Test_routingTable_isEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		rt := newRoutingTable(0)
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("empty adding port 0", func(t *testing.T) {
		rt := newRoutingTable(1)
		rt.addNode(net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("not empty", func(t *testing.T) {
		rt := newRoutingTable(1)
		rt.addNode(net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234})
		if rt.isEmpty() {
			t.Error("expected non-empty routing table")
		}
	})
}
