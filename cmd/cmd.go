package cmd

import (
	"fmt"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/secrets"
	"github.com/negz/secret-volume/server"
	"github.com/negz/secret-volume/volume"

	"github.com/benschw/srv-lb/dns"
	"github.com/benschw/srv-lb/lb"
	"github.com/benschw/srv-lb/strategy/random"
	"github.com/facebookgo/httpdown"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	pod   = kingpin.Arg("pod", "Pod in which to lookup Talos").Required().String()
	addr  = kingpin.Flag("addr", "Address at which to serve requests (host:port)").Default(":10002").String()
	srv   = kingpin.Flag("srv", "SRV record at which to lookup Poppy. Overrides pod").String()
	ns    = kingpin.Flag("ns", "DNS server to use to lookup Poppy SRV records (host:port)").String()
	mount = kingpin.Flag("mountpoint", "Where to mount secret volumes").Short('m').Default("/secrets").ExistingDir()
	virt  = kingpin.Flag("virtual", "Use an in-memory filesystem and a no-op mounter for testing").Bool()
	stop  = kingpin.Flag("close-after", "Wait this long at shutdown before closing HTTP connections.").Default("1m").Duration()
	kill  = kingpin.Flag("kill-after", "Wait this long at shutdown before exiting.").Default("2m").Duration()
)

const srvFormat = "_spotify-poppy._tcp.services.%s.spotify.net"

func setupLb(ns, pod, srv string) lb.LoadBalancer {
	var lib dns.Lookup
	if ns == "" {
		// TODO(negz): Handle error if/when https://github.com/benschw/srv-lb/pull/5 is merged
		lib = dns.NewDefaultLookupLib()
	} else {
		lib = dns.NewLookupLib(ns)
	}

	var record string
	if srv == "" {
		record = fmt.Sprintf(srvFormat, pod)
	} else {
		record = srv
	}

	return lb.New(&lb.Config{Dns: lib, Strategy: random.RandomStrategy}, record)
}

func Run() {
	kingpin.Parse()

	// Setup TSP
	sp, err := secrets.NewTalosSecretProducer(setupLb(*ns, *pod, *srv))
	kingpin.FatalIfError(err, "unable to setup Talos secret producer")

	// Setup volume manager
	m, fs := setupFs(*virt, *mount)
	sps := map[api.SecretSource]secrets.SecretProducer{api.Talos: sp}
	vm, err := volume.NewSecretManager(m, sps, volume.Filesystem(fs))
	kingpin.FatalIfError(err, "unable to setup secret volume manager")

	// Setup HTTP handlers
	handlers, err := server.NewHTTPHandlers(vm)
	kingpin.FatalIfError(err, "unable to setup HTTP handlers")

	// Serve!
	hd := &httpdown.HTTP{StopTimeout: *stop, KillTimeout: *kill}
	http := handlers.HTTPServer(*addr)
	kingpin.FatalIfError(httpdown.ListenAndServe(http, hd), "HTTP server error")
}

/*
TODO(negz):
- Docstrings
- expvar, github.com/pkg/errors, etc
- Vendor and dockerize
- Generate Puppet client certs for each volume (container) automatically?
*/
