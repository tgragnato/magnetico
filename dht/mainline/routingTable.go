package mainline

import (
	"net"
	"sync"

	"tgragnato.it/magnetico/stats"
)

type routingTable struct {
	sync.RWMutex
	nodes        []net.UDPAddr
	maxNeighbors uint
	filterNodes  []net.IPNet
	info_hashes  [10][20]byte
}

func newRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) *routingTable {
	return &routingTable{
		nodes:        make([]net.UDPAddr, 0, maxNeighbors*maxNeighbors),
		maxNeighbors: maxNeighbors,
		filterNodes:  filterNodes,
		info_hashes:  [10][20]byte{},
	}
}

func (rt *routingTable) isAllowed(node net.UDPAddr) bool {
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

func (rt *routingTable) addNodes(nodes []net.UDPAddr) {
	filteredNodes := []net.UDPAddr{}
	for _, node := range nodes {
		if !rt.isAllowed(node) {
			continue
		}
		filteredNodes = append(filteredNodes, node)
	}
	if len(filteredNodes) == 0 {
		return
	}

	rt.Lock()
	defer rt.Unlock()

	if len(rt.nodes)+len(filteredNodes) > int(rt.maxNeighbors*rt.maxNeighbors) {
		rt.nodes = rt.nodes[:0]
		go stats.GetInstance().IncRtClearing()
	}

	rt.nodes = append(rt.nodes, filteredNodes...)
}

func (rt *routingTable) getNodes() []net.UDPAddr {
	rt.Lock()
	defer rt.Unlock()

	if len(rt.nodes) <= int(rt.maxNeighbors) || rt.maxNeighbors < 2 {
		nodes := rt.nodes
		rt.nodes = rt.nodes[:0]
		return nodes
	}

	nodes := rt.nodes[:rt.maxNeighbors-1]
	rt.nodes = rt.nodes[rt.maxNeighbors:]
	return nodes
}

func (rt *routingTable) isEmpty() bool {
	rt.RLock()
	defer rt.RUnlock()

	return len(rt.nodes) == 0
}

func (rt *routingTable) dump(ipv4 bool) (nodes []net.UDPAddr) {
	rt.RLock()
	defer rt.RUnlock()

	for i := 0; i < 100 && i < len(rt.nodes); i++ {
		if ipv4 && rt.nodes[i].IP.To4() != nil ||
			!ipv4 && rt.nodes[i].IP.To4() == nil {
			nodes = append(nodes, rt.nodes[i])
		}
	}
	return
}

func (rt *routingTable) addHashes(info_hashes [][20]byte) {
	rt.Lock()
	defer rt.Unlock()

	for i := 0; i < len(info_hashes) && i < 10; i++ {
		rt.info_hashes[i] = info_hashes[i]
	}
}

func (rt *routingTable) getHashes() [10][20]byte {
	rt.RLock()
	defer rt.RUnlock()

	return rt.info_hashes
}
