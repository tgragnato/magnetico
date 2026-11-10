package persistence

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type bitmagnet struct {
	url        string
	debug      bool
	sourceName string
	cache      map[string]time.Time
	dhtRouting *bitmagnetRoutingTable
	dhtOutbox  []DHTOutboundMessage
	sync.Mutex
}

type bitmagnetRoutingTable struct {
	sync.RWMutex
	nodes        map[string]uint
	maxNeighbors uint
	filterNodes  []net.IPNet
	infoHashes   [10][20]byte
}

func (b *bitmagnet) ConfigureDHTRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) error {
	b.Lock()
	defer b.Unlock()
	if b.dhtRouting != nil {
		return nil
	}
	b.dhtRouting = &bitmagnetRoutingTable{
		nodes:        make(map[string]uint),
		maxNeighbors: maxNeighbors,
		filterNodes:  append([]net.IPNet(nil), filterNodes...),
	}
	return nil
}

func (b *bitmagnet) dhtRoutingTable() (*bitmagnetRoutingTable, error) {
	b.Lock()
	defer b.Unlock()
	if b.dhtRouting == nil {
		return nil, errors.New("DHT routing table is not configured")
	}
	return b.dhtRouting, nil
}

func (b *bitmagnet) AddDHTNodes(ctx context.Context, nodes []net.UDPAddr) error {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return err
	}
	return rt.AddNodes(ctx, nodes)
}

func (b *bitmagnet) GetDHTNodes(ctx context.Context) ([]net.UDPAddr, error) {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return nil, err
	}
	return rt.GetNodes(ctx)
}

func (b *bitmagnet) IsDHTRoutingEmpty(ctx context.Context) (bool, error) {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return false, err
	}
	return rt.IsEmpty(ctx)
}

func (b *bitmagnet) DumpDHTNodes(ctx context.Context, ipv4 bool) ([]net.UDPAddr, error) {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return nil, err
	}
	return rt.Dump(ctx, ipv4)
}

func (b *bitmagnet) AddDHTHashes(ctx context.Context, hashes [][20]byte) error {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return err
	}
	return rt.AddHashes(ctx, hashes)
}

func (b *bitmagnet) GetDHTHashes(ctx context.Context) ([10][20]byte, error) {
	rt, err := b.dhtRoutingTable()
	if err != nil {
		return [10][20]byte{}, err
	}
	return rt.GetHashes(ctx)
}

func (b *bitmagnet) EnqueueDHTMessage(ctx context.Context, message DHTOutboundMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message, err := prepareDHTOutboundMessage(message)
	if err != nil {
		return err
	}
	b.Lock()
	defer b.Unlock()
	b.dhtOutbox = append(b.dhtOutbox, message)
	return nil
}

func (b *bitmagnet) NextDHTMessage(ctx context.Context) (DHTOutboundMessage, bool, error) {
	if err := ctx.Err(); err != nil {
		return DHTOutboundMessage{}, false, err
	}
	b.Lock()
	defer b.Unlock()
	if len(b.dhtOutbox) == 0 {
		return DHTOutboundMessage{}, false, nil
	}
	return b.dhtOutbox[0], true, nil
}

func (b *bitmagnet) AckDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.Lock()
	defer b.Unlock()
	for i, message := range b.dhtOutbox {
		if message.ID == messageID {
			b.dhtOutbox = append(b.dhtOutbox[:i], b.dhtOutbox[i+1:]...)
			return nil
		}
	}
	return nil
}

func (b *bitmagnet) ReleaseDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.Lock()
	defer b.Unlock()
	for i, message := range b.dhtOutbox {
		if message.ID == messageID {
			b.dhtOutbox = append(b.dhtOutbox[:i], b.dhtOutbox[i+1:]...)
			b.dhtOutbox = append(b.dhtOutbox, message)
			return nil
		}
	}
	return nil
}

func (rt *bitmagnetRoutingTable) AddNodes(ctx context.Context, nodes []net.UDPAddr) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	filteredNodes := make([]net.UDPAddr, 0, len(nodes))
	for _, node := range nodes {
		if rt.isAllowed(node) {
			filteredNodes = append(filteredNodes, node)
		}
	}
	if len(filteredNodes) == 0 {
		return nil
	}

	rt.Lock()
	defer rt.Unlock()

	if len(rt.nodes)+len(filteredNodes) > int(rt.maxNeighbors*rt.maxNeighbors) {
		clear(rt.nodes)
	}
	for _, node := range filteredNodes {
		rt.nodes[node.String()]++
	}
	return nil
}

func (rt *bitmagnetRoutingTable) GetNodes(ctx context.Context) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rt.Lock()
	defer rt.Unlock()

	nodes := make([]net.UDPAddr, 0)
	for node := range rt.nodes {
		if uint(len(nodes)) >= rt.maxNeighbors {
			break
		}
		addr, err := bitmagnetAddrFromString(node)
		if err != nil {
			continue
		}
		nodes = append(nodes, addr)
		if rt.nodes[node] > 1 {
			rt.nodes[node]--
		} else {
			delete(rt.nodes, node)
		}
	}
	return nodes, nil
}

func (rt *bitmagnetRoutingTable) IsEmpty(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	rt.RLock()
	defer rt.RUnlock()
	return len(rt.nodes) == 0, nil
}

func (rt *bitmagnetRoutingTable) Dump(ctx context.Context, ipv4 bool) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rt.RLock()
	defer rt.RUnlock()

	nodes := make([]net.UDPAddr, 0, 100)
	for node := range rt.nodes {
		if len(nodes) >= 100 {
			break
		}
		addr, err := bitmagnetAddrFromString(node)
		if err != nil {
			continue
		}
		if ipv4 && addr.IP.To4() != nil || !ipv4 && addr.IP.To4() == nil {
			nodes = append(nodes, addr)
		}
	}
	return nodes, nil
}

func (rt *bitmagnetRoutingTable) AddHashes(ctx context.Context, hashes [][20]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rt.Lock()
	defer rt.Unlock()
	for i := 0; i < len(hashes) && i < len(rt.infoHashes); i++ {
		rt.infoHashes[i] = hashes[i]
	}
	return nil
}

func (rt *bitmagnetRoutingTable) GetHashes(ctx context.Context) ([10][20]byte, error) {
	if err := ctx.Err(); err != nil {
		return [10][20]byte{}, err
	}
	rt.RLock()
	defer rt.RUnlock()
	return rt.infoHashes, nil
}

func (rt *bitmagnetRoutingTable) isAllowed(node net.UDPAddr) bool {
	if len(rt.filterNodes) > 0 {
		for _, filterNode := range rt.filterNodes {
			if filterNode.Contains(node.IP) {
				return true
			}
		}
		return false
	}

	if !node.IP.IsGlobalUnicast() || node.IP.IsPrivate() {
		return false
	}
	if node.Port != 80 && node.Port != 443 &&
		node.Port < 1024 || node.Port > 65535 {
		return false
	}
	return true
}

func bitmagnetAddrFromString(addr string) (net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return net.UDPAddr{}, err
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return net.UDPAddr{}, err
	}
	return net.UDPAddr{IP: net.ParseIP(host), Port: portNum}, nil
}

func makeBitmagnet(url_ *url.URL) (Database, error) {
	b := new(bitmagnet)

	url_.Scheme = strings.Replace(url_.Scheme, "bitmagnet", "http", 1)

	url_.ForceQuery = false
	if url_.Query().Get("debug") == "true" {
		b.debug = true
	}
	b.sourceName = url_.Query().Get("source")
	if b.sourceName == "" {
		b.sourceName = "magnetico"
	}
	url_.RawQuery = ""

	url_.Fragment = ""
	url_.RawFragment = ""

	b.url = url_.String()

	b.cache = map[string]time.Time{}
	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			go b.cleanup()
		}
	}()

	return b, nil
}

func (b *bitmagnet) cleanup() {
	b.Lock()
	defer b.Unlock()

	for key, value := range b.cache {
		if time.Now().After(value) {
			delete(b.cache, key)
		}
	}
}

func (b *bitmagnet) Engine() databaseEngine {
	return Bitmagnet
}

func (b *bitmagnet) DoesTorrentExist(infoHash []byte) (bool, error) {
	b.Lock()
	defer b.Unlock()
	_, found := b.cache[string(infoHash)]
	return found, nil
}

func (b *bitmagnet) AddNewTorrent(infoHash []byte, name string, files []File) error {
	totalSize := int64(0)
	for _, file := range files {
		totalSize += file.Size
	}
	data, err := json.Marshal(map[string]any{
		"infoHash":    hex.EncodeToString(infoHash),
		"name":        name,
		"size":        totalSize,
		"publishedAt": time.Now().UTC().Format(time.RFC3339),
		"source":      b.sourceName,
	})
	if err != nil {
		return errors.New("failed to encode metadata " + err.Error())
	}

	b.Lock()
	defer b.Unlock()

	if _, found := b.cache[string(infoHash)]; found {
		return errors.New("torrent already exists")
	}

	dataBuffer := bytes.NewBuffer(data)
	dataBuffer.Write([]byte("\n"))
	resp, err := http.Post(b.url, "application/json", dataBuffer)
	if err != nil {
		return errors.New("failed to post metadata " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if b.debug {
		log.Printf("Response: %s\n", string(body))
	}

	b.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)
	return nil
}

func (b *bitmagnet) Close() error {
	return nil
}

func (b *bitmagnet) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (b *bitmagnet) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {
	return 0, nil
}

func (b *bitmagnet) QueryTorrents(query string, epoch int64, orderBy OrderingCriteria, ascending bool, limit uint64, lastOrderedValue *float64, lastID *uint64) ([]TorrentMetadata, error) {
	return nil, errors.New("query not supported")
}

func (b *bitmagnet) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	return nil, errors.New("fetch not supported")
}

func (b *bitmagnet) GetFiles(infoHash []byte) ([]File, error) {
	return nil, errors.New("file fetch not supported")
}

func (b *bitmagnet) GetStatistics(from string, n uint) (*Statistics, error) {
	return nil, errors.New("statistics not supported")
}

func (b *bitmagnet) Export() (chan SimpleTorrentSummary, error) {
	return nil, errors.New("export not supported")
}
