package main

import (
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sagostin/tbgo/sbc"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var (
	// flags
	showVersion   = flag.Bool("version", false, "Print version information")
	listenAddress = flag.String("web.listen-address", ":9000", "Address to listen on for web interface and telemetry")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path to expose metrics of the exporter")
	tbApiUrl      = flag.String("tb.url", "https://localhost:12358", "TelcoBridges API URL")
	tbUsername    = flag.String("tb.username", "", "TelcoBridges username")
	tbPassword    = flag.String("tb.password", "", "TelcoBridges api password")
	tbConfig      = flag.String("tb.config", "", "TelcoBridges config")
	tbIdentifier  = flag.String("tb.identifier", "", "TelcoBridges identifier")
)

func init() {
	prometheus.MustRegister(version.NewCollector("tb_exporter"))
}

func main() {
	flag.Parse()

	// check if all required flags are set
	if *tbIdentifier == "" {
		fmt.Fprintln(os.Stderr, "Please provide a identifier for TelcoBridges API")
		os.Exit(1)
	}

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("tb-exporter"))
		os.Exit(0)
	}

	if *tbApiUrl == "" {
		fmt.Fprintln(os.Stderr, "Please provide a address for TelcoBridges API")
		os.Exit(1)
	}

	if *tbUsername == "" {
		fmt.Fprintln(os.Stderr, "Please provide a username for TelcoBridges API")
		os.Exit(1)
	}

	if *tbPassword == "" {
		fmt.Fprintln(os.Stderr, "Please provide a password for TelcoBridges API")
		os.Exit(1)
	}

	log.Infoln("Starting Service API exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	config := sbc.NewClientConfig()
	config.APIHost = *tbApiUrl
	config.APIUsername = *tbUsername
	config.APIPassword = *tbPassword

	client := sbc.NewClient(config)

	tbId := *tbIdentifier

	// create new exporter
	e, err := NewExporter(client, tbId, tbConfig)
	if err != nil {
		fmt.Println("Error initializing Service API exporter.")
		os.Exit(1)
	}

	// register exporter in prometheus
	prometheus.MustRegister(e)

	// serve metrics endpoint & redirect / to metrics endpoint
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Infof("Listening on %s", *listenAddress)

	// listen to requests
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		log.Fatal(err)
	}
}
