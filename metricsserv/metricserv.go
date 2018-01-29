package main

import (
	"github.com/co60ca/beaconpi"
)

func main() {
	config := beaconpi.MetricsParameters{Port: "32967",
		DriverName: "postgres",
		DataSourceName: "user=bkennedy dbname=beacons sslmode=disable"}
	beaconpi.MetricStart(&config)
}
