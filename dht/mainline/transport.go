package mainline

import (
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"tgragnato.it/magnetico/v2/bencode"
	"tgragnato.it/magnetico/v2/persistence"
	"tgragnato.it/magnetico/v2/stats"
)

// OutboundRateLimiter spaces outgoing datagrams across all transports sharing it.
type OutboundRateLimiter struct {
	interval time.Duration
	mu       sync.Mutex
	next     time.Time
}

type OutboundQueueState struct {
	pending atomic.Uint64
	limit   uint64
}

// NewOutboundQueueState creates the shared pending-message counter and its high-water mark.
func NewOutboundQueueState(limit uint64) *OutboundQueueState {
	return &OutboundQueueState{limit: limit}
}

// Enqueue records a message waiting in the shared outbox.
func (s *OutboundQueueState) Enqueue() {
	s.pending.Add(1)
}

// Acknowledge removes one successfully sent message from the pending count.
func (s *OutboundQueueState) Acknowledge() {
	for {
		current := s.pending.Load()
		if current == 0 || s.pending.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// Full reports whether the shared pending count reached its high-water mark.
func (s *OutboundQueueState) Full() bool {
	return s != nil && s.pending.Load() >= s.limit
}

// NewOutboundRateLimiter creates a shared limiter; zero leaves sending unlimited.
func NewOutboundRateLimiter(maxRPS uint) *OutboundRateLimiter {
	limiter := &OutboundRateLimiter{}
	if maxRPS > 0 {
		limiter.interval = time.Second / time.Duration(maxRPS)
		if limiter.interval == 0 {
			limiter.interval = time.Nanosecond
		}
	}
	return limiter
}

// Wait reserves the next globally available send slot or returns on cancellation.
func (l *OutboundRateLimiter) Wait(ctx context.Context) error {
	if l == nil || l.interval <= 0 {
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	now := time.Now()
	if l.next.Before(now) {
		l.next = now
	}
	sendAt := l.next
	l.next = sendAt.Add(l.interval)
	l.mu.Unlock()

	if delay := time.Until(sendAt); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return ctx.Err()
}

type Transport struct {
	conn    *net.UDPConn
	laddr   *net.UDPAddr
	started bool
	buffer  []byte

	// OnMessage is the function that will be called when Transport receives a packet that is
	// successfully unmarshalled as a syntactically correct Message (but -of course- the checking
	// the semantic correctness of the Message is left to Protocol).
	onMessage func(*Message, *net.UDPAddr)

	maxNeighbors         uint
	queuedCommunications uint64
	outbox               persistence.Database
	outboundQueue        *OutboundQueueState
	ctx                  context.Context
}

func NewTransport(laddr string, onMessage func(*Message, *net.UDPAddr), maxNeighbors uint) *Transport {
	t := new(Transport)
	/*   The field size sets a theoretical limit of 65,535 bytes (8 byte header + 65,527 bytes of
	 * data) for a UDP datagram. However the actual limit for the data length, which is imposed by
	 * the underlying IPv4 protocol, is 65,507 bytes (65,535 − 8 byte UDP header − 20 byte IP
	 * header).
	 *
	 *   In IPv6 jumbograms it is possible to have UDP packets of size greater than 65,535 bytes.
	 * RFC 2675 specifies that the length field is set to zero if the length of the UDP header plus
	 * UDP data is greater than 65,535.
	 *
	 * https://en.wikipedia.org/wiki/User_Datagram_Protocol
	 */
	t.buffer = make([]byte, 65507)
	t.onMessage = onMessage
	t.maxNeighbors = maxNeighbors
	t.queuedCommunications = 0

	var err error
	t.laddr, err = net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		panic("Could not resolve the UDP address for the trawler! " + err.Error())
	}

	return t
}

// ConfigureOutboundQueue routes outgoing messages through the shared persistence outbox.
func (t *Transport) ConfigureOutboundQueue(database persistence.Database, queueState *OutboundQueueState) {
	t.outbox = database
	t.outboundQueue = queueState
}

func (t *Transport) Start(ctx context.Context) {
	// Why check whether the Transport `t` started or not, here and not -for instance- in
	// t.Terminate()?
	// Because in t.Terminate() the programmer (i.e. you & me) would stumble upon an error while
	// trying close an uninitialised net.UDPConn or something like that: it's mostly harmless
	// because its effects are immediate. But if you try to start a Transport `t` for the second
	// (or the third, 4th, ...) time, it will keep spawning goroutines and any small mistake may
	// end up in a debugging horror.
	//                                                                   Here ends my justification.
	if t.started {
		panic("Attempting to Start() a mainline/Transport that has been already started!")
	}
	t.started = true
	t.ctx = ctx

	var err error
	t.conn, err = net.ListenUDP("udp", t.laddr)
	if err != nil {
		log.Fatalf("Could NOT bind the socket! %s\n", err.Error())
	}

	go t.readMessages()
}

func (t *Transport) Terminate() {
	if t.conn != nil {
		_ = t.conn.Close()
	}
}

// readMessages is a goroutine!
func (t *Transport) readMessages() {
	for {
		n, from, err := t.conn.ReadFromUDP(t.buffer)
		if err != nil {
			go stats.GetInstance().IncUDPError(false)
			break
		}

		if n == 0 {
			/* Datagram sockets in various domains  (e.g., the UNIX and Internet domains) permit
			 * zero-length datagrams. When such a datagram is received, the return value (n) is 0.
			 */
			continue
		}

		var msg Message
		err = bencode.Unmarshal(t.buffer[:n], &msg)
		if err != nil {
			// couldn't unmarshal packet data
			continue
		}

		t.onMessage(&msg, from)
	}
}

func (t *Transport) WriteMessages(msg *Message, addr *net.UDPAddr) error {
	if msg == nil || addr == nil {
		return nil
	}

	data, err := bencode.Marshal(msg)
	if err != nil {
		return err
	}

	if t.outbox == nil {
		return t.writeDatagram(data, addr)
	}
	message := persistence.DHTOutboundMessage{
		Destination: *addr,
		Payload:     data,
	}
	if t.outboundQueue != nil {
		t.outboundQueue.Enqueue()
	} else {
		atomic.AddUint64(&t.queuedCommunications, 1)
	}
	if err := t.outbox.EnqueueDHTMessage(t.ctx, message); err != nil {
		if t.outboundQueue != nil {
			t.outboundQueue.Acknowledge()
		} else {
			t.decrementQueued()
		}
		return err
	}
	return nil
}

func (t *Transport) writeDatagram(payload []byte, addr *net.UDPAddr) error {
	if t.conn == nil {
		return net.ErrClosed
	}
	_, err := t.conn.WriteToUDP(payload, addr)
	log.Printf("Sent datagram to %s, err: %v\n", addr.String(), err)
	return err
}

func (t *Transport) SendDatagram(payload []byte, addr *net.UDPAddr) error {
	return t.writeDatagram(payload, addr)
}

func (t *Transport) decrementQueued() {
	for {
		current := atomic.LoadUint64(&t.queuedCommunications)
		if current == 0 || atomic.CompareAndSwapUint64(&t.queuedCommunications, current, current-1) {
			return
		}
	}
}

// Transport is full if the queued communications are more than the maximum neighbors.
func (t *Transport) Full() bool {
	if t.outboundQueue != nil {
		return t.outboundQueue.Full()
	}
	return t.queuedCommunications >= uint64(t.maxNeighbors)
}
