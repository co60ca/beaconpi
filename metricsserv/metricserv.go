package main

import (
	"github.com/co60ca/beaconpi"
        "flag"
)

func getflags() (out beaconpi.MetricsParameters) {
	flag.StringVar(&out.DriverName, "db-driver-name", "",
		"Required: The database driver name")
	flag.StringVar(&out.DataSourceName, "db-datasource-name", "",
		"Required: The database datasource name, may be multiple tokes")
        flag.StringVar(&out.Port, "port", "", "Required: Port for serving http")
        flag.StringVar(&out.AllowedOrigin, "allowed-origin", "http://localhost:3000", "Origin, including http(s) for valid domains that may access the resource, * is invalid for our application.")
        flag.Parse()
        return
}

func main() {
        config := getflags()
	beaconpi.MetricStart(&config)
}
