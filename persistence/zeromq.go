//go:build cgo

package persistence

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	zmq "github.com/pebbe/zmq4"
)

type zeromq struct {
	socket      *zmq.Socket
	cache       map[string]time.Time
	dhtContext  *zmq.Context
	dhtEndpoint string
	dhtDone     chan struct{}
	sync.Mutex
}

var zeromqDHTSequence atomic.Uint64

type zeromqDHTRequest struct {
	Operation    string
	MaxNeighbors uint
	FilterNodes  []net.IPNet
	Nodes        []net.UDPAddr
	Hashes       [][20]byte
	IPv4         bool
	Outbound     DHTOutboundMessage
	MessageID    string
}

type zeromqDHTResponse struct {
	Error    string
	Nodes    []net.UDPAddr
	Empty    bool
	Hashes   [10][20]byte
	Outbound DHTOutboundMessage
	Found    bool
}

type zeromqDHTState struct {
	nodes        map[string]uint
	maxNeighbors uint
	filterNodes  []net.IPNet
	hashes       [10][20]byte
	outbound     []DHTOutboundMessage
}

func makeZeroMQ(url_ *url.URL) (Database, error) {
	url_.Scheme = "tcp"
	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		return nil, err
	}
	err = socket.Bind(url_.String())
	if err != nil {
		return nil, err
	}
	instance := &zeromq{
		socket: socket,
		cache:  map[string]time.Time{},
	}
	if err := instance.startDHTRouter(); err != nil {
		_ = socket.Close()
		return nil, err
	}
	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			go instance.cleanup()
		}
	}()
	return instance, nil
}

func (instance *zeromq) cleanup() {
	instance.Lock()
	defer instance.Unlock()

	for key, value := range instance.cache {
		if time.Now().After(value) {
			delete(instance.cache, key)
		}
	}
}

func (instance *zeromq) Engine() databaseEngine {
	return ZeroMQ
}

func (instance *zeromq) startDHTRouter() error {
	ctx, err := zmq.NewContext()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("inproc://magnetico-dht-routing-%d", zeromqDHTSequence.Add(1))
	instance.dhtContext = ctx
	instance.dhtEndpoint = endpoint
	instance.dhtDone = make(chan struct{})
	ready := make(chan error, 1)
	go instance.serveDHTRouting(ready)
	if err := <-ready; err != nil {
		_ = ctx.Term()
		instance.dhtContext = nil
		instance.dhtEndpoint = ""
		instance.dhtDone = nil
		return err
	}
	return nil
}

func (instance *zeromq) serveDHTRouting(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(instance.dhtDone)
	reply, err := instance.dhtContext.NewSocket(zmq.REP)
	if err != nil {
		ready <- err
		return
	}
	if err := reply.Bind(instance.dhtEndpoint); err != nil {
		_ = reply.Close()
		ready <- err
		return
	}
	ready <- nil
	defer reply.Close()

	var state *zeromqDHTState
	for {
		payload, err := reply.RecvBytes(0)
		if err != nil {
			return
		}
		var request zeromqDHTRequest
		response := zeromqDHTResponse{}
		if err := json.Unmarshal(payload, &request); err != nil {
			response.Error = err.Error()
		} else if request.Operation == "shutdown" {
			response = zeromqDHTResponse{}
		} else {
			response, state = handleZeroMQDHTRequest(state, request)
		}
		payload, err = json.Marshal(response)
		if err != nil {
			return
		}
		if _, err := reply.SendBytes(payload, 0); err != nil {
			return
		}
		if request.Operation == "shutdown" {
			return
		}
	}
}

func handleZeroMQDHTRequest(state *zeromqDHTState, request zeromqDHTRequest) (zeromqDHTResponse, *zeromqDHTState) {
	response := zeromqDHTResponse{}
	if request.Operation == "configure" {
		if state == nil {
			state = &zeromqDHTState{nodes: make(map[string]uint)}
		}
		state.maxNeighbors = request.MaxNeighbors
		state.filterNodes = append([]net.IPNet(nil), request.FilterNodes...)
		return response, state
	}
	if state == nil {
		response.Error = "DHT routing table is not configured"
		return response, nil
	}
	switch request.Operation {
	case "add_nodes":
		filtered := make([]string, 0, len(request.Nodes))
		for _, node := range request.Nodes {
			if zeromqDHTNodeAllowed(node, state.filterNodes) {
				filtered = append(filtered, node.String())
			}
		}
		if len(state.nodes)+len(filtered) > int(state.maxNeighbors*state.maxNeighbors) {
			clear(state.nodes)
		}
		for _, node := range filtered {
			state.nodes[node]++
		}
	case "get_nodes":
		response.Nodes = make([]net.UDPAddr, 0)
		for node := range state.nodes {
			if uint(len(response.Nodes)) >= state.maxNeighbors {
				break
			}
			addr, err := zeromqDHTAddrFromString(node)
			if err != nil {
				continue
			}
			response.Nodes = append(response.Nodes, addr)
			if state.nodes[node] > 1 {
				state.nodes[node]--
			} else {
				delete(state.nodes, node)
			}
		}
	case "is_empty":
		response.Empty = len(state.nodes) == 0
	case "dump":
		response.Nodes = make([]net.UDPAddr, 0, 100)
		for node := range state.nodes {
			if len(response.Nodes) >= 100 {
				break
			}
			addr, err := zeromqDHTAddrFromString(node)
			if err != nil || request.IPv4 != (addr.IP.To4() != nil) {
				continue
			}
			response.Nodes = append(response.Nodes, addr)
		}
	case "add_hashes":
		for i := 0; i < len(request.Hashes) && i < len(state.hashes); i++ {
			state.hashes[i] = request.Hashes[i]
		}
	case "get_hashes":
		response.Hashes = state.hashes
	case "enqueue_message":
		state.outbound = append(state.outbound, request.Outbound)
	case "next_message":
		if len(state.outbound) > 0 {
			response.Outbound = state.outbound[0]
			response.Found = true
		}
	case "ack_message":
		for i, message := range state.outbound {
			if message.ID == request.MessageID {
				state.outbound = append(state.outbound[:i], state.outbound[i+1:]...)
				break
			}
		}
	case "release_message":
		for i, message := range state.outbound {
			if message.ID == request.MessageID {
				state.outbound = append(state.outbound[:i], state.outbound[i+1:]...)
				state.outbound = append(state.outbound, message)
				break
			}
		}
	default:
		response.Error = "unknown DHT routing operation"
	}
	return response, state
}

func zeromqDHTNodeAllowed(node net.UDPAddr, filterNodes []net.IPNet) bool {
	if len(filterNodes) > 0 {
		for _, filterNode := range filterNodes {
			if filterNode.Contains(node.IP) {
				return true
			}
		}
		return false
	}
	if !node.IP.IsGlobalUnicast() || node.IP.IsPrivate() {
		return false
	}
	return node.Port == 80 || node.Port == 443 || node.Port >= 1024 && node.Port <= 65535
}

func zeromqDHTAddrFromString(value string) (net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return net.UDPAddr{}, err
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return net.UDPAddr{}, err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return net.UDPAddr{}, errors.New("invalid DHT node IP")
	}
	return net.UDPAddr{IP: ip, Port: portNumber}, nil
}

func (instance *zeromq) callDHT(ctx context.Context, request zeromqDHTRequest) (zeromqDHTResponse, error) {
	if err := ctx.Err(); err != nil {
		return zeromqDHTResponse{}, err
	}
	if instance.dhtContext == nil || instance.dhtEndpoint == "" {
		return zeromqDHTResponse{}, errors.New("ZeroMQ DHT router is not started")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	client, err := instance.dhtContext.NewSocket(zmq.REQ)
	if err != nil {
		return zeromqDHTResponse{}, err
	}
	defer client.Close()
	if err := client.SetSndtimeo(5 * time.Second); err != nil {
		return zeromqDHTResponse{}, err
	}
	if err := client.SetRcvtimeo(5 * time.Second); err != nil {
		return zeromqDHTResponse{}, err
	}
	if err := client.Connect(instance.dhtEndpoint); err != nil {
		return zeromqDHTResponse{}, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return zeromqDHTResponse{}, err
	}
	if _, err := client.SendBytes(payload, 0); err != nil {
		return zeromqDHTResponse{}, err
	}
	payload, err = client.RecvBytes(0)
	if err != nil {
		return zeromqDHTResponse{}, err
	}
	var response zeromqDHTResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return zeromqDHTResponse{}, err
	}
	if response.Error != "" {
		return response, errors.New(response.Error)
	}
	return response, nil
}

func (instance *zeromq) ConfigureDHTRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) error {
	_, err := instance.callDHT(context.Background(), zeromqDHTRequest{
		Operation:    "configure",
		MaxNeighbors: maxNeighbors,
		FilterNodes:  filterNodes,
	})
	return err
}

func (instance *zeromq) AddDHTNodes(ctx context.Context, nodes []net.UDPAddr) error {
	_, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "add_nodes", Nodes: nodes})
	return err
}

func (instance *zeromq) GetDHTNodes(ctx context.Context) ([]net.UDPAddr, error) {
	response, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "get_nodes"})
	return response.Nodes, err
}

func (instance *zeromq) IsDHTRoutingEmpty(ctx context.Context) (bool, error) {
	response, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "is_empty"})
	return response.Empty, err
}

func (instance *zeromq) DumpDHTNodes(ctx context.Context, ipv4 bool) ([]net.UDPAddr, error) {
	response, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "dump", IPv4: ipv4})
	return response.Nodes, err
}

func (instance *zeromq) AddDHTHashes(ctx context.Context, hashes [][20]byte) error {
	_, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "add_hashes", Hashes: hashes})
	return err
}

func (instance *zeromq) GetDHTHashes(ctx context.Context) ([10][20]byte, error) {
	response, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "get_hashes"})
	return response.Hashes, err
}

func (instance *zeromq) EnqueueDHTMessage(ctx context.Context, message DHTOutboundMessage) error {
	message, err := prepareDHTOutboundMessage(message)
	if err != nil {
		return err
	}
	_, err = instance.callDHT(ctx, zeromqDHTRequest{Operation: "enqueue_message", Outbound: message})
	return err
}

func (instance *zeromq) NextDHTMessage(ctx context.Context) (DHTOutboundMessage, bool, error) {
	response, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "next_message"})
	return response.Outbound, response.Found, err
}

func (instance *zeromq) AckDHTMessage(ctx context.Context, messageID string) error {
	_, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "ack_message", MessageID: messageID})
	return err
}

func (instance *zeromq) ReleaseDHTMessage(ctx context.Context, messageID string) error {
	_, err := instance.callDHT(ctx, zeromqDHTRequest{Operation: "release_message", MessageID: messageID})
	return err
}

func (instance *zeromq) DoesTorrentExist(infoHash []byte) (bool, error) {
	instance.Lock()
	defer instance.Unlock()
	_, found := instance.cache[string(infoHash)]
	return found, nil
}

func (instance *zeromq) AddNewTorrent(infoHash []byte, name string, files []File) error {
	data, err := json.Marshal(SimpleTorrentSummary{
		InfoHash: hex.EncodeToString(infoHash),
		Name:     name,
		Files:    files,
	})
	if err != nil {
		return errors.New("failed to encode metadata " + err.Error())
	}

	instance.Lock()
	defer instance.Unlock()

	if _, found := instance.cache[string(infoHash)]; found {
		return errors.New("torrent already exists")
	}
	instance.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)

	_, err = instance.socket.SendMessage(data)
	return err
}

func (instance *zeromq) Close() error {
	if instance.dhtDone != nil {
		if _, err := instance.callDHT(context.Background(), zeromqDHTRequest{Operation: "shutdown"}); err != nil {
			_ = instance.dhtContext.Term()
			<-instance.dhtDone
			return err
		}
		<-instance.dhtDone
	}
	if instance.dhtContext != nil {
		if err := instance.dhtContext.Term(); err != nil {
			return err
		}
	}
	if instance.socket != nil {
		return instance.socket.Close()
	}
	return nil
}

func (instance *zeromq) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (instance *zeromq) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {
	return 0, nil
}

func (instance *zeromq) QueryTorrents(
	query string,
	epoch int64,
	orderBy OrderingCriteria,
	ascending bool,
	limit uint64,
	lastOrderedValue *float64,
	lastID *uint64,
) ([]TorrentMetadata, error) {
	return nil, errors.New("query not supported")
}

func (instance *zeromq) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	return nil, errors.New("fetch not supported")
}

func (instance *zeromq) GetFiles(infoHash []byte) ([]File, error) {
	return nil, errors.New("file fetch not supported")
}

func (instance *zeromq) GetStatistics(from string, n uint) (*Statistics, error) {
	return nil, errors.New("statistics not supported")
}

func (instance *zeromq) Export() (chan SimpleTorrentSummary, error) {
	return nil, errors.New("export not supported")
}
