package stats

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Stats saves the metrics for each particular operation.
type Stats struct {
	// bootstrap represents the number of times the DHT has been bootstrapped.
	bootstrap prometheus.Counter
	// writeError represents the number of times there was an error writing a message to the UDP socket.
	writeError prometheus.Counter
	// readError represents the number of times there was an error reading a message from the UDP socket.
	readError prometheus.Counter
	// rtClearing represents the number of times the routing table has been cleared.
	rtClearing prometheus.Counter
	// nonUTF8 represents the number of times a torrent has been ignored due to its name not being UTF-8 compliant.
	nonUTF8 prometheus.Counter
	// checkError represents the number of times there was an error checking whether a torrent exists.
	checkError prometheus.Counter
	// addError represents the number of times there was an error adding a torrent to the database.
	addError prometheus.Counter
	// mseEncryption represents the number of times a peer connection has been obfuscated with mse.
	mseEncryption prometheus.Counter
	// extensions represents the number of times a peer connection has been negotiated with a given extension set.
	extensions map[string]prometheus.Counter

	sync.Mutex
}

func (s *Stats) Collect(ch chan<- prometheus.Metric) {
	s.bootstrap.Collect(ch)
	s.writeError.Collect(ch)
	s.readError.Collect(ch)
	s.rtClearing.Collect(ch)
	s.nonUTF8.Collect(ch)
	s.checkError.Collect(ch)
	s.addError.Collect(ch)
	s.mseEncryption.Collect(ch)

	s.Lock()
	defer s.Unlock()
	for _, counter := range s.extensions {
		counter.Collect(ch)
	}
}

func (s *Stats) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(s, descs)
}

// IncBootstrap increments the bootstrap counter in the Stats struct.
func (s *Stats) IncBootstrap() {
	s.bootstrap.Inc()
}

// IncUDPError increments the UDP error count in the Stats struct.
// If the 'write' parameter is true, it increments the writeError count.
// Otherwise, it increments the readError count.
func (s *Stats) IncUDPError(write bool) {
	if write {
		s.writeError.Inc()
	} else {
		s.readError.Inc()
	}
}

// IncRtClearing increments the rtClearing field of the Stats struct.
func (s *Stats) IncRtClearing() {
	s.rtClearing.Inc()
}

// IncNonUTF8 increments the nonUTF8 counter in the Stats struct.
func (s *Stats) IncNonUTF8() {
	s.nonUTF8.Inc()
}

// IncDBError increments the error count for the database.
// If add is true, it increments the addError count.
// If add is false, it increments the checkError count.
func (s *Stats) IncDBError(add bool) {
	if add {
		s.addError.Inc()
	} else {
		s.checkError.Inc()
	}
}

// IncLeech increments the leech statistics based on the provided 'peerExtensions'.
func (s *Stats) IncLeech(peerExtensions [8]byte) {
	s.mseEncryption.Inc()

	s.Lock()
	defer s.Unlock()

	extensionSet := string(fmt.Sprintf("%v", peerExtensions))
	extensionSet = regexp.MustCompile(`[^a-zA-Z0-9_:]`).ReplaceAllString(extensionSet, "_")

	if _, ok := s.extensions[extensionSet]; !ok {
		s.extensions[extensionSet] = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "magnetico",
			Name:      "extension" + extensionSet,
			Help:      "Number of times a peer connection has been negotiated with a given extension set",
		})
	}

	s.extensions[extensionSet].Inc()
}
