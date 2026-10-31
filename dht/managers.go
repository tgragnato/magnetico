package dht

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"tgragnato.it/magnetico/v2/dht/mainline"
	"tgragnato.it/magnetico/v2/persistence"
	"tgragnato.it/magnetico/v2/stats"
)

type Service interface {
	Start()
	Terminate()
	SendDatagram([]byte, *net.UDPAddr) error
}

type Result interface {
	InfoHash() [20]byte
	PeerAddrs() []net.TCPAddr
}

type Manager struct {
	mu               sync.RWMutex
	output           chan Result
	indexingServices []Service
	outboundDone     chan struct{}
}

func NewManager(ctx context.Context, addrs []string, maxNeighbors uint, bootstrappingNodes []string, filterNodes []net.IPNet, database persistence.Database, maxRPS uint) (*Manager, error) {
	if err := database.ConfigureDHTRoutingTable(maxNeighbors, filterNodes); err != nil {
		return nil, err
	}

	rateLimiter := mainline.NewOutboundRateLimiter(maxRPS)
	outboundQueue := mainline.NewOutboundQueueState(uint64(maxNeighbors) * uint64(len(addrs)))
	manager := new(Manager)
	manager.output = make(chan Result, 20)

	for _, addr := range addrs {
		service := mainline.NewIndexingService(ctx, addr, maxNeighbors, mainline.IndexingServiceEventHandlers{
			OnResult: manager.onIndexingResult,
		}, bootstrappingNodes, database, outboundQueue)
		manager.indexingServices = append(manager.indexingServices, service)
		service.Start()
	}
	manager.outboundDone = make(chan struct{})
	go manager.dispatchOutbound(ctx, database, rateLimiter, outboundQueue)

	return manager, nil
}

func (m *Manager) dispatchOutbound(ctx context.Context, database persistence.Database, rateLimiter *mainline.OutboundRateLimiter, queue *mainline.OutboundQueueState) {
	defer close(m.outboundDone)
	serviceIndex := 0
	for {
		message, found, err := database.NextDHTMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			go stats.GetInstance().IncDBError(false)
			if !waitForOutbound(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}
		if !found {
			if !waitForOutbound(ctx, 5*time.Millisecond) {
				return
			}
			continue
		}
		if err := rateLimiter.Wait(ctx); err != nil {
			return
		}
		if len(m.indexingServices) == 0 {
			_ = database.ReleaseDHTMessage(ctx, message.ID)
			return
		}

		service := m.indexingServices[serviceIndex]
		serviceIndex = (serviceIndex + 1) % len(m.indexingServices)
		if err := service.SendDatagram(message.Payload, &message.Destination); err != nil {
			go stats.GetInstance().IncUDPError(true)
			if releaseErr := database.ReleaseDHTMessage(ctx, message.ID); releaseErr != nil {
				log.Printf("Could not release DHT outbound message! %s\n", releaseErr.Error())
			}
			if !waitForOutbound(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}
		if err := database.AckDHTMessage(ctx, message.ID); err != nil {
			go stats.GetInstance().IncDBError(true)
			if !waitForOutbound(ctx, 100*time.Millisecond) {
				return
			}
			continue
		}
		queue.Acknowledge()
	}
}

func waitForOutbound(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
	if m.outboundDone != nil {
		<-m.outboundDone
	}
}
