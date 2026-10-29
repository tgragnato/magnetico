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
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.output
}

func (m *Manager) onIndexingResult(res mainline.IndexingResult) {
	m.mu.RLock()
	select {
	case m.output <- res:
		m.mu.RUnlock()
		return
	default:
		m.mu.RUnlock()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case m.output <- res:
		return
	default:
		oldChan := m.output
		newChan := make(chan Result, cap(oldChan)+10)
		m.output = newChan
		for {
			select {
			case oldRes := <-oldChan:
				newChan <- oldRes
			default:
				close(oldChan)
				newChan <- res
				return
			}
		}
	}
}

func (m *Manager) Terminate() {
	for _, service := range m.indexingServices {
		service.Terminate()
	}
}
