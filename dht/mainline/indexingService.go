package mainline

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"net"
	"reflect"
	"strconv"
	"time"

	"tgragnato.it/magnetico/stats"
)

type IndexingService struct {
	// Private
	protocol      *Protocol
	started       bool
	eventHandlers IndexingServiceEventHandlers

	nodeID []byte
	nodes  *routingTable

	counter          uint16
	getPeersRequests map[[2]byte][20]byte // GetPeersQuery.`t` -> infohash

	bootstrapNodes []string
}

type IndexingServiceEventHandlers struct {
	OnResult func(IndexingResult)
}

type IndexingResult struct {
	infoHash  [20]byte
	peerAddrs []net.TCPAddr
}

func (ir IndexingResult) InfoHash() [20]byte {
	return ir.infoHash
}

func (ir IndexingResult) PeerAddrs() []net.TCPAddr {
	return ir.peerAddrs
}

func NewIndexingService(laddr string, maxNeighbors uint, eventHandlers IndexingServiceEventHandlers, bootstrapNodes []string, filterNodes []net.IPNet) *IndexingService {
	service := new(IndexingService)
	service.protocol = NewProtocol(
		laddr,
		ProtocolEventHandlers{
			OnPingQuery:                  service.onPingQuery,
			OnFindNodeQuery:              service.onFindNodeQuery,
			OnGetPeersQuery:              service.onGetPeersQuery,
			OnAnnouncePeerQuery:          service.onAnnouncePeerQuery,
			OnGetPeersResponse:           service.onGetPeersResponse,
			OnFindNodeResponse:           service.onFindNodeResponse,
			OnPingORAnnouncePeerResponse: service.onPingORAnnouncePeerResponse,
			OnSampleInfohashesQuery:      service.onSampleInfohashesQuery,
			OnSampleInfohashesResponse:   service.onSampleInfohashesResponse,
		},
		maxNeighbors,
	)
	service.nodeID = randomNodeID()
	service.nodes = newRoutingTable(maxNeighbors, filterNodes)
	service.eventHandlers = eventHandlers

	service.getPeersRequests = make(map[[2]byte][20]byte)
	service.bootstrapNodes = bootstrapNodes

	return service
}

func (is *IndexingService) Start() {
	if is.started {
		panic("Attempting to Start() a mainline/IndexingService that has been already started!")
	}
	is.started = true

	is.protocol.Start()
	go is.index()
}

func (is *IndexingService) Terminate() {
	is.protocol.Terminate()
}

func (is *IndexingService) index() {
	ticker := time.NewTicker(time.Second)
	for ; true; <-ticker.C {
		if is.nodes.isEmpty() {
			is.bootstrap()
		} else if !is.protocol.transport.Full() {
			is.findNeighbors()
		}
	}
}

func (is *IndexingService) bootstrap() {
	for _, node := range is.bootstrapNodes {
		dnsName, portStr, err := net.SplitHostPort(node)
		if err != nil {
			continue
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		bootstrappingIPs := []net.IP{}
		if ipAddrs, err := net.LookupIP(dnsName); err == nil {
			bootstrappingIPs = append(bootstrappingIPs, ipAddrs...)
		}

		if len(bootstrappingIPs) == 0 || port == 0 {
			continue
		}

		for _, ip := range bootstrappingIPs {
			go is.protocol.SendMessage(
				NewFindNodeQuery(is.nodeID, randomNodeID()),
				&net.UDPAddr{IP: ip, Port: port},
			)
		}
	}

	go stats.GetInstance().IncBootstrap()
}

func (is *IndexingService) findNeighbors() {
	for _, addr := range is.nodes.getNodes() {
		go is.protocol.SendMessage(
			NewSampleInfohashesQuery(is.nodeID, []byte("aa"), randomNodeID()),
			&addr,
		)
	}
}

func (is *IndexingService) onFindNodeResponse(response *Message, addr *net.UDPAddr) {
	neighbors := []net.UDPAddr{}
	for _, node := range response.R.Nodes {
		neighbors = append(neighbors, node.Addr)
	}
	for _, node := range response.R.Nodes6 {
		neighbors = append(neighbors, node.Addr)
	}

	if len(neighbors) > 0 {
		go is.nodes.addNodes(neighbors)
	}
}

func (is *IndexingService) onGetPeersResponse(msg *Message, addr *net.UDPAddr) {
	var t [2]byte
	copy(t[:], msg.T)

	infoHash := is.getPeersRequests[t]
	// We got a response, so free the key!
	delete(is.getPeersRequests, t)

	// BEP 51 specifies that
	//     The new sample_infohashes remote procedure call requests that a remote node return a string of multiple
	//     concatenated infohashes (20 bytes each) FOR WHICH IT HOLDS GET_PEERS VALUES.
	//                                                                          ^^^^^^
	// So theoretically we should never hit the case where `values` is empty, but c'est la vie.
	if len(msg.R.Values) == 0 {
		return
	}

	peerAddrs := make([]net.TCPAddr, 0)
	for _, peer := range msg.R.Values {
		if peer.Port == 0 {
			continue
		}

		peerAddrs = append(peerAddrs, net.TCPAddr{
			IP:   peer.IP,
			Port: peer.Port,
		})
	}

	go is.eventHandlers.OnResult(IndexingResult{
		infoHash:  infoHash,
		peerAddrs: peerAddrs,
	})
}

func (is *IndexingService) onSampleInfohashesResponse(msg *Message, addr *net.UDPAddr) {
	info_hashes := [][20]byte{}

	// request samples
	for i := 0; i < len(msg.R.Samples)/20; i++ {
		var infoHash [20]byte
		copy(infoHash[:], msg.R.Samples[i:(i+1)*20])
		info_hashes = append(info_hashes, infoHash)

		msg := NewGetPeersQuery(is.nodeID, infoHash[:])
		t := toBigEndianBytes(is.counter)
		msg.T = t[:]

		is.protocol.SendMessage(msg, addr)

		is.getPeersRequests[t] = infoHash
		is.counter++
	}

	go is.nodes.addHashes(info_hashes)

	neighbors := []net.UDPAddr{}
	if msg.R.Num > len(msg.R.Samples)/20 && time.Duration(msg.R.Interval) <= time.Minute {
		neighbors = append(neighbors, *addr)
	}
	for _, node := range msg.R.Nodes {
		neighbors = append(neighbors, node.Addr)
	}
	for _, node := range msg.R.Nodes6 {
		neighbors = append(neighbors, node.Addr)
	}
	if len(neighbors) > 0 {
		go is.nodes.addNodes(neighbors)
	}
}

func (is *IndexingService) onPingORAnnouncePeerResponse(msg *Message, addr *net.UDPAddr) {
	go is.nodes.addNodes([]net.UDPAddr{*addr})
}

func (is *IndexingService) onPingQuery(msg *Message, addr *net.UDPAddr) {
	go is.nodes.addNodes([]net.UDPAddr{*addr})

	go is.protocol.SendMessage(
		NewPingResponse(msg.T, is.nodeID),
		addr,
	)
}

func (is *IndexingService) onAnnouncePeerQuery(msg *Message, addr *net.UDPAddr) {
	addresses := []net.UDPAddr{*addr}

	if msg.A.Port > 0 &&
		msg.A.Port <= 65535 &&
		addr.Port != msg.A.Port {
		addresses = append(addresses, net.UDPAddr{
			IP:   addr.IP,
			Port: msg.A.Port,
		})
	}

	if msg.A.ImpliedPort > 0 &&
		msg.A.ImpliedPort <= 65535 &&
		addr.Port != msg.A.ImpliedPort &&
		msg.A.Port != msg.A.ImpliedPort {
		addresses = append(addresses, net.UDPAddr{
			IP:   addr.IP,
			Port: msg.A.ImpliedPort,
		})
	}

	go is.nodes.addNodes(addresses)

	go is.protocol.SendMessage(
		NewAnnouncePeerResponse(msg.T, is.nodeID),
		addr,
	)
}

func (is *IndexingService) onFindNodeQuery(msg *Message, addr *net.UDPAddr) {
	compactNodeInfos := []CompactNodeInfo{}
	for _, node := range is.nodes.dump(addr.IP.To4() != nil) {
		compactNodeInfos = append(compactNodeInfos, CompactNodeInfo{
			ID:   randomNodeID(),
			Addr: node,
		})
	}

	go is.protocol.SendMessage(
		NewFindNodeResponse(msg.T, is.nodeID, compactNodeInfos),
		addr,
	)

	go is.nodes.addNodes([]net.UDPAddr{*addr})
}

func (is *IndexingService) onGetPeersQuery(msg *Message, addr *net.UDPAddr) {
	compactPeers := []CompactPeer{}
	for _, node := range is.nodes.dump(addr.IP.To4() != nil) {
		compactPeers = append(compactPeers, CompactPeer{
			IP:   node.IP,
			Port: node.Port,
		})
	}

	go is.protocol.SendMessage(
		NewGetPeersResponseWithValues(
			msg.T,
			is.nodeID,
			is.protocol.CalculateToken(addr.IP),
			compactPeers,
		),
		addr,
	)

	go is.nodes.addNodes([]net.UDPAddr{*addr})
}

func (is *IndexingService) onSampleInfohashesQuery(msg *Message, addr *net.UDPAddr) {
	go is.nodes.addNodes([]net.UDPAddr{*addr})

	// the remote is an indexer, send a find_node query to obtain some peers
	go is.protocol.SendMessage(
		NewFindNodeQuery(is.nodeID, randomNodeID()),
		addr,
	)

	hash_stream := []byte{}
	for _, info_hash := range is.nodes.getHashes() {
		if !reflect.DeepEqual(info_hash, [20]byte{}) {
			hash_stream = append(hash_stream, info_hash[:]...)
		}
	}
	if len(hash_stream) == 0 {
		return
	}

	go is.protocol.SendMessage(
		NewSampleInfohashesResponse(
			msg.T,
			is.nodeID,
			hash_stream,
		),
		addr,
	)
}

// toBigEndianBytes Convert UInt16 To BigEndianBytes
func toBigEndianBytes(v uint16) [2]byte {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], v)
	return b
}

func randomNodeID() []byte {
	nodeID := make([]byte, 20)
	_, err := rand.Read(nodeID)
	if err != nil {
		for i := 0; i < 20; i++ {
			nodeID[i] = byte(mrand.Intn(256))
		}
	}
	return nodeID
}
