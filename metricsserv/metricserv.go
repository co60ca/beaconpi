package main

import (
	"github.com/co60ca/beaconpi"
)

func main() {
	config := beaconpi.MetricsParameters{Port: "32967"}
	beaconpi.MetricStart(&config)
}
