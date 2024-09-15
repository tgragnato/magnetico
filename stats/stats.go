package stats

import (
	"fmt"
	"sync"
)

// Stats saves the metrics for each particular operation.
type Stats struct {
	// bootstrap represents the number of times the DHT has been bootstrapped.
	bootstrap uint64
	// writeError represents the number of times there was an error writing a message to the UDP socket.
	writeError uint64
	// readError represents the number of times there was an error reading a message from the UDP socket.
	readError uint64
	// rtClearing represents the number of times the routing table has been cleared.
	rtClearing uint64
	// nonUTF8 represents the number of times a torrent has been ignored due to its name not being UTF-8 compliant.
	nonUTF8 uint64
	// checkError represents the number of times there was an error checking whether a torrent exists.
	checkError uint64
	// addError represents the number of times there was an error adding a torrent to the database.
	addError uint64
	// mseEncryption represents the number of times a peer connection has been obfuscated with mse.
	mseEncryption uint64
	// extensions represents the number of times a peer connection has been negotiated with a given extension set.
	extensions map[string]uint64

	sync.Mutex
}

// IncBootstrap increments the bootstrap counter in the Stats struct.
func (s *Stats) IncBootstrap() {
	s.Lock()
	defer s.Unlock()
	s.bootstrap++
}

// IncUDPError increments the UDP error count in the Stats struct.
// If the 'write' parameter is true, it increments the writeError count.
// Otherwise, it increments the readError count.
func (s *Stats) IncUDPError(write bool) {
	s.Lock()
	defer s.Unlock()
	if write {
		s.writeError++
	} else {
		s.readError++
	}
}

// IncRtClearing increments the rtClearing field of the Stats struct.
func (s *Stats) IncRtClearing() {
	s.Lock()
	defer s.Unlock()
	s.rtClearing++
}

// IncNonUTF8 increments the nonUTF8 counter in the Stats struct.
func (s *Stats) IncNonUTF8() {
	s.Lock()
	defer s.Unlock()
	s.nonUTF8++
}

// IncDBError increments the error count for the database.
// If add is true, it increments the addError count.
// If add is false, it increments the checkError count.
func (s *Stats) IncDBError(add bool) {
	s.Lock()
	defer s.Unlock()
	if add {
		s.addError++
	} else {
		s.checkError++
	}
}

// IncLeech increments the leech statistics based on the provided parameters.
// If the 'obfuscated' parameter is true, it increments the 'mseEncryption' counter.
// If the 'obfuscated' parameter is false, it increments the 'plaintext' counter.
// It also increments the counter for the given 'peerExtensions'.
func (s *Stats) IncLeech(peerExtensions [8]byte) {
	s.Lock()
	defer s.Unlock()
	s.mseEncryption++
	s.extensions[string(fmt.Sprintf("%v", peerExtensions))]++
}
