package metadata

import (
	"log"
	"net"
	"time"

	"github.com/tgragnato/magnetico/dht"
	"github.com/tgragnato/magnetico/persistence"
)

const (
	// PeerIDLength The peer_id is exactly 20 bytes (characters) long.
	// https://wiki.theory.org/BitTorrentSpecification#peer_id
	PeerIDLength = 20
	// PeerPrefix Azureus-style
	PeerPrefix = "-UT3600-"
)

type Metadata struct {
	InfoHash []byte
	// Name should be thought of "Title" of the torrent. For single-file torrents, it is the name
	// of the file, and for multi-file torrents, it is the name of the root directory.
	Name         string
	TotalSize    uint64
	DiscoveredOn int64
	// Files must be populated for both single-file and multi-file torrents!
	Files []persistence.File
}

type Sink struct {
	PeerID   []byte
	deadline time.Duration
	drain    chan Metadata

	incomingInfoHashes *infoHashes

	terminated  bool
	termination chan interface{}
}

func NewSink(deadline time.Duration, maxNLeeches int) *Sink {
	ms := new(Sink)

	ms.PeerID = randomID()
	ms.deadline = deadline
	ms.drain = make(chan Metadata, 10)
	ms.incomingInfoHashes = newInfoHashes(maxNLeeches)
	ms.termination = make(chan interface{})

	return ms
}

func (ms *Sink) Sink(res dht.Result) {
	if ms.terminated {
		log.Panicln("Trying to Sink() an already closed Sink!")
	}

	infoHash := res.InfoHash()
	peerAddrs := res.PeerAddrs()
	if len(peerAddrs) <= 0 {
		return
	}

	go ms.leech(infoHash, peerAddrs[1:], peerAddrs[0])
}

func (ms *Sink) leech(infoHash [20]byte, peerAddrs []net.TCPAddr, firstPeer net.TCPAddr) {
	ms.incomingInfoHashes.push(infoHash, peerAddrs)
	NewLeech(infoHash, &firstPeer, ms.PeerID, LeechEventHandlers{
		OnSuccess: ms.flush,
		OnError:   ms.onLeechError,
	}).Do(time.Now().Add(ms.deadline))
}

func (ms *Sink) Drain() <-chan Metadata {
	if ms.terminated {
		log.Panicln("Trying to Drain() an already closed Sink!")
	}
	return ms.drain
}

func (ms *Sink) Terminate() {
	ms.terminated = true
	close(ms.termination)
	close(ms.drain)
}

func (ms *Sink) flush(result Metadata) {
	if ms.terminated {
		return
	}

	ms.drain <- result

	var infoHash [20]byte
	copy(infoHash[:], result.InfoHash)
	go ms.incomingInfoHashes.flush(infoHash)
}

func (ms *Sink) onLeechError(infoHash [20]byte, err error) {
	if peer := ms.incomingInfoHashes.pop(infoHash); peer != nil {
		go NewLeech(infoHash, peer, ms.PeerID, LeechEventHandlers{
			OnSuccess: ms.flush,
			OnError:   ms.onLeechError,
		}).Do(time.Now().Add(ms.deadline))
	}
}
