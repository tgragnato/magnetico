package mainline

import (
	"net"
	"sync"
)

type routingTable struct {
	sync.RWMutex
	nodes        []net.UDPAddr
	maxNeighbors uint
	filterNodes  []net.IPNet
}

func newRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) *routingTable {
	return &routingTable{
		nodes:        make([]net.UDPAddr, 0, maxNeighbors),
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

func (rt *routingTable) addNode(node net.UDPAddr) {
	if !rt.isAllowed(node) {
		return
	}

	rt.Lock()
	defer rt.Unlock()

	rt.nodes = append(rt.nodes, node)
}

func (rt *routingTable) getNodes() []net.UDPAddr {
	rt.Lock()
	defer rt.Unlock()

	if len(rt.nodes) <= int(rt.maxNeighbors) {
		nodes := rt.nodes
		rt.nodes = []net.UDPAddr{}
		return nodes
	}

	nodes := rt.nodes[:rt.maxNeighbors]
	rt.nodes = rt.nodes[rt.maxNeighbors+1:]
	return nodes
}

func (rt *routingTable) isEmpty() bool {
	rt.RLock()
	defer rt.RUnlock()

	return len(rt.nodes) == 0
}
