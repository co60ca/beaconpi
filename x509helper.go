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
	"crypto/x509"
	"io/ioutil"
	"log"
)

func LoadFileToCert(file string) *x509.CertPool {
	certs := x509.NewCertPool()
	cert, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("File: %s Error: %s", file, err)
		return nil
	}
	if !certs.AppendCertsFromPEM(cert) {
		return nil
	}
	return certs
}
