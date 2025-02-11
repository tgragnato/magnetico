package mainline

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	mrand "math/rand/v2"
	"net"
	"sync"
	"time"

	"tgragnato.it/magnetico/stats"
)

type Protocol struct {
	tokenSecret   []byte
	tokenLock     sync.Mutex
	transport     *Transport
	eventHandlers ProtocolEventHandlers
	started       bool
}

type ProtocolEventHandlers struct {
	OnPingQuery                  func(*Message, *net.UDPAddr)
	OnFindNodeQuery              func(*Message, *net.UDPAddr)
	OnGetPeersQuery              func(*Message, *net.UDPAddr)
	OnAnnouncePeerQuery          func(*Message, *net.UDPAddr)
	OnGetPeersResponse           func(*Message, *net.UDPAddr)
	OnFindNodeResponse           func(*Message, *net.UDPAddr)
	OnPingORAnnouncePeerResponse func(*Message, *net.UDPAddr)

	// Added by BEP 51
	OnSampleInfohashesQuery    func(*Message, *net.UDPAddr)
	OnSampleInfohashesResponse func(*Message, *net.UDPAddr)
}

func NewProtocol(laddr string, eventHandlers ProtocolEventHandlers, maxNeighbors uint) (p *Protocol) {
	p = new(Protocol)
	p.eventHandlers = eventHandlers
	p.transport = NewTransport(laddr, p.onMessage, maxNeighbors)
	return
}

func (p *Protocol) Start() {
	if p.started {
		panic("Attempting to Start() a mainline/Protocol that has been already started!")
	}
	p.started = true

	p.transport.Start()
	p.updateTokenSecret()

	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			p.updateTokenSecret()
		}
	}()
}

func (p *Protocol) Terminate() {
	if !p.started {
		panic("Attempted to Terminate() a mainline/Protocol that has not been Start()ed!")
	}

	p.transport.Terminate()
}

func (p *Protocol) onMessage(msg *Message, addr *net.UDPAddr) {
	switch msg.Y {
	case "q":
		switch msg.Q {
		case "ping":
			if !validatePingQueryMessage(msg) {
				return
			}
			// Check whether there is a registered event handler for the ping queries, before
			// attempting to call.
			if p.eventHandlers.OnPingQuery != nil {
				p.eventHandlers.OnPingQuery(msg, addr)
			}

		case "find_node":
			if !validateFindNodeQueryMessage(msg) {
				return
			}
			if p.eventHandlers.OnFindNodeQuery != nil {
				p.eventHandlers.OnFindNodeQuery(msg, addr)
			}

		case "get_peers":
			if !validateGetPeersQueryMessage(msg) {
				return
			}
			if p.eventHandlers.OnGetPeersQuery != nil {
				p.eventHandlers.OnGetPeersQuery(msg, addr)
			}

		case "announce_peer":
			if !validateAnnouncePeerQueryMessage(msg) {
				return
			}
			if p.eventHandlers.OnAnnouncePeerQuery != nil {
				p.eventHandlers.OnAnnouncePeerQuery(msg, addr)
			}

		case "sample_infohashes": // Added by BEP 51
			if !validateSampleInfohashesQueryMessage(msg) {
				return
			}
			if p.eventHandlers.OnSampleInfohashesQuery != nil {
				p.eventHandlers.OnSampleInfohashesQuery(msg, addr)
			}

		default:
			return
		}
	case "r":
		// Query messages have a `q` field which indicates their type but response messages have no such field that we
		// can rely on.
		// The idea is you'd use transaction ID (the `t` key) to deduce the type of a response message, as it must be
		// sent in response to a query message (with the same transaction ID) that we have sent earlier.
		// This approach is, unfortunately, not very practical for our needs since we send up to thousands messages per
		// second, meaning that we'd run out of transaction IDs very quickly (since some [many?] clients assume
		// transaction IDs are no longer than 2 bytes), and we'd also then have to consider retention too (as we might
		// not get a response at all).
		// Our approach uses an ad-hoc pattern matching: all response messages share a subset of fields (such as `t`,
		// `y`) but only one type of them contain a particular field (such as `token` field is unique to `get_peers`
		// responses, `samples` is unique to `sample_infohashes` etc).
		//
		// sample_infohashes > get_peers > find_node > ping / announce_peer
		if len(msg.R.Samples) != 0 { // The message should be a sample_infohashes response.
			if !validateSampleInfohashesResponseMessage(msg) {
				return
			}
			if p.eventHandlers.OnSampleInfohashesResponse != nil {
				p.eventHandlers.OnSampleInfohashesResponse(msg, addr)
			}
		} else if len(msg.R.Token) != 0 { // The message should be a get_peers response.
			if !validateGetPeersResponseMessage(msg) {
				return
			}
			if p.eventHandlers.OnGetPeersResponse != nil {
				p.eventHandlers.OnGetPeersResponse(msg, addr)
			}
		} else if len(msg.R.Nodes) != 0 || len(msg.R.Nodes6) != 0 { // The message should be a find_node response.
			if !validateFindNodeResponseMessage(msg) {
				return
			}
			if p.eventHandlers.OnFindNodeResponse != nil {
				p.eventHandlers.OnFindNodeResponse(msg, addr)
			}
		} else { // The message should be a ping or an announce_peer response.
			if !validatePingORannouncePeerResponseMessage(msg) {
				return
			}
			if p.eventHandlers.OnPingORAnnouncePeerResponse != nil {
				p.eventHandlers.OnPingORAnnouncePeerResponse(msg, addr)
			}
		}
	default:
	}
}

func (p *Protocol) SendMessage(msg *Message, addr *net.UDPAddr) {
	if addr.Port < 1 || addr.Port > 65535 {
		return
	}

	if err := p.transport.WriteMessages(msg, addr); err != nil {
		go stats.GetInstance().IncUDPError(true)
	}
}

func NewPingQuery(id []byte) *Message {
	return &Message{
		Y: "q",
		T: []byte("aa"),
		Q: "ping",
		A: QueryArguments{
			ID: id,
		},
	}
}

func NewFindNodeQuery(id []byte, target []byte) *Message {
	return &Message{
		Y: "q",
		T: []byte("aa"),
		Q: "find_node",
		A: QueryArguments{
			ID:     id,
			Target: target,
		},
	}
}

func NewGetPeersQuery(id []byte, infoHash []byte) *Message {
	return &Message{
		Y: "q",
		T: []byte("aa"),
		Q: "get_peers",
		A: QueryArguments{
			ID:       id,
			InfoHash: infoHash,
		},
	}
}

func NewAnnouncePeerQuery(id []byte, implied_port bool, info_hash []byte, port uint16, token []byte) *Message {
	if implied_port {
		return &Message{
			Y: "q",
			T: []byte("aa"),
			Q: "announce_peer",
			A: QueryArguments{
				ID:          id,
				ImpliedPort: int(port),
				InfoHash:    info_hash,
				Port:        int(port),
				Token:       token,
			},
		}
	}

	return &Message{
		Y: "q",
		T: []byte("aa"),
		Q: "announce_peer",
		A: QueryArguments{
			ID:       id,
			InfoHash: info_hash,
			Port:     int(port),
			Token:    token,
		},
	}
}

func NewSampleInfohashesQuery(id []byte, t []byte, target []byte) *Message {
	return &Message{
		Y: "q",
		T: t,
		Q: "sample_infohashes",
		A: QueryArguments{
			ID:     id,
			Target: target,
		},
	}
}

func NewPingResponse(t []byte, id []byte) *Message {
	return &Message{
		Y: "r",
		T: t,
		R: ResponseValues{
			ID: id,
		},
	}
}

func NewFindNodeResponse(t []byte, id []byte, nodes []CompactNodeInfo) *Message {
	// Assumes that all nodes are IPv4 or all are IPv6.
	if len(nodes) > 0 && nodes[0].Addr.IP.To4() != nil {
		return &Message{
			Y: "r",
			T: t,
			R: ResponseValues{
				ID:    id,
				Nodes: nodes,
			},
		}
	}

	return &Message{
		Y: "r",
		T: t,
		R: ResponseValues{
			ID:     id,
			Nodes6: nodes,
		},
	}
}

func NewGetPeersResponseWithValues(t []byte, id []byte, token []byte, values []CompactPeer) *Message {
	return &Message{
		Y: "r",
		T: t,
		R: ResponseValues{
			ID:     id,
			Token:  token,
			Values: values,
		},
	}
}

func NewGetPeersResponseWithNodes(t []byte, id []byte, token []byte, nodes []CompactNodeInfo) *Message {
	return &Message{
		Y: "r",
		T: t,
		R: ResponseValues{
			ID:    id,
			Token: token,
			Nodes: nodes,
		},
	}
}

func NewSampleInfohashesResponse(t []byte, id []byte, info_hashes []byte) *Message {
	return &Message{
		Y: "r",
		T: t,
		R: ResponseValues{
			ID:       id,
			Interval: 30,
			Num:      len(info_hashes) / 20,
			Samples:  info_hashes,
		},
	}
}

func NewAnnouncePeerResponse(t []byte, id []byte) *Message {
	// Because they are indistinguishable.
	return NewPingResponse(t, id)
}

func (p *Protocol) CalculateToken(address net.IP) []byte {
	p.tokenLock.Lock()
	defer p.tokenLock.Unlock()
	sum := sha1.Sum(append(p.tokenSecret, address...))
	return sum[:]
}

func (p *Protocol) VerifyToken(address net.IP, token []byte) bool {
	p.tokenLock.Lock()
	defer p.tokenLock.Unlock()
	// Compare the provided token with the calculated token
	calculatedToken := sha1.Sum(append(p.tokenSecret, address...))
	return bytes.Equal(calculatedToken[:], token)
}

func (p *Protocol) updateTokenSecret() {
	p.tokenLock.Lock()
	defer p.tokenLock.Unlock()
	_, err := rand.Read(p.tokenSecret)
	if err != nil {
		len := mrand.IntN(256)
		p.tokenSecret = make([]byte, len)
		for i := 0; i < len; i++ {
			p.tokenSecret[i] = byte(mrand.IntN(256))
		}
	}
}

func validatePingQueryMessage(msg *Message) bool {
	return len(msg.A.ID) == 20
}

func validateFindNodeQueryMessage(msg *Message) bool {
	return len(msg.A.ID) == 20 &&
		len(msg.A.Target) == 20
}

func validateGetPeersQueryMessage(msg *Message) bool {
	return len(msg.A.ID) == 20 &&
		len(msg.A.InfoHash) == 20
}

func validateAnnouncePeerQueryMessage(msg *Message) bool {
	return len(msg.A.ID) == 20 &&
		len(msg.A.InfoHash) == 20 &&
		msg.A.Port > 0 &&
		len(msg.A.Token) > 0
}

func validateSampleInfohashesQueryMessage(msg *Message) bool {
	return len(msg.A.ID) == 20 &&
		len(msg.A.Target) == 20
}

func validatePingORannouncePeerResponseMessage(msg *Message) bool {
	return len(msg.R.ID) == 20
}

func validateFindNodeResponseMessage(msg *Message) bool {
	return len(msg.R.ID) == 20 &&
		(len(msg.R.Nodes) >= 0 || len(msg.R.Nodes6) >= 0)
}

func validateGetPeersResponseMessage(msg *Message) bool {
	return len(msg.R.ID) == 20 &&
		len(msg.R.Token) > 0 &&
		(len(msg.R.Values) > 0 || len(msg.R.Nodes) >= 0 || len(msg.R.Nodes6) >= 0)
}

func validateSampleInfohashesResponseMessage(msg *Message) bool {
	return len(msg.R.ID) == 20 &&
		msg.R.Interval >= 0 &&
		(len(msg.R.Nodes) >= 0 || len(msg.R.Nodes6) >= 0) &&
		msg.R.Num >= 0 &&
		len(msg.R.Samples)%20 == 0
}
