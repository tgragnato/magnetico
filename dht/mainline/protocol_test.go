package mainline

import (
	"net"
	"testing"
)

var protocolTest_validInstances = []struct {
	validator func(*Message) bool
	msg       Message
}{
	// ping Query:
	{
		validator: validatePingQueryMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "ping",
			A: QueryArguments{
				ID: []byte("abcdefghij0123456789"),
			},
		},
	},
	// ping or announce_peer Response:
	// Also, includes NUL and EOT characters as transaction ID (`t`).
	{
		validator: validatePingORannouncePeerResponseMessage,
		msg: Message{
			T: []byte("\x00\x04"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// find_node Query:
	{
		validator: validateFindNodeQueryMessage,
		msg: Message{
			T: []byte("\x09\x0a"),
			Y: "q",
			Q: "find_node",
			A: QueryArguments{
				ID:     []byte("abcdefghij0123456789"),
				Target: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// find_node Response with no nodes (`nodes` key still exists):
	{
		validator: validateFindNodeResponseMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{},
			},
		},
	},
	// find_node Response with a single node:
	{
		validator: validateFindNodeResponseMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
				},
			},
		},
	},
	// find_node Response with 8 nodes (all the same except the very last one):
	{
		validator: validateFindNodeResponseMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID: []byte("0123456789abcdefghij"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("zyxwvutsrqponmlkjihg"),
						Addr: net.UDPAddr{IP: []byte("\xf5\x8e\x82\x8b"), Port: 6931, Zone: ""},
					},
				},
			},
		},
	},
	// get_peers Query:
	{
		validator: validateGetPeersQueryMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "get_peers",
			A: QueryArguments{
				ID:       []byte("abcdefghij0123456789"),
				InfoHash: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
	// get_peers Response with 2 peers (`values`):
	{
		validator: validateGetPeersResponseMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("abcdefghij0123456789"),
				Token: []byte("aoeusnth"),
				Values: []CompactPeer{
					{IP: []byte("axje"), Port: 11893},
					{IP: []byte("idht"), Port: 28269},
				},
			},
		},
	},
	// get_peers Response with 2 closest nodes (`nodes`):
	{
		validator: validateGetPeersResponseMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "r",
			R: ResponseValues{
				ID:    []byte("abcdefghij0123456789"),
				Token: []byte("aoeusnth"),
				Nodes: []CompactNodeInfo{
					{
						ID:   []byte("abcdefghijklmnopqrst"),
						Addr: net.UDPAddr{IP: []byte("\x8b\x82\x8e\xf5"), Port: 3169, Zone: ""},
					},
					{
						ID:   []byte("zyxwvutsrqponmlkjihg"),
						Addr: net.UDPAddr{IP: []byte("\xf5\x8e\x82\x8b"), Port: 6931, Zone: ""},
					},
				},
			},
		},
	},
	// announce_peer Query without optional `implied_port` argument:
	{
		validator: validateAnnouncePeerQueryMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "announce_peer",
			A: QueryArguments{
				ID:       []byte("abcdefghij0123456789"),
				InfoHash: []byte("mnopqrstuvwxyz123456"),
				Port:     6881,
				Token:    []byte("aoeusnth"),
			},
		},
	},
	// announce_peer Query with optional `implied_port` argument:
	{
		validator: validateAnnouncePeerQueryMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "announce_peer",
			A: QueryArguments{
				ID:          []byte("abcdefghij0123456789"),
				InfoHash:    []byte("mnopqrstuvwxyz123456"),
				ImpliedPort: 6881,
				Port:        6881,
				Token:       []byte("aoeusnth"),
			},
		},
	},
	// sample_infohashes Query
	{
		validator: validateSampleInfohashesQueryMessage,
		msg: Message{
			T: []byte("aa"),
			Y: "q",
			Q: "sample_infohashes",
			A: QueryArguments{
				ID:     []byte("abcdefghij0123456789"),
				Target: []byte("mnopqrstuvwxyz123456"),
			},
		},
	},
}

func TestValidators(t *testing.T) {
	t.Parallel()
	for i, instance := range protocolTest_validInstances {
		msg := instance.msg
		if isValid := instance.validator(&msg); !isValid {
			t.Errorf("False-positive for valid msg #%d!", i+1)
		}
	}
}

func TestNewFindNodeQuery(t *testing.T) {
	t.Parallel()
	if !validateFindNodeQueryMessage(NewFindNodeQuery([]byte("qwertyuopasdfghjklzx"), []byte("xzlkjhgfdsapouytrewq"))) {
		t.Errorf("NewFindNodeQuery returned an invalid message!")
	}
}

func TestNewFindNodeResponse(t *testing.T) {
	t.Parallel()
	if !validateFindNodeResponseMessage(NewFindNodeResponse([]byte("tt"), []byte("qwertyuopasdfghjklzx"), []CompactNodeInfo{})) {
		t.Errorf("NewFindNodeResponse returned an invalid message!")
	}
}

func TestNewPingQuery(t *testing.T) {
	t.Parallel()
	if !validatePingQueryMessage(NewPingQuery([]byte("qwertyuopasdfghjklzx"))) {
		t.Errorf("NewPingResponse returned an invalid message!")
	}
}

func TestNewPingResponse(t *testing.T) {
	t.Parallel()
	if !validatePingORannouncePeerResponseMessage(NewAnnouncePeerResponse([]byte("tt"), []byte("qwertyuopasdfghjklzx"))) {
		t.Errorf("NewPingResponse returned an invalid message!")
	}
}

func TestNewGetPeersQuery(t *testing.T) {
	t.Parallel()
	if !validateGetPeersQueryMessage(NewGetPeersQuery([]byte("qwertyuopasdfghjklzx"), []byte("xzlkjhgfdsapouytrewq"))) {
		t.Errorf("NewGetPeersQuery returned an invalid message!")
	}
}

func TestNewGetPeersResponseWithNodes(t *testing.T) {
	t.Parallel()
	if !validateGetPeersResponseMessage(NewGetPeersResponseWithNodes([]byte("tt"), []byte("qwertyuopasdfghjklzx"), []byte("token"), []CompactNodeInfo{})) {
		t.Errorf("NewGetPeersResponseWithNodes returned an invalid message!")
	}
}

func TestNewGetPeersResponseWithValues(t *testing.T) {
	t.Parallel()
	if !validateGetPeersResponseMessage(NewGetPeersResponseWithValues([]byte("tt"), []byte("qwertyuopasdfghjklzx"), []byte("token"), []CompactPeer{})) {
		t.Errorf("NewGetPeersResponseWithValues returned an invalid message!")
	}
}

func TestNewAnnouncePeerQuery(t *testing.T) {
	t.Parallel()
	if !validateAnnouncePeerQueryMessage(NewAnnouncePeerQuery([]byte("qwertyuopasdfghjklzx"), false, []byte("xzlkjhgfdsapouytrewq"), 6881, []byte("token"))) {
		t.Errorf("NewAnnouncePeerQuery returned an invalid message!")
	}
	if !validateAnnouncePeerQueryMessage(NewAnnouncePeerQuery([]byte("qwertyuopasdfghjklzx"), true, []byte("xzlkjhgfdsapouytrewq"), 6881, []byte("token"))) {
		t.Errorf("NewAnnouncePeerQuery returned an invalid message!")
	}
}

func TestNewSampleInfohashesQuery(t *testing.T) {
	t.Parallel()
	if !validateSampleInfohashesQueryMessage(NewSampleInfohashesQuery([]byte("qwertyuopasdfghjklzx"), []byte("tt"), []byte("xzlkjhgfdsapouytrewq"))) {
		t.Errorf("NewSampleInfohashesQuery returned an invalid message!")
	}
}

func TestNewProtocol(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Error("Panic while building a Transport for the Protocol")
		}
	}()

	service := new(IndexingService)
	protocol := NewProtocol("0.0.0.0:0", ProtocolEventHandlers{
		OnFindNodeResponse:           service.onFindNodeResponse,
		OnGetPeersResponse:           service.onGetPeersResponse,
		OnSampleInfohashesResponse:   service.onSampleInfohashesResponse,
		OnPingORAnnouncePeerResponse: service.onPingORAnnouncePeerResponse,
	})
	protocol.Start()
	protocol.Terminate()
}

func TestProtocol_Start(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Starting a mainline/Protocol that has been already started should panic")
		}
	}()

	service := new(IndexingService)
	protocol := NewProtocol("0.0.0.0:0", ProtocolEventHandlers{
		OnFindNodeResponse:           service.onFindNodeResponse,
		OnGetPeersResponse:           service.onGetPeersResponse,
		OnSampleInfohashesResponse:   service.onSampleInfohashesResponse,
		OnPingORAnnouncePeerResponse: service.onPingORAnnouncePeerResponse,
	})
	protocol.Start()
	protocol.Start()
}

func TestProtocol_Terminate(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Terminating a mainline/Protocol that has been already stopped should panic")
		}
	}()

	service := new(IndexingService)
	protocol := NewProtocol("0.0.0.0:0", ProtocolEventHandlers{
		OnFindNodeResponse:           service.onFindNodeResponse,
		OnGetPeersResponse:           service.onGetPeersResponse,
		OnSampleInfohashesResponse:   service.onSampleInfohashesResponse,
		OnPingORAnnouncePeerResponse: service.onPingORAnnouncePeerResponse,
	})
	protocol.Terminate()
	protocol.Terminate()
}

func TestVerifyToken_ValidToken(t *testing.T) {
	t.Parallel()

	p := &Protocol{
		tokenSecret: []byte("secret"),
	}

	address := net.IPv4(192, 168, 0, 1)
	calculatedToken := p.CalculateToken(address)

	if !p.VerifyToken(address, calculatedToken) {
		t.Error("VerifyToken returned false for a valid token")
	}
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	t.Parallel()

	p := &Protocol{
		tokenSecret: []byte("secret"),
	}

	address := net.IPv4(192, 168, 0, 1)
	invalidToken := []byte("invalid")

	if p.VerifyToken(address, invalidToken) {
		t.Error("VerifyToken returned true for an invalid token")
	}
}
