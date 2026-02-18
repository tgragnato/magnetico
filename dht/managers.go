package dht

import (
	"net"
	"sync"

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
	mu               sync.RWMutex
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
	m.mu.Lock()
	ch := m.output
	m.mu.Unlock()
	return ch
}

func (m *Manager) onIndexingResult(res mainline.IndexingResult) {
	select {
	case m.output <- res:
		return
	default:
		// Channel full: swap to a larger channel under mutex so the consumer
		// gets the new channel from Output() on the next call. We close the
		// old channel only after draining it; the consumer may still be
		// receiving from the old channel and will get (nil, false) when closed.
		m.mu.Lock()
		oldChan := m.output
		newChan := make(chan Result, cap(oldChan)+10)
		m.output = newChan
		m.mu.Unlock()
		for oldRes := range oldChan {
			newChan <- oldRes
		}
		close(oldChan)
		newChan <- res
	}
}

func (m *Manager) Terminate() {
	for _, service := range m.indexingServices {
		service.Terminate()
	}
}
