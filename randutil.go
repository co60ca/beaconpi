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

package beaconpi

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
)

// initRand must be called to get a random number generator with a seed from
// the system
func initRand() *rand.Rand {
	randbyte := make([]byte, 4)
	_, err := crand.Read(randbyte)
	if err != nil {
		panic("Getting random seed from system failed: " + err.Error())
	}
	seed, _ := binary.Varint(randbyte)
	rng := rand.New(rand.NewSource(seed))
	return rng
}

// getRand wraps initRand as an interface for getting a *rand.Rand
func getRand() *rand.Rand {
	rng := initRand()
	return rng
}

// RandBase64 returns a random Base64 string with a size of multiples of 3
// bytes when decoded
func randBase64(rng *rand.Rand, sets int) string {
	b := make([]byte, sets*3)
	rng.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
