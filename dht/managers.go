package dht

import (
	"log"
	"net"
	"time"

	"github.com/tgragnato/magnetico/dht/mainline"
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

func NewManager(addrs []string, interval time.Duration, maxNeighbors uint, bootstrappingNodes []string) *Manager {
	manager := new(Manager)
	manager.output = make(chan Result, 20)

	for _, addr := range addrs {
		service := mainline.NewIndexingService(addr, interval, maxNeighbors, mainline.IndexingServiceEventHandlers{
			OnResult: manager.onIndexingResult,
		}, bootstrappingNodes)
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
		log.Println("DHT manager output ch is full, idx result dropped!")
	}
}

func (m *Manager) Terminate() {
	for _, service := range m.indexingServices {
		service.Terminate()
	}
}
