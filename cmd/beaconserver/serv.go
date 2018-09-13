// Beacon Pi, a edge node system for iBeacons and Edge nodes made of Pi
// Copyright (C) 2017  Maeve Kennedy
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//
package main

import (
	"github.com/co60ca/beaconpi"
	"log"
)

func main() {
	config := beaconpi.GetFlags()
	log.Printf("Config: %#v", config)
	beaconpi.StartServer(config.X509cert, config.X509key,
			config.Drivername, config.DSN, nil)
}
