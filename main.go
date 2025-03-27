package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tgragnato.it/magnetico/v2/dht"
	"tgragnato.it/magnetico/v2/dht/mainline"
	"tgragnato.it/magnetico/v2/metadata"
	"tgragnato.it/magnetico/v2/opflags"
	"tgragnato.it/magnetico/v2/persistence"
	"tgragnato.it/magnetico/v2/stats"
	"tgragnato.it/magnetico/v2/web"
)

func main() {
	// opFlags is the "operational flags"
	opFlags := opflags.OpFlags{}
	if err := opFlags.Parse(); err != nil {
		log.Fatalln(err.Error())
	}

	// Handle Ctrl-C gracefully.
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	database, err := persistence.MakeDatabase(opFlags.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not open the database %s. %s\n", opFlags.DatabaseURL, err.Error())
	}
	defer func() {
		if err = database.Close(); err != nil {
			log.Printf("Could not close database! %s\n", err.Error())
		}
	}()

	// Export the database to file if requested.
	if opFlags.Export != "" {
		err := persistence.MakeExport(database, opFlags.Export, interruptChan)
		if err != nil {
			log.Fatalf("Could not export the database %s\n", err.Error())
		}
		return
	}

	// Import the database from file if requested.
	if opFlags.Import != "" {
		err := persistence.MakeImport(database, opFlags.Import, interruptChan)
		if err != nil {
			log.Fatalf("Could not import the database %s\n", err.Error())
		}
		return
	}

	// Reload credentials when you receive SIGHUP
	sighupChan := make(chan os.Signal, 1)
	signal.Notify(sighupChan, syscall.SIGHUP)
	go func() {
		for range sighupChan {
			if opFlags.Credentials == nil {
				// Ignoring SIGHUP since no credential file was supplied
				continue
			}

			opFlags.Credentials = make(map[string][]byte)
			if err := opFlags.LoadCred(); err != nil {
				log.Fatalf("couldn't load credentials %s\n", err.Error())
			}
		}
	}()

	if opFlags.PyroscopeURL != "" {
		profiler, err := stats.InitPyroscope(opFlags.PyroscopeURL)
		if err != nil {
			log.Fatalf("could not start pyroscope %s\n", err.Error())
		}
		defer func() {
			if err := profiler.Stop(); err != nil {
				log.Printf("Could not stop pyroscope! %s\n", err.Error())
			}
		}()
	}

	if opFlags.RunWeb {
		go web.StartWeb(opFlags.Addr, opFlags.Timeout, opFlags.Credentials, database)
	}

	if !opFlags.RunDaemon {
		<-interruptChan
		return
	}

	mainline.DefaultThrottleRate = int(opFlags.MaxRPS)
	trawlingManager := dht.NewManager(
		opFlags.IndexerAddrs,
		opFlags.IndexerMaxNeighbors,
		opFlags.BootstrappingNodes,
		opFlags.FilterNodesIpNets,
	)
	metadataSink := metadata.NewSink(
		time.Duration(opFlags.LeechDeadline)*time.Second,
		int(opFlags.LeechMaxN),
		opFlags.FilterNodesIpNets,
	)

	// The Event Loop
	for stopped := false; !stopped; {
		select {
		case result := <-trawlingManager.Output():
			infoHash := result.InfoHash()

			exists, err := database.DoesTorrentExist(infoHash[:])
			if err != nil {
				go stats.GetInstance().IncDBError(false)
			} else if !exists {
				metadataSink.Sink(result)
			}

		case md := <-metadataSink.Drain():
			if err := database.AddNewTorrent(md.InfoHash, md.Name, md.Files); err != nil {
				go stats.GetInstance().IncDBError(true)
			}

		case <-interruptChan:
			trawlingManager.Terminate()
			stopped = true
		}
	}
}
