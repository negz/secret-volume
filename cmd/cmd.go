// Package cmd is essentially the main() of the secret volume service.
// It is a separate package to allow convenient use of Go build tags to control
// debug logging and system calls.
package cmd

import (
	"os"
	"path/filepath"

	"github.com/negz/secret-volume/api"
	"github.com/negz/secret-volume/secrets"
	"github.com/negz/secret-volume/server"
	"github.com/negz/secret-volume/volume"
	"github.com/uber-go/zap"

	"github.com/benschw/srv-lb/dns"
	"github.com/benschw/srv-lb/lb"
	"github.com/benschw/srv-lb/strategy/random"
	"github.com/facebookgo/httpdown"

	"gopkg.in/alecthomas/kingpin.v2"
)

func setupTalosLb(ns, srv string) lb.LoadBalancer {
	var lib dns.Lookup
	if ns == "" {
		// TODO(negz): Handle error if/when https://github.com/benschw/srv-lb/pull/5 is merged
		lib = dns.NewDefaultLookupLib()
	} else {
		lib = dns.NewLookupLib(ns)
		log.Debug("Using nameserver", zap.String("ns", ns))
	}
	log.Debug("Using Talos SRV", zap.String("srv", srv))

	return lb.New(&lb.Config{Dns: lib, Strategy: random.RandomStrategy}, srv)
}

// Run is effectively the main() of the secretvolume binary.
// It lives here in its own package to allow convenient use of Go build tags
// to control debug logging and system calls.
func Run() {
	var (
		app = kingpin.New(filepath.Base(os.Args[0]), "Manages sets of files containing secrets.").DefaultEnvars()

		talos  = app.Flag("talos-srv", "Enables Talos by providing an SRV record at which to find it.").String()
		addr   = app.Flag("addr", "Address at which to serve requests (host:port).").Default(":10002").String()
		ns     = app.Flag("ns", "DNS server to use to lookup SRV records (host:port).").String()
		parent = app.Flag("parent", "Directory under which to mount secret volumes.").Default("/secrets").String()
		virt   = app.Flag("virtual", "Use an in-memory filesystem and a no-op parenter for testing.").Bool()
		stop   = app.Flag("close-after", "Wait this long at shutdown before closing HTTP connections.").Default("1m").Duration()
		kill   = app.Flag("kill-after", "Wait this long at shutdown before exiting.").Default("2m").Duration()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	m, fs, err := setupFs(*virt, *parent)
	kingpin.FatalIfError(err, "cannot setup filesystem and parenter")

	sps := make(map[api.SecretSource]secrets.Producer)
	if *talos != "" {
		sp, terr := secrets.NewTalosProducer(setupTalosLb(*ns, *talos))
		kingpin.FatalIfError(terr, "cannot setup Talos secret producer")
		sps[api.Talos] = sp
	}

	vm, err := volume.NewManager(m, sps, volume.Filesystem(fs))
	kingpin.FatalIfError(err, "cannot setup secret volume manager")

	handlers, err := server.NewHTTPHandlers(vm)
	kingpin.FatalIfError(err, "cannot setup HTTP handlers")

	hd := &httpdown.HTTP{StopTimeout: *stop, KillTimeout: *kill}
	http := handlers.HTTPServer(*addr)
	kingpin.FatalIfError(httpdown.ListenAndServe(http, hd), "HTTP server error")
}
