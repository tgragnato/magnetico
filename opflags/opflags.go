package opflags

import (
	"errors"
	"fmt"
	"net"
	"reflect"
)

type OpFlags struct {
	DatabaseURL  string `long:"database" description:"URL of the database." default:"postgres://magnetico:magnetico@localhost:5432/magnetico?sslmode=disable" yaml:"databaseURL"`
	PyroscopeURL string `long:"pyroscope" description:"URL of the Pyroscope server." default:"" yaml:"pyroscopeURL"`

	IndexerAddrs        []string `long:"indexer-addr" description:"Address(es) to be used by indexing DHT nodes." default:"0.0.0.0:0" yaml:"indexerAddrs"`
	IndexerMaxNeighbors uint     `long:"indexer-max-neighbors" description:"Maximum number of neighbors of an indexer." default:"5000" yaml:"indexerMaxNeighbors"`

	LeechDeadline uint `long:"leech-deadline" description:"Deadline for leeches in seconds." default:"5" yaml:"leechDeadline"`
	LeechMaxN     uint `long:"leech-max-n" description:"Maximum number of leeches." default:"1000" yaml:"leechMaxN"`
	MaxRPS        uint `long:"max-rps" description:"Maximum requests per second." default:"500" yaml:"maxRPS"`

	BootstrappingNodes []string `long:"bootstrap-node" description:"Host(s) to be used for bootstrapping." default:"dht.tgragnato.it:80" default:"dht.tgragnato.it:443" default:"dht.tgragnato.it:1337" default:"dht.tgragnato.it:6969" default:"dht.tgragnato.it:6881" default:"dht.tgragnato.it:25401" yaml:"bootstrappingNodes"`
	FilterNodesCIDRs   []string `long:"filter-nodes-cidrs" description:"List of CIDRs on which Magnetico can operate. Empty is open mode." default:"" yaml:"filterNodesCIDRs"`
	FilterNodesIpNets  []net.IPNet

	Addr        string `short:"a" long:"addr"        description:"Address (host:port) to serve on" default:"[::1]:8080" yaml:"addr"`
	Cred        string `short:"c" long:"credentials" description:"Path to the credentials file" default:"" yaml:"cred"`
	Credentials map[string][]byte

	RunDaemon bool `short:"d" long:"daemon" description:"Run the crawler without the web interface." yaml:"runDaemon"`
	RunWeb    bool `short:"w" long:"web"    description:"Run the web interface without the crawler." yaml:"runWeb"`

	ConfigFilePath string `long:"config-file-path" description:"Configuration YAML file path. If not filled in, it will default to disabled." default:""`
}

func (o *OpFlags) check() error {
	if !o.RunDaemon && !o.RunWeb {
		o.RunDaemon = true
		o.RunWeb = true
	}

	if o.RunWeb {
		o.Credentials = make(map[string][]byte)
		if err := o.LoadCred(); err != nil {
			return err
		}
	}

	if o.RunDaemon {
		if err := o.checkAddrs(); err != nil {
			return err
		}

		if o.LeechMaxN > 1000 {
			fmt.Println(
				"Beware that on many systems max # of file descriptors per process is limited to 1024. " +
					"Setting maximum number of leeches greater than 1k might cause \"too many open files\" errors!",
			)
		}

		o.FilterNodesIpNets = []net.IPNet{}
		for _, cidr := range o.FilterNodesCIDRs {
			if cidr == "" {
				continue
			}
			if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
				o.FilterNodesIpNets = append(o.FilterNodesIpNets, *ipnet)
			} else {
				return fmt.Errorf("error while parsing CIDR %s: %s", cidr, err.Error())
			}
		}
		if len(o.FilterNodesIpNets) != 0 && reflect.DeepEqual(o.BootstrappingNodes, []string{"dht.tgragnato.it:80", "dht.tgragnato.it:443", "dht.tgragnato.it:1337", "dht.tgragnato.it:6969", "dht.tgragnato.it:6881", "dht.tgragnato.it:25401"}) {
			return fmt.Errorf("you should specify your own internal bootstrapping nodes in filter mode")
		}
	}

	return nil
}

func (o *OpFlags) checkAddrs() error {
	if len(o.IndexerAddrs) == 0 || len(o.IndexerAddrs) == 1 && o.IndexerAddrs[0] == "" {
		return errors.New("IndexerAddrs cannot be empty")
	}

	for _, addr := range o.IndexerAddrs {
		// We are using ResolveUDPAddr but it works equally well for checking TCPAddr(esses) as
		// well.
		_, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return err
		}
	}
	return nil
}
