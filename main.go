package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tgragnato.it/magnetico/dht"
	"tgragnato.it/magnetico/dht/mainline"
	"tgragnato.it/magnetico/metadata"
	"tgragnato.it/magnetico/opflags"
	"tgragnato.it/magnetico/persistence"
	"tgragnato.it/magnetico/stats"
	"tgragnato.it/magnetico/web"
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
		defer profiler.Stop()
	}

	database, err := persistence.MakeDatabase(opFlags.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not open the database %s. %s\n", opFlags.DatabaseURL, err.Error())
	}
	defer func() {
		if err = database.Close(); err != nil {
			log.Printf("Could not close database! %s\n", err.Error())
		}
	}()

	if opFlags.RunWeb {
		go web.StartWeb(opFlags.Addr, opFlags.Credentials, database)
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
