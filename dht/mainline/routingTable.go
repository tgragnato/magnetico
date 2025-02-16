package mainline

import (
	"net"
	"strconv"
	"sync"

	"tgragnato.it/magnetico/stats"
)

type routingTable struct {
	sync.RWMutex
	nodes        map[string]uint
	maxNeighbors uint
	filterNodes  []net.IPNet
	info_hashes  [10][20]byte
}

func newRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) *routingTable {
	return &routingTable{
		nodes:        map[string]uint{},
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
		for node := range rt.nodes {
			delete(rt.nodes, node)
		}
		go stats.GetInstance().IncRtClearing()
	}

	for _, node := range filteredNodes {
		rt.nodes[node.String()]++
	}
}

func (rt *routingTable) getNodes() []net.UDPAddr {
	rt.Lock()
	defer rt.Unlock()

	counter := uint(0)
	nodes := []net.UDPAddr{}
	for node := range rt.nodes {
		if counter >= rt.maxNeighbors {
			break
		}
		addr, err := addrFromStr(node)
		if err != nil {
			continue
		}

		nodes = append(nodes, addr)
		if rt.nodes[node] > 1 {
			rt.nodes[node]--
		} else {
			delete(rt.nodes, node)
		}
		counter++
	}

	return nodes
}

func (rt *routingTable) isEmpty() bool {
	rt.RLock()
	defer rt.RUnlock()

	return len(rt.nodes) == 0
}

func (rt *routingTable) dump(ipv4 bool) []net.UDPAddr {
	rt.RLock()
	defer rt.RUnlock()

	counter := uint(0)
	nodes := []net.UDPAddr{}
	for node := range rt.nodes {
		if counter >= 100 {
			break
		}

		addr, err := addrFromStr(node)
		if err != nil {
			continue
		}
		if ipv4 && addr.IP.To4() != nil || !ipv4 && addr.IP.To4() == nil {
			nodes = append(nodes, addr)
		}
	}

	return nodes
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

func addrFromStr(addr string) (net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return net.UDPAddr{}, err
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return net.UDPAddr{}, err
	}
	return net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: portNum,
	}, nil
}
