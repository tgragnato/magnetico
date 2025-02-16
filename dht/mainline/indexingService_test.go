package mainline

import (
	"math/rand/v2"
	"net"
	"sort"
	"strconv"
	"testing"
	"time"
)

func sortNodes(nodes []net.UDPAddr) {
	sort.Slice(nodes, func(i, j int) bool {
		if ip1, ip2 := nodes[i].IP.String(), nodes[j].IP.String(); ip1 != ip2 {
			return ip1 < ip2
		}
		return nodes[i].Port < nodes[j].Port
	})
}

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

	randomPort := rand.IntN(64511) + 1024
	tests := []struct {
		name          string
		laddr         string
		maxNeighbors  uint
		eventHandlers IndexingServiceEventHandlers
	}{
		{
			name:          "Loopback Random IPv4",
			laddr:         net.JoinHostPort("127.0.0.1", strconv.Itoa(randomPort)),
			maxNeighbors:  0,
			eventHandlers: IndexingServiceEventHandlers{},
		},
		{
			name:          "Loopback Random IPv6",
			laddr:         net.JoinHostPort("::1", strconv.Itoa(randomPort)),
			maxNeighbors:  0,
			eventHandlers: IndexingServiceEventHandlers{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := NewIndexingService(tt.laddr, tt.maxNeighbors, tt.eventHandlers, []string{"dht.tgragnato.it"}, []net.IPNet{})
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

func TestOnFindNodeResponse(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")

	tests := []struct {
		name      string
		response  *Message
		addr      *net.UDPAddr
		wantNodes []net.UDPAddr
	}{
		{
			name: "Single Node",
			response: &Message{
				R: ResponseValues{
					Nodes: []CompactNodeInfo{
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			},
		},
		{
			name: "Multiple Nodes",
			response: &Message{
				R: ResponseValues{
					Nodes: []CompactNodeInfo{
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}},
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.2"), Port: 6882}},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
				{IP: net.ParseIP("127.0.0.2"), Port: 6882},
			},
		},
		{
			name: "No Nodes",
			response: &Message{
				R: ResponseValues{
					Nodes: []CompactNodeInfo{},
				},
			},
			addr:      &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantNodes: []net.UDPAddr{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
			}
			is.onFindNodeResponse(tt.response, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != len(tt.wantNodes) {
				t.Errorf("onFindNodeResponse() got %d nodes, want %d nodes", len(gotNodes), len(tt.wantNodes))
			}

			// Sort slices for a stable comparison
			sortNodes(gotNodes)
			sortNodes(tt.wantNodes)

			for i, gotNode := range gotNodes {
				if gotNode.IP.String() != tt.wantNodes[i].IP.String() || gotNode.Port != tt.wantNodes[i].Port {
					t.Errorf("onFindNodeResponse() got node %v, want node %v", gotNode, tt.wantNodes[i])
				}
			}
		})
	}
}

func TestOnAnnouncePeerQuery(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name      string
		msg       *Message
		addr      *net.UDPAddr
		wantNodes []net.UDPAddr
	}{
		{
			name: "Announce with Port",
			msg: &Message{
				A: QueryArguments{
					Port: 6881,
				},
				T: []byte("aa"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6882},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6882},
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			},
		},
		{
			name: "Announce with ImpliedPort",
			msg: &Message{
				A: QueryArguments{
					ImpliedPort: 6883,
				},
				T: []byte("bb"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6882},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6882},
				{IP: net.ParseIP("127.0.0.1"), Port: 6883},
			},
		},
		{
			name: "Announce with Port and ImpliedPort",
			msg: &Message{
				A: QueryArguments{
					Port:        6881,
					ImpliedPort: 6883,
				},
				T: []byte("cc"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6882},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6882},
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
				{IP: net.ParseIP("127.0.0.1"), Port: 6883},
			},
		},
		{
			name: "Announce with No Port",
			msg: &Message{
				A: QueryArguments{},
				T: []byte("dd"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6882},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6882},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID: randomNodeID(),
			}
			is.onAnnouncePeerQuery(tt.msg, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != len(tt.wantNodes) {
				t.Errorf("onAnnouncePeerQuery() got %d nodes, want %d nodes", len(gotNodes), len(tt.wantNodes))
			}

			// Sort slices for a stable comparison
			sortNodes(gotNodes)
			sortNodes(tt.wantNodes)

			for i, gotNode := range gotNodes {
				if gotNode.IP.String() != tt.wantNodes[i].IP.String() || gotNode.Port != tt.wantNodes[i].Port {
					t.Errorf("onAnnouncePeerQuery() got node %v, want node %v", gotNode, tt.wantNodes[i])
				}
			}
		})
	}
}

func TestOnPingQuery(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name string
		msg  *Message
		addr *net.UDPAddr
	}{
		{
			name: "Ping Query",
			msg: &Message{
				T: []byte("aa"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID: randomNodeID(),
			}
			is.onPingQuery(tt.msg, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != 1 {
				t.Errorf("onPingQuery() got %d nodes, want 1 node", len(gotNodes))
			}

			if len(gotNodes) == 1 {
				gotNode := gotNodes[0]
				if gotNode.IP.String() != tt.addr.IP.String() || gotNode.Port != tt.addr.Port {
					t.Errorf("onPingQuery() got node %v, want node %v", gotNode, tt.addr)
				}
			}
		})
	}
}

func TestOnPingORAnnouncePeerResponse(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name string
		msg  *Message
		addr *net.UDPAddr
	}{
		{
			name: "Ping OR Announce Peer Response",
			msg: &Message{
				T: []byte("aa"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID: randomNodeID(),
			}
			is.onPingORAnnouncePeerResponse(tt.msg, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != 1 {
				t.Errorf("onPingORAnnouncePeerResponse() got %d nodes, want 1 node", len(gotNodes))
			}

			if len(gotNodes) == 1 {
				gotNode := gotNodes[0]
				if gotNode.IP.String() != tt.addr.IP.String() || gotNode.Port != tt.addr.Port {
					t.Errorf("onPingORAnnouncePeerResponse() got node %v, want node %v", gotNode, tt.addr)
				}
			}
		})
	}
}

func TestOnFindNodeQuery(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name string
		msg  *Message
		addr *net.UDPAddr
	}{
		{
			name: "Find Node Query",
			msg: &Message{
				T: []byte("aa"),
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID: randomNodeID(),
			}
			is.onFindNodeQuery(tt.msg, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != 1 {
				t.Errorf("onFindNodeQuery() got %d nodes, want 1 node", len(gotNodes))
			}

			if len(gotNodes) == 1 {
				gotNode := gotNodes[0]
				if gotNode.IP.String() != tt.addr.IP.String() || gotNode.Port != tt.addr.Port {
					t.Errorf("onFindNodeQuery() got node %v, want node %v", gotNode, tt.addr)
				}
			}
		})
	}
}

func TestOnGetPeersResponse(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name          string
		msg           *Message
		addr          *net.UDPAddr
		getPeersReq   map[[2]byte][20]byte
		wantPeerAddrs []net.TCPAddr
		wantInfoHash  [20]byte
	}{
		{
			name: "Single Peer",
			msg: &Message{
				T: []byte{0, 1},
				R: ResponseValues{
					Values: []CompactPeer{
						{IP: net.ParseIP("127.0.0.1"), Port: 6881},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			getPeersReq: map[[2]byte][20]byte{
				{0, 1}: {1, 2, 3},
			},
			wantPeerAddrs: []net.TCPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			},
			wantInfoHash: [20]byte{1, 2, 3},
		},
		{
			name: "Multiple Peers",
			msg: &Message{
				T: []byte{0, 2},
				R: ResponseValues{
					Values: []CompactPeer{
						{IP: net.ParseIP("127.0.0.1"), Port: 6881},
						{IP: net.ParseIP("127.0.0.2"), Port: 6882},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			getPeersReq: map[[2]byte][20]byte{
				{0, 2}: {4, 5, 6},
			},
			wantPeerAddrs: []net.TCPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
				{IP: net.ParseIP("127.0.0.2"), Port: 6882},
			},
			wantInfoHash: [20]byte{4, 5, 6},
		},
		{
			name: "No Peers",
			msg: &Message{
				T: []byte{0, 3},
				R: ResponseValues{
					Values: []CompactPeer{},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			getPeersReq: map[[2]byte][20]byte{
				{0, 3}: {7, 8, 9},
			},
			wantPeerAddrs: []net.TCPAddr{},
			wantInfoHash:  [20]byte{7, 8, 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultChan := make(chan IndexingResult, 1)
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID:           randomNodeID(),
				getPeersRequests: tt.getPeersReq,
				eventHandlers: IndexingServiceEventHandlers{
					OnResult: func(result IndexingResult) {
						resultChan <- result
					},
				},
			}
			is.onGetPeersResponse(tt.msg, tt.addr)
			time.Sleep(time.Second)

			select {
			case result := <-resultChan:
				if result.infoHash != tt.wantInfoHash {
					t.Errorf("onGetPeersResponse() got infoHash %v, want %v", result.infoHash, tt.wantInfoHash)
				}
				if len(result.peerAddrs) != len(tt.wantPeerAddrs) {
					t.Errorf("onGetPeersResponse() got %d peerAddrs, want %d peerAddrs", len(result.peerAddrs), len(tt.wantPeerAddrs))
				}
				for i, gotPeerAddr := range result.peerAddrs {
					if gotPeerAddr.IP.String() != tt.wantPeerAddrs[i].IP.String() || gotPeerAddr.Port != tt.wantPeerAddrs[i].Port {
						t.Errorf("onGetPeersResponse() got peerAddr %v, want peerAddr %v", gotPeerAddr, tt.wantPeerAddrs[i])
					}
				}
			case <-time.After(time.Second):
				if len(tt.wantPeerAddrs) > 0 {
					t.Error("onGetPeersResponse() did not call OnResult")
				}
			}
		})
	}
}

func TestOnSampleInfohashesResponse(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	transport := &Transport{
		conn:           &net.UDPConn{},
		started:        true,
		onMessage:      func(*Message, *net.UDPAddr) {},
		throttlingRate: 0,
		maxNeighbors:   10,
	}

	tests := []struct {
		name           string
		msg            *Message
		addr           *net.UDPAddr
		wantInfoHashes [][20]byte
		wantNodes      []net.UDPAddr
	}{
		{
			name: "Single InfoHash",
			msg: &Message{
				R: ResponseValues{
					Samples:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
					Num:      1,
					Interval: 30,
					Nodes: []CompactNodeInfo{
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantInfoHashes: [][20]byte{
				{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			},
		},
		{
			name: "Multiple InfoHashes",
			msg: &Message{
				R: ResponseValues{
					Samples: []byte{
						1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
						21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
					},
					Num:      2,
					Interval: 30,
					Nodes: []CompactNodeInfo{
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}},
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.2"), Port: 6882}},
					},
				},
			},
			addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantInfoHashes: [][20]byte{
				{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
				{21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40},
			},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
				{IP: net.ParseIP("127.0.0.2"), Port: 6882},
			},
		},
		{
			name: "No InfoHashes",
			msg: &Message{
				R: ResponseValues{
					Samples:  []byte{},
					Num:      0,
					Interval: 30,
					Nodes: []CompactNodeInfo{
						{Addr: net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}},
					},
				},
			},
			addr:           &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			wantInfoHashes: [][20]byte{},
			wantNodes: []net.UDPAddr{
				{IP: net.ParseIP("127.0.0.1"), Port: 6881},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := &IndexingService{
				nodes: newRoutingTable(
					10,
					[]net.IPNet{*cidr},
				),
				protocol: &Protocol{
					transport: transport,
				},
				nodeID:           randomNodeID(),
				getPeersRequests: make(map[[2]byte][20]byte),
			}
			is.onSampleInfohashesResponse(tt.msg, tt.addr)
			time.Sleep(time.Second)

			gotNodes := is.nodes.getNodes()
			if len(gotNodes) != len(tt.wantNodes) {
				t.Errorf("onSampleInfohashesResponse() got %d nodes, want %d nodes", len(gotNodes), len(tt.wantNodes))
			}

			// Sort slices for a stable comparison
			sortNodes(gotNodes)
			sortNodes(tt.wantNodes)

			for i, gotNode := range gotNodes {
				if gotNode.IP.String() != tt.wantNodes[i].IP.String() || gotNode.Port != tt.wantNodes[i].Port {
					t.Errorf("onSampleInfohashesResponse() got node %v, want node %v", gotNode, tt.wantNodes[i])
				}
			}
		})
	}
}

func TestOnGetPeersQuery(t *testing.T) {
	t.Parallel()

	_, cidr, _ := net.ParseCIDR("127.0.0.0/8")
	is := &IndexingService{
		nodes: newRoutingTable(
			10,
			[]net.IPNet{*cidr},
		),
		protocol: NewProtocol("0.0.0.0:0", ProtocolEventHandlers{}, 1000),
	}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6881}

	is.onGetPeersQuery(NewGetPeersQuery(randomNodeID(), randomNodeID()), addr)
	time.Sleep(time.Second)

	gotNodes := is.nodes.getNodes()
	if len(gotNodes) != 1 {
		t.Errorf("onGetPeersQuery() got %d nodes, want 1 node", len(gotNodes))
	}
	for _, gotNode := range gotNodes {
		if gotNode.IP.String() != addr.IP.String() || gotNode.Port != addr.Port {
			t.Errorf("onGetPeersQuery() got node %v, want node %v", gotNode, addr)
		}
	}
}
