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
		nodes := rt.dump(true)
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
		nodes := rt.dump(true)
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
		nodes := rt.dump(true)
		if len(nodes) != 10 {
			t.Error("expected 10 nodes")
		}
	})
}
func Test_routingTable_isAllowed(t *testing.T) {
	t.Parallel()

	t.Run("allowed global unicast port 80", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 80}
		if !rt.isAllowed(node) {
			t.Error("expected node to be allowed")
		}
	})

	t.Run("allowed global unicast port 443", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 443}
		if !rt.isAllowed(node) {
			t.Error("expected node to be allowed")
		}
	})

	t.Run("not allowed private IP", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(192, 168, 0, 1), Port: 1234}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})

	t.Run("not allowed loopback IP", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})

	t.Run("not allowed port 0", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 0}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})

	t.Run("not allowed port 1023", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 1023}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})

	t.Run("allowed port 1024", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 1024}
		if !rt.isAllowed(node) {
			t.Error("expected node to be allowed")
		}
	})

	t.Run("not allowed port 65536", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 65536}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})

	t.Run("allowed with filter", func(t *testing.T) {
		_, ipNet, _ := net.ParseCIDR("8.8.8.0/24")
		rt := newRoutingTable(1, []net.IPNet{*ipNet})
		node := net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 1234}
		if !rt.isAllowed(node) {
			t.Error("expected node to be allowed")
		}
	})

	t.Run("not allowed with filter", func(t *testing.T) {
		_, ipNet, _ := net.ParseCIDR("8.8.8.0/24")
		rt := newRoutingTable(1, []net.IPNet{*ipNet})
		node := net.UDPAddr{IP: net.IPv4(9, 9, 9, 9), Port: 1234}
		if rt.isAllowed(node) {
			t.Error("expected node to be not allowed")
		}
	})
}

func Test_routingTable_addHashes(t *testing.T) {
	t.Parallel()

	t.Run("add less than 10 hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		hashes := [][20]byte{
			{0x01}, {0x02}, {0x03},
		}
		rt.addHashes(hashes)
		storedHashes := rt.getHashes()
		for i, hash := range hashes {
			if storedHashes[i] != hash {
				t.Errorf("expected hash %v, got %v", hash, storedHashes[i])
			}
		}
	})

	t.Run("add exactly 10 hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		hashes := [][20]byte{
			{0x01}, {0x02}, {0x03}, {0x04}, {0x05},
			{0x06}, {0x07}, {0x08}, {0x09}, {0x0A},
		}
		rt.addHashes(hashes)
		storedHashes := rt.getHashes()
		for i, hash := range hashes {
			if storedHashes[i] != hash {
				t.Errorf("expected hash %v, got %v", hash, storedHashes[i])
			}
		}
	})

	t.Run("add more than 10 hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		hashes := [][20]byte{
			{0x01}, {0x02}, {0x03}, {0x04}, {0x05},
			{0x06}, {0x07}, {0x08}, {0x09}, {0x0A},
			{0x0B}, {0x0C},
		}
		rt.addHashes(hashes)
		storedHashes := rt.getHashes()
		for i := 0; i < 10; i++ {
			if storedHashes[i] != hashes[i] {
				t.Errorf("expected hash %v, got %v", hashes[i], storedHashes[i])
			}
		}
	})

	t.Run("add empty hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		hashes := [][20]byte{}
		rt.addHashes(hashes)
		storedHashes := rt.getHashes()
		for _, hash := range storedHashes {
			if hash != [20]byte{} {
				t.Errorf("expected empty hash, got %v", hash)
			}
		}
	})
}

func Test_routingTable_getHashes(t *testing.T) {
	t.Parallel()

	t.Run("get empty hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		hashes := rt.getHashes()
		for _, hash := range hashes {
			if hash != [20]byte{} {
				t.Errorf("expected empty hash, got %v", hash)
			}
		}
	})

	t.Run("get added hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		expectedHashes := [][20]byte{
			{0x01}, {0x02}, {0x03},
		}
		rt.addHashes(expectedHashes)
		hashes := rt.getHashes()
		for i, hash := range expectedHashes {
			if hashes[i] != hash {
				t.Errorf("expected hash %v, got %v", hash, hashes[i])
			}
		}
	})

	t.Run("get exactly 10 hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		expectedHashes := [][20]byte{
			{0x01}, {0x02}, {0x03}, {0x04}, {0x05},
			{0x06}, {0x07}, {0x08}, {0x09}, {0x0A},
		}
		rt.addHashes(expectedHashes)
		hashes := rt.getHashes()
		for i, hash := range expectedHashes {
			if hashes[i] != hash {
				t.Errorf("expected hash %v, got %v", hash, hashes[i])
			}
		}
	})

	t.Run("get more than 10 hashes", func(t *testing.T) {
		rt := newRoutingTable(1, nil)
		expectedHashes := [][20]byte{
			{0x01}, {0x02}, {0x03}, {0x04}, {0x05},
			{0x06}, {0x07}, {0x08}, {0x09}, {0x0A},
			{0x0B}, {0x0C},
		}
		rt.addHashes(expectedHashes)
		hashes := rt.getHashes()
		for i := 0; i < 10; i++ {
			if hashes[i] != expectedHashes[i] {
				t.Errorf("expected hash %v, got %v", expectedHashes[i], hashes[i])
			}
		}
	})
}
