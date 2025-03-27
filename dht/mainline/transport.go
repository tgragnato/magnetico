package mainline

import (
	"log"
	"net"
	"sync/atomic"
	"time"

	"tgragnato.it/magnetico/v2/bencode"
	"tgragnato.it/magnetico/v2/stats"
)

var (
	//Throttle rate that transport will have at Start time. Set <= 0 for unlimited requests
	DefaultThrottleRate = -1
)

type Transport struct {
	conn    *net.UDPConn
	laddr   *net.UDPAddr
	started bool
	buffer  []byte

	// OnMessage is the function that will be called when Transport receives a packet that is
	// successfully unmarshalled as a syntactically correct Message (but -of course- the checking
	// the semantic correctness of the Message is left to Protocol).
	onMessage func(*Message, *net.UDPAddr)

	throttlingRate         int           //available messages per second. If <=0, it is considered disabled
	throttleTicketsChannel chan struct{} //channel giving tickets (allowance) to make send a message
	maxNeighbors           uint
	queuedCommunications   uint64
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
	t.throttleTicketsChannel = make(chan struct{})
	t.throttlingRate = DefaultThrottleRate
	t.maxNeighbors = maxNeighbors
	t.queuedCommunications = 0

	var err error
	t.laddr, err = net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		panic("Could not resolve the UDP address for the trawler! " + err.Error())
	}

	return t
}

func (t *Transport) Start() {
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

	var err error
	t.conn, err = net.ListenUDP("udp", t.laddr)
	if err != nil {
		log.Fatalf("Could NOT bind the socket! %s\n", err.Error())
	}

	go t.readMessages()
	go t.Throttle()
}

func (t *Transport) Terminate() {
	t.conn.Close()
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

// Manages throttling for transport. To be called as a routine at Start time. Should never return.
func (t *Transport) Throttle() {
	if t.throttlingRate > 0 {
		resetChannel := make(chan struct{})

		dealer := func(resetRequest chan struct{}) {
			ticketGiven := 0
			tooManyTicketGiven := false
			for {
				select {
				case <-t.throttleTicketsChannel:
					{
						ticketGiven++
						if ticketGiven >= t.throttlingRate {
							tooManyTicketGiven = true
							break
						}
					}
				case <-resetRequest:
					{
						return
					}
				}

				if tooManyTicketGiven {
					break
				}
			}

			<-resetRequest
		}

		go dealer(resetChannel)
		for range time.Tick(1 * time.Second) {
			resetChannel <- struct{}{}

			go dealer(resetChannel)
		}

	} else {
		// no limit, keep giving tickets to whoever requests it
		for {
			<-t.throttleTicketsChannel
		}
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

	// get ticket but prioritize get_peers and find_node
	if msg.Q != "get_peers" && msg.Q != "find_node" {
		atomic.AddUint64(&t.queuedCommunications, 1)
		defer atomic.AddUint64(&t.queuedCommunications, ^uint64(0))
		t.throttleTicketsChannel <- struct{}{}
	}

	_, err = t.conn.WriteToUDP(data, addr)
	return err
}

// Transport is full if the queued communications are more than the maximum neighbors.
func (t *Transport) Full() bool {
	return t.queuedCommunications >= uint64(t.maxNeighbors)
}
