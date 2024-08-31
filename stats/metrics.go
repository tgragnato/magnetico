package stats

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

var (
	instance *Stats
	once     sync.Once
)

// GetStats returns a singleton instance of Stats
func GetInstance() *Stats {
	once.Do(func() {
		instance = &Stats{extensions: map[string]uint64{}}
		go func() {
			for range time.NewTicker(time.Hour).C {
				instance.printMetrics()
			}
		}()
	})
	return instance
}

func (s *Stats) printMetrics() {
	s.Lock()
	defer s.Unlock()
	message := "\n --- \n"

	if s.bootstrap != 0 {
		message += fmt.Sprintf("dht: the routing table was bootstrapped %d times\n", s.bootstrap)
		s.bootstrap = 0
	}

	if s.rtClearing != 0 {
		message += fmt.Sprintf("dht: the routing table was cleared %d times\n", s.rtClearing)
		s.rtClearing = 0
	}

	if s.writeError != 0 {
		message += fmt.Sprintf("dht: there was an error writing a message to the UDP socket %d times\n", s.writeError)
		s.writeError = 0
	}

	if s.readError != 0 {
		message += fmt.Sprintf("dht: there was an error reading a message from the UDP socket %d times\n", s.readError)
		s.readError = 0
	}

	if s.nonUTF8 != 0 {
		message += fmt.Sprintf("persistence: a torrent was ignored due to its name not being UTF-8 compliant %d times\n", s.nonUTF8)
		s.nonUTF8 = 0
	}

	if s.checkError != 0 {
		message += fmt.Sprintf("persistence: there was an error checking whether a torrent exists %d times\n", s.checkError)
		s.checkError = 0
	}

	if s.addError != 0 {
		message += fmt.Sprintf("persistence: there was an error adding a torrent to the database %d times\n", s.addError)
		s.addError = 0
	}

	if s.mseEncryption != 0 && s.plaintext != 0 {
		message += fmt.Sprintf("metainfo: the peer connection was obfuscated with mse %d%% of time\n", s.mseEncryption/s.plaintext)
		s.mseEncryption = 0
		s.plaintext = 0
	}

	if len(s.extensions) != 0 {
		// Sort extensions by value
		sortedExtensions := make([]string, 0, len(s.extensions))
		for ext := range s.extensions {
			sortedExtensions = append(sortedExtensions, ext)
		}
		sort.Slice(sortedExtensions, func(i, j int) bool {
			return s.extensions[sortedExtensions[i]] > s.extensions[sortedExtensions[j]]
		})

		// Add sorted extensions to the message and clear the map
		for _, ext := range sortedExtensions {
			message += fmt.Sprintf("metainfo: the extension %s was encountered %d times\n", ext, s.extensions[ext])
			delete(s.extensions, ext)
		}
	}

	log.Print(message + "\n --- \n")
}
