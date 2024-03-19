package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tgragnato/magnetico/dht"
	"github.com/tgragnato/magnetico/dht/mainline"
	"github.com/tgragnato/magnetico/metadata"
	"github.com/tgragnato/magnetico/persistence"
)

type opFlags struct {
	DatabaseURL string

	IndexerAddrs        []string
	IndexerInterval     time.Duration
	IndexerMaxNeighbors uint

	LeechMaxN          int
	BootstrappingNodes []string
}

func main() {
	// opFlags is the "operational flags"
	opFlags, err := parseFlags()
	if err != nil {
		// Do not print any error messages as jessevdk/go-flags already did.
		return
	}

	// Handle Ctrl-C gracefully.
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	database, err := persistence.MakeDatabase(opFlags.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not open the database %s. %v", opFlags.DatabaseURL, err)
	}

	trawlingManager := dht.NewManager(opFlags.IndexerAddrs, opFlags.IndexerInterval, opFlags.IndexerMaxNeighbors, opFlags.BootstrappingNodes)
	metadataSink := metadata.NewSink(5*time.Second, opFlags.LeechMaxN)

	// The Event Loop
	for stopped := false; !stopped; {
		select {
		case result := <-trawlingManager.Output():
			infoHash := result.InfoHash()

			exists, err := database.DoesTorrentExist(infoHash[:])
			if err != nil {
				log.Fatalf("Could not check whether torrent exists! %V", err)
			} else if !exists {
				metadataSink.Sink(result)
			}

		case md := <-metadataSink.Drain():
			if err := database.AddNewTorrent(md.InfoHash, md.Name, md.Files); err != nil {
				log.Fatalf("Could not add new torrent to the database. %v", err)
			}

		case <-interruptChan:
			trawlingManager.Terminate()
			stopped = true
		}
	}

	if err = database.Close(); err != nil {
		log.Printf("Could not close database! %v", err)
	}
}

func parseFlags() (*opFlags, error) {
	var cmdF struct {
		DatabaseURL string `long:"database" description:"URL of the database."`

		IndexerAddrs        []string `long:"indexer-addr" description:"Address(es) to be used by indexing DHT nodes." default:"0.0.0.0:0"`
		IndexerInterval     uint     `long:"indexer-interval" description:"Indexing interval in integer seconds." default:"1"`
		IndexerMaxNeighbors uint     `long:"indexer-max-neighbors" description:"Maximum number of neighbors of an indexer." default:"10000"`

		LeechMaxN uint `long:"leech-max-n" description:"Maximum number of leeches." default:"1000"`
		MaxRPS    uint `long:"max-rps" description:"Maximum requests per second." default:"0"`

		BootstrappingNodes []string `long:"bootstrap-node" description:"Host(s) to be used for bootstrapping." default:"dht.tgragnato.it"`
	}

	opF := new(opFlags)

	_, err := flags.Parse(&cmdF)
	if err != nil {
		return nil, err
	}

	if cmdF.DatabaseURL == "" {
		opF.DatabaseURL =
			"postgres://magnetico:magnetico@localhost:5432/magnetico?sslmode=disable"
	} else {
		opF.DatabaseURL = cmdF.DatabaseURL
	}

	if err = checkAddrs(cmdF.IndexerAddrs); err != nil {
		log.Fatalf("Of argument (list) `trawler-ml-addr` %v", err)
	} else {
		opF.IndexerAddrs = cmdF.IndexerAddrs
	}

	opF.IndexerInterval = time.Duration(cmdF.IndexerInterval) * time.Second
	opF.IndexerMaxNeighbors = cmdF.IndexerMaxNeighbors

	opF.LeechMaxN = int(cmdF.LeechMaxN)
	if opF.LeechMaxN > 1000 {
		log.Println(
			"Beware that on many systems max # of file descriptors per process is limited to 1024. " +
				"Setting maximum number of leeches greater than 1k might cause \"too many open files\" errors!",
		)
	}

	mainline.DefaultThrottleRate = int(cmdF.MaxRPS)
	opF.BootstrappingNodes = cmdF.BootstrappingNodes

	return opF, nil
}

func checkAddrs(addrs []string) error {
	for _, addr := range addrs {
		// We are using ResolveUDPAddr but it works equally well for checking TCPAddr(esses) as
		// well.
		_, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}
	}
	return nil
}
