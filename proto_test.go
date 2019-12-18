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
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func genRandomLog(bi int) BeaconLog {
	var rval BeaconLog
	rval.Datetime = time.Now()
	rval.Rssi = int16(rand.Intn(64) - 72)
	rval.BeaconIndex = uint16(bi)
	return rval
}
func genRandomBeacon() BeaconData {
	var rval BeaconData
	_, _ = rand.Read(rval.Uuid[:])
	rval.Major = uint16(rand.Intn(32))
	rval.Minor = uint16(rand.Intn(32))
	return rval
}

func genRandomPacket(nbeacons, nlog int) *BeaconLogPacket {
	var rval BeaconLogPacket
	rand.Read(rval.Uuid[:])

	for i := 0; i < nbeacons; i++ {
		rval.Beacons = append(rval.Beacons, genRandomBeacon())
	}
	for i := 0; i < nlog; i++ {
		rval.Logs = append(rval.Logs, genRandomLog(rand.Intn(nbeacons)))
	}
	return &rval
}

func TestEncodePacket(t *testing.T) {
	blp := genRandomPacket(1, 1)
	t.Logf("Packet structure %#v\n", *blp)
	binblp, err := blp.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed due to %s", err)
	}
	t.Logf("Packet structure %#v\n", binblp)

	t.Log("Next packet")
	p := BeaconLogPacket{Flags: 0x0, Uuid: Uuid{0x52, 0xfd, 0xfc, 0x7, 0x21,
		0x82, 0x65, 0x4f, 0x16, 0x3f, 0x5f, 0xf, 0x9a, 0x62, 0x1d, 0x72},
		Logs: []BeaconLog{BeaconLog{Rssi: -64, BeaconIndex: 0x0}},
		Beacons: []BeaconData{BeaconData{Uuid: Uuid{0x95, 0x66, 0xc7, 0x4d, 0x10,
			0x3, 0x7c, 0x4d, 0x7b, 0xbb, 0x4, 0x7, 0xd1, 0xe2, 0xc6, 0x49},
			Major: 0x6, Minor: 0x19}}}

	t.Logf("Packet structure %#v\n", p)
	binblp, err = p.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed due to %s", err)
	}
	if !bytes.Equal(binblp, []byte{0x0, 0x52, 0xfd, 0xfc, 0x7, 0x21, 0x82, 0x65,
		0x4f, 0x16, 0x3f, 0x5f, 0xf, 0x9a, 0x62, 0x1d, 0x72, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0,
		0x95, 0x66, 0xc7, 0x4d, 0x10, 0x3, 0x7c, 0x4d, 0x7b, 0xbb, 0x4, 0x7, 0xd1,
		0xe2, 0xc6, 0x49, 0x6, 0x0, 0x19, 0x0, 0xcf, 0x37, 0x28, 0xe4, 0xa6, 0xdb,
		0xe7, 0xff, 0xc0, 0xff, 0x0, 0x0}) {
		t.Fatalf("binblp not same as byte string %s", binblp)
	}

	t.Logf("Packet structure %#v\n", binblp)
}

func TestDecodePacket(t *testing.T) {
	p := []byte{0x0, 0x52, 0xfd, 0xfc, 0x7, 0x21, 0x82, 0x65,
		0x4f, 0x16, 0x3f, 0x5f, 0xf, 0x9a, 0x62, 0x1d, 0x72, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0,
		0x95, 0x66, 0xc7, 0x4d, 0x10, 0x3, 0x7c, 0x4d, 0x7b, 0xbb, 0x4, 0x7, 0xd1,
		0xe2, 0xc6, 0x49, 0x6, 0x0, 0x19, 0x0, 0x0, 0x60, 0xD7, 0x1D, 0x14, 0x00,
		0x00, 0x00, 0xc0, 0xff, 0x0, 0x0}

	var tar BeaconLogPacket
	if err := tar.UnmarshalBinary(p); err != nil {
		t.Fatalf("Failed to parse with %s\n", err)
	}
	t.Log("Time: ", tar.Logs[0].Datetime)
}

func TestEncodeResponse(t *testing.T) {
	var packet BeaconResponsePacket

	packet.Flags = 0x5550
	packet.Data = "1234567890"
	res, err := packet.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	d := []byte{0x50, 0x55, 0xa, 0x0, 0x0, 0x0, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x30}

	if !bytes.Equal(d, res) {
		t.Fatal("Bytes not the same value")
	}
	var decodetest BeaconResponsePacket
	decodetest.UnmarshalBinary(d)
	if decodetest.Flags != packet.Flags {
		t.Fatal("Flags not correct")
	}
	if decodetest.Data != packet.Data {
		t.Fatal("Data not correct")
	}
}
