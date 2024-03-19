package mainline

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	mrand "math/rand"
	"net"
	"time"
)

type IndexingService struct {
	// Private
	protocol      *Protocol
	started       bool
	interval      time.Duration
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

func NewIndexingService(laddr string, interval time.Duration, maxNeighbors uint, eventHandlers IndexingServiceEventHandlers, bootstrapNodes []string) *IndexingService {
	service := new(IndexingService)
	service.interval = interval
	service.protocol = NewProtocol(
		laddr,
		ProtocolEventHandlers{
			OnFindNodeResponse:           service.onFindNodeResponse,
			OnGetPeersResponse:           service.onGetPeersResponse,
			OnSampleInfohashesResponse:   service.onSampleInfohashesResponse,
			OnPingORAnnouncePeerResponse: service.onPingORAnnouncePeerResponse,
		},
	)
	service.nodeID = randomNodeID()
	service.nodes = newRoutingTable(maxNeighbors)
	service.eventHandlers = eventHandlers

	service.getPeersRequests = make(map[[2]byte][20]byte)
	service.bootstrapNodes = bootstrapNodes

	return service
}

func (is *IndexingService) Start() {
	if is.started {
		log.Panicln("Attempting to Start() a mainline/IndexingService that has been already started! (Programmer error.)")
	}
	is.started = true

	is.protocol.Start()
	go is.index()
}

func (is *IndexingService) Terminate() {
	is.protocol.Terminate()
}

func (is *IndexingService) index() {
	for range time.Tick(is.interval) {
		if is.nodes.isEmpty() {
			is.bootstrap()
		} else {
			is.findNeighbors()
		}
	}
}

func (is *IndexingService) bootstrap() {
	bootstrappingPorts := []int{80, 443, 1337, 6969, 6881, 25401}

	bootstrappingIPs := make([]net.IP, 0)

	if len(is.bootstrapNodes) == 0 {
		log.Fatal("No bootstrapping nodes configured!")
	}

	log.Println(is.bootstrapNodes)
	for _, dnsName := range is.bootstrapNodes {
		if ipAddrs, err := net.LookupIP(dnsName); err == nil {
			bootstrappingIPs = append(bootstrappingIPs, ipAddrs...)
		} else {
			log.Printf("Warning: Failed to lookup IP for bootstrap node %s: %v\n", dnsName, err)
		}
	}

	if len(bootstrappingIPs) == 0 {
		log.Println("Could NOT resolve the IP for any of the bootstrapping nodes!")
		return
	}

	for _, ip := range bootstrappingIPs {
		for _, port := range bootstrappingPorts {
			go is.protocol.SendMessage(
				NewFindNodeQuery(is.nodeID, randomNodeID()),
				&net.UDPAddr{IP: ip, Port: port},
			)
		}
	}
}

func (is *IndexingService) findNeighbors() {
	for _, addr := range is.nodes.getNodes() {
		is.protocol.SendMessage(
			NewSampleInfohashesQuery(is.nodeID, []byte("aa"), randomNodeID()),
			&addr,
		)
	}
}

func (is *IndexingService) onFindNodeResponse(response *Message, addr *net.UDPAddr) {
	for _, node := range response.R.Nodes {
		go is.nodes.addNode(node.Addr)
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

	is.eventHandlers.OnResult(IndexingResult{
		infoHash:  infoHash,
		peerAddrs: peerAddrs,
	})
}

func (is *IndexingService) onSampleInfohashesResponse(msg *Message, addr *net.UDPAddr) {
	// request samples
	for i := 0; i < len(msg.R.Samples)/20; i++ {
		var infoHash [20]byte
		copy(infoHash[:], msg.R.Samples[i:(i+1)*20])

		msg := NewGetPeersQuery(is.nodeID, infoHash[:])
		t := toBigEndianBytes(is.counter)
		msg.T = t[:]

		is.protocol.SendMessage(msg, addr)

		is.getPeersRequests[t] = infoHash
		is.counter++
	}

	if msg.R.Num > len(msg.R.Samples)/20 && time.Duration(msg.R.Interval) <= is.interval {
		go is.nodes.addNode(*addr)
	}
	for _, node := range msg.R.Nodes {
		go is.nodes.addNode(node.Addr)
	}
}

func (is *IndexingService) onPingORAnnouncePeerResponse(msg *Message, addr *net.UDPAddr) {
	is.protocol.SendMessage(
		NewAnnouncePeerResponse(msg.T, is.nodeID),
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
