package main

import (
	"encoding/json"
	"flag"
	"github.com/co60ca/beaconpi"
	log "github.com/sirupsen/logrus"
	"os"
)

func getflags() (out beaconpi.MetricsParameters) {
	flag.StringVar(&out.DriverName, "db-driver-name", "",
		"Required: The database driver name")
	flag.StringVar(&out.DataSourceName, "db-datasource-name", "",
		"Required: The database datasource name, may be multiple tokes")
	flag.StringVar(&out.Port, "port", "", "Required: Port for serving http")
	flag.StringVar(&out.AllowedOrigin, "allowed-origin", "http://localhost:3000", "Origin, including http(s) for valid domains that may access the resource, * is invalid for our application.")
	cfgfile := flag.String("config", "", "Required for SMTP use")
	flag.Parse()

	if *cfgfile != "" {
		// Parse cfgfile
		f, err := os.Open(*cfgfile)
		if err != nil {
			log.Panicf("Failed to open file %s with %s", *cfgfile, err)
		}
		dec := json.NewDecoder(f)
		if err = dec.Decode(&out); err != nil {
			log.Panicf("Failed to decode file %s with %s", *cfgfile, err)
		}
	}
	return
}

func main() {
	config := getflags()
	beaconpi.MetricStart(&config)
}
