// Package cmd is essentially the main() of the secret volume service.
// It is a separate package to allow convenient use of Go build tags to control
// debug logging and system calls.
package cmd

import (
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
	// TODO(negz): Read Talos SRV from configuration
	srv = kingpin.Arg("talos-srv", "SRV record at which to lookup Talos.").String()

	addr   = kingpin.Flag("addr", "Address at which to serve requests (host:port)").Default(":10002").String()
	ns     = kingpin.Flag("ns", "DNS server to use to lookup SRV records (host:port)").String()
	parent = kingpin.Flag("parent", "Directory under which to mount secret volumes").Short('p').Default("/secrets").ExistingDir()
	virt   = kingpin.Flag("virtual", "Use an in-memory filesystem and a no-op parenter for testing").Bool()
	stop   = kingpin.Flag("close-after", "Wait this long at shutdown before closing HTTP connections.").Default("1m").Duration()
	kill   = kingpin.Flag("kill-after", "Wait this long at shutdown before exiting.").Default("2m").Duration()
)

func setupLb(ns, srv string) lb.LoadBalancer {
	var lib dns.Lookup
	if ns == "" {
		// TODO(negz): Handle error if/when https://github.com/benschw/srv-lb/pull/5 is merged
		lib = dns.NewDefaultLookupLib()
	} else {
		lib = dns.NewLookupLib(ns)
	}

	return lb.New(&lb.Config{Dns: lib, Strategy: random.RandomStrategy}, srv)
}

// Run is effectively the main() of the secretvolume binary.
// It lives here in its own package to allow convenient use of Go build tags
// to control debug logging and system calls.
func Run() {
	kingpin.Parse()

	sp, err := secrets.NewTalosProducer(setupLb(*ns, *srv))
	kingpin.FatalIfError(err, "cannot setup Talos secret producer")

	m, fs, err := setupFs(*virt, *parent)
	kingpin.FatalIfError(err, "cannot setup filesystem and parenter")

	sps := map[api.SecretSource]secrets.Producer{api.Talos: sp}
	vm, err := volume.NewManager(m, sps, volume.Filesystem(fs))
	kingpin.FatalIfError(err, "cannot setup secret volume manager")

	handlers, err := server.NewHTTPHandlers(vm)
	kingpin.FatalIfError(err, "cannot setup HTTP handlers")

	hd := &httpdown.HTTP{StopTimeout: *stop, KillTimeout: *kill}
	http := handlers.HTTPServer(*addr)
	kingpin.FatalIfError(httpdown.ListenAndServe(http, hd), "HTTP server error")
}
