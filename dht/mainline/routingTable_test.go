package mainline

import (
	"net"
	"testing"
)

func Test_routingTable_isEmpty(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		rt := newRoutingTable(0, nil)
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("empty adding port 0", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(1, 1, 1, 1), Port: 0}})
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("empty with loopback", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(127, 0, 0, 1), Port: 1234}})
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("empty with private address", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(192, 168, 0, 1), Port: 1234}})
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("not empty 80", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(1, 1, 1, 1), Port: 80}})
		if rt.isEmpty() {
			t.Error("expected non-empty routing table")
		}
	})

	t.Run("empty 123", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(1, 1, 1, 1), Port: 123}})
		if !rt.isEmpty() {
			t.Error("expected empty routing table")
		}
	})

	t.Run("not empty 443", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(1, 1, 1, 1), Port: 443}})
		if rt.isEmpty() {
			t.Error("expected non-empty routing table")
		}
	})

	t.Run("not empty 1234", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{{IP: net.IPv4(1, 1, 1, 1), Port: 1234}})
		if rt.isEmpty() {
			t.Error("expected non-empty routing table")
		}
	})
}

func Test_routingTable_dump(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		nodes := rt.dump()
		if len(nodes) != 0 {
			t.Error("expected empty node list")
		}
	})

	t.Run("less than 10 nodes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{
			{IP: net.IPv4(1, 1, 1, 1), Port: 1234},
			{IP: net.IPv4(2, 2, 2, 2), Port: 5678},
		})
		nodes := rt.dump()
		if len(nodes) != 2 {
			t.Error("expected 2 nodes")
		}
	})

	t.Run("more than 10 nodes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		rt.addNodes([]net.UDPAddr{
			{IP: net.IPv4(1, 1, 1, 1), Port: 1234},
			{IP: net.IPv4(2, 2, 2, 2), Port: 5678},
			{IP: net.IPv4(3, 3, 3, 3), Port: 9012},
			{IP: net.IPv4(4, 4, 4, 4), Port: 3456},
			{IP: net.IPv4(5, 5, 5, 5), Port: 7890},
			{IP: net.IPv4(6, 6, 6, 6), Port: 1234},
			{IP: net.IPv4(7, 7, 7, 7), Port: 5678},
			{IP: net.IPv4(8, 8, 8, 8), Port: 9012},
			{IP: net.IPv4(9, 9, 9, 9), Port: 3456},
			{IP: net.IPv4(10, 10, 10, 10), Port: 7890},
			{IP: net.IPv4(11, 11, 11, 11), Port: 1234},
		})
		nodes := rt.dump()
		if len(nodes) != 10 {
			t.Error("expected 10 nodes")
		}
	})
}
