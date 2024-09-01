package mainline

import (
	"net"
	"sync"

	"github.com/tgragnato/magnetico/stats"
)

type routingTable struct {
	sync.RWMutex
	nodes        []net.UDPAddr
	maxNeighbors uint
	filterNodes  []net.IPNet
}

func newRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) *routingTable {
	return &routingTable{
		nodes:        make([]net.UDPAddr, 0, maxNeighbors*maxNeighbors),
		maxNeighbors: maxNeighbors,
		filterNodes:  filterNodes,
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

func (rt *routingTable) dump() (nodes []net.UDPAddr) {
	rt.RLock()
	defer rt.RUnlock()

	for i := 0; i < 10 && i < len(rt.nodes); i++ {
		nodes = append(nodes, rt.nodes[i])
	}
	return
}
