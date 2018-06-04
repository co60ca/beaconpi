package main

import (
	"github.com/co60ca/beaconpi"
	"os"
)

func main() {
	defaultfile, err := os.Open("mockconf.json")
	if err != nil {
		panic(err)
	}
	defer defaultfile.Close()
	conf, err := beaconpi.ReadMockConfiguration(defaultfile)
	if err != nil {
		panic(err)
	}
	conf.HCIDump()
}
