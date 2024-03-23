package metadata

import (
	"net"
	"sync"
)

type infoHashes struct {
	sync.RWMutex
	infoHashes  map[[20]byte][]net.TCPAddr
	maxNLeeches int
}

func newInfoHashes(maxNLeeches int) *infoHashes {
	ih := &infoHashes{
		infoHashes:  make(map[[20]byte][]net.TCPAddr),
		maxNLeeches: maxNLeeches,
	}
	return ih
}

func (ih *infoHashes) push(infoHash [20]byte, peerAddresses []net.TCPAddr) {
	if len(peerAddresses) <= 0 {
		return
	}

	ih.Lock()
	defer ih.Unlock()

	for _, addr := range peerAddresses {
		if !addr.IP.IsGlobalUnicast() || addr.IP.IsPrivate() {
			continue
		}
		if addr.Port < 1024 || addr.Port > 65535 {
			continue
		}

		if len(ih.infoHashes[infoHash]) >= ih.maxNLeeches {
			return
		}

		if checkDuplicate(ih.infoHashes[infoHash], addr) {
			continue
		}

		ih.infoHashes[infoHash] = append(ih.infoHashes[infoHash], addr)
	}
}

func checkDuplicate(peerAddresses []net.TCPAddr, addr net.TCPAddr) bool {
	for _, existingAddr := range peerAddresses {
		if existingAddr.IP.Equal(addr.IP) &&
			existingAddr.Port == addr.Port {
			return true
		}
	}
	return false
}

func (ih *infoHashes) pop(infoHash [20]byte) *net.TCPAddr {
	ih.Lock()
	defer ih.Unlock()

	peerAddresses, exists := ih.infoHashes[infoHash]
	if !exists {
		return nil
	}
	if len(peerAddresses) <= 0 {
		go ih.flush(infoHash)
		return nil
	}

	peerAddress := peerAddresses[0]
	ih.infoHashes[infoHash] = peerAddresses[1:]
	return &peerAddress
}

func (ih *infoHashes) flush(infoHash [20]byte) {
	ih.Lock()
	defer ih.Unlock()
	delete(ih.infoHashes, infoHash)
}
