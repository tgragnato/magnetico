package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/tgragnato/magnetico/dht"
	"github.com/tgragnato/magnetico/dht/mainline"
	"github.com/tgragnato/magnetico/metadata"
	"github.com/tgragnato/magnetico/persistence"
	"github.com/tgragnato/magnetico/stats"
	"github.com/tgragnato/magnetico/web"
)

var opFlags struct {
	RunDaemon bool
	RunWeb    bool

	DatabaseURL string

	IndexerAddrs        []string
	IndexerMaxNeighbors uint

	LeechMaxN          int
	BootstrappingNodes []string
	FilterNodesCIDRs   []net.IPNet

	Addr string

	Credentials     map[string][]byte
	CredentialsPath string
}

func main() {
	// opFlags is the "operational flags"
	if parseFlags() != nil {
		// Do not print any error messages as jessevdk/go-flags already did.
		return
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
			if err := loadCred(opFlags.CredentialsPath); err != nil {
				log.Fatalf("couldn't load credentials %s\n", err.Error())
			}
		}
	}()

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

	trawlingManager := dht.NewManager(opFlags.IndexerAddrs, opFlags.IndexerMaxNeighbors, opFlags.BootstrappingNodes, opFlags.FilterNodesCIDRs)
	metadataSink := metadata.NewSink(5*time.Second, opFlags.LeechMaxN, opFlags.FilterNodesCIDRs)

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

func parseFlags() error {
	var cmdF struct {
		DatabaseURL string `long:"database" description:"URL of the database." default:"postgres://magnetico:magnetico@localhost:5432/magnetico?sslmode=disable"`

		IndexerAddrs        []string `long:"indexer-addr" description:"Address(es) to be used by indexing DHT nodes." default:"0.0.0.0:0"`
		IndexerMaxNeighbors uint     `long:"indexer-max-neighbors" description:"Maximum number of neighbors of an indexer." default:"5000"`

		LeechMaxN uint `long:"leech-max-n" description:"Maximum number of leeches." default:"1000"`
		MaxRPS    uint `long:"max-rps" description:"Maximum requests per second." default:"500"`

		BootstrappingNodes []string `long:"bootstrap-node" description:"Host(s) to be used for bootstrapping." default:"dht.tgragnato.it"`
		FilterNodesCIDRs   []string `long:"filter-nodes-cidrs" description:"List of CIDRs on which Magnetico can operate. Empty is open mode." default:""`

		Addr string `short:"a" long:"addr"        description:"Address (host:port) to serve on" default:"[::1]:8080"`
		Cred string `short:"c" long:"credentials" description:"Path to the credentials file" default:""`

		RunDaemon bool `short:"d" long:"daemon" description:"Run the crawler without the web interface."`
		RunWeb    bool `short:"w" long:"web"    description:"Run the web interface without the crawler."`
	}

	if _, err := flags.Parse(&cmdF); err != nil {
		return err
	}

	if cmdF.RunDaemon && !cmdF.RunWeb {
		opFlags.RunDaemon = true
		opFlags.RunWeb = false
	} else if !cmdF.RunDaemon && cmdF.RunWeb {
		opFlags.RunDaemon = false
		opFlags.RunWeb = true
	} else {
		opFlags.RunDaemon = true
		opFlags.RunWeb = true
	}

	opFlags.DatabaseURL = cmdF.DatabaseURL

	if opFlags.RunWeb {
		opFlags.Addr = cmdF.Addr
		opFlags.CredentialsPath = cmdF.Cred

		opFlags.Credentials = make(map[string][]byte)
		if err := loadCred(opFlags.CredentialsPath); err == nil {
			web.UpdateCredentials(opFlags.Credentials)
		}
	}

	if opFlags.RunDaemon {
		if err := checkAddrs(cmdF.IndexerAddrs); err != nil {
			log.Fatalf("Of argument (list) `trawler-ml-addr` %s\n", err.Error())
		} else {
			opFlags.IndexerAddrs = cmdF.IndexerAddrs
		}

		opFlags.IndexerMaxNeighbors = cmdF.IndexerMaxNeighbors

		opFlags.LeechMaxN = int(cmdF.LeechMaxN)
		if opFlags.LeechMaxN > 1000 {
			log.Println(
				"Beware that on many systems max # of file descriptors per process is limited to 1024. " +
					"Setting maximum number of leeches greater than 1k might cause \"too many open files\" errors!",
			)
		}

		mainline.DefaultThrottleRate = int(cmdF.MaxRPS)
		opFlags.BootstrappingNodes = cmdF.BootstrappingNodes

		opFlags.FilterNodesCIDRs = []net.IPNet{}
		for _, cidr := range cmdF.FilterNodesCIDRs {
			if cidr == "" {
				continue
			}
			if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
				opFlags.FilterNodesCIDRs = append(opFlags.FilterNodesCIDRs, *ipnet)
			} else {
				log.Fatalf("Error while parsing CIDR %s: %s\n", cidr, err.Error())
			}
		}
		if len(opFlags.FilterNodesCIDRs) != 0 && reflect.DeepEqual(cmdF.BootstrappingNodes, []string{"dht.tgragnato.it"}) {
			log.Fatalln("You should specify your own internal bootstrapping nodes in filter mode.")
		}
	}

	return nil
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

func loadCred(cred string) error {
	if cred == "" {
		return nil
	}

	file, err := os.Open(cred)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	for lineno := 1; true; lineno++ {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.New("Error while reading line " + strconv.Itoa(lineno) + " " + err.Error())
		}

		line = line[:len(line)-1] // strip '\n'

		/* The following regex checks if the line satisfies the following conditions:
		 *
		 * <USERNAME>:<BCRYPT HASH>
		 *
		 * where
		 *     <USERNAME> must start with a small-case a-z character, might contain non-consecutive
		 *   underscores in-between, and consists of small-case a-z characters and digits 0-9.
		 *
		 *     <BCRYPT HASH> is the output of the well-known bcrypt function.
		 */
		re := regexp.MustCompile(`^[a-z](?:_?[a-z0-9])*:\$2[aby]?\$\d{1,2}\$[./A-Za-z0-9]{53}$`)
		if !re.Match(line) {
			return fmt.Errorf("on line %d: format should be: <USERNAME>:<BCRYPT HASH>, instead got: %s", lineno, line)
		}

		tokens := bytes.Split(line, []byte(":"))
		opFlags.Credentials[string(tokens[0])] = tokens[1]
	}

	return nil
}
