package dht

import (
	"net"

	"tgragnato.it/magnetico/v2/dht/mainline"
)

type Service interface {
	Start()
	Terminate()
}

type Result interface {
	InfoHash() [20]byte
	PeerAddrs() []net.TCPAddr
}

type Manager struct {
	output           chan Result
	indexingServices []Service
}

func NewManager(addrs []string, maxNeighbors uint, bootstrappingNodes []string, filterNodes []net.IPNet) *Manager {
	manager := new(Manager)
	manager.output = make(chan Result, 20)

	for _, addr := range addrs {
		service := mainline.NewIndexingService(addr, maxNeighbors, mainline.IndexingServiceEventHandlers{
			OnResult: manager.onIndexingResult,
		}, bootstrappingNodes, filterNodes)
		manager.indexingServices = append(manager.indexingServices, service)
		service.Start()
	}

	return manager
}

func (m *Manager) Output() <-chan Result {
	return m.output
}

func (m *Manager) onIndexingResult(res mainline.IndexingResult) {
	select {
	case m.output <- res:
	default:
		newChan := make(chan Result, len(m.output)+10)
		for oldRes := range m.output {
			newChan <- oldRes
		}
		close(m.output)
		m.output = newChan
		m.output <- res
	}
}

func (m *Manager) Terminate() {
	for _, service := range m.indexingServices {
		service.Terminate()
	}
}
