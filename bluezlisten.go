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

package beaconpi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"time"
)

func ProduceBLEAdv(bleadv chan *bytes.Buffer) {
	log.Println("Starting hcitool for BLE")
	hcitool := exec.Command("hcitool", "lescan", "--duplicates")
	if err := hcitool.Start(); err != nil {
		log.Printf("Error starting hcitool: %s", err)
		return
	}
	log.Println("Starting hcidump")
	hcidump := exec.Command("hcidump", "--raw")
	read, err := hcidump.StdoutPipe()
	if err := hcidump.Start(); err != nil {
		log.Printf("Error starting hcidump: %s", err)
		return
	}
	log.Println("hcidump started")
	if err != nil {
		log.Printf("Error connecting to stdout: %s", err)
	}
	scan := bufio.NewScanner(read)
	scan.Split(bufio.ScanWords)

	var buffer *bytes.Buffer = new(bytes.Buffer)

	// Skip header
	for scan.Scan() {
		if scan.Text() == ">" {
			break
		}
	}
	// Main section
	for scan.Scan() {
		token := scan.Text()
		if token == ">" && buffer.Len() != 0 {
			bleadv <- buffer
			buffer = new(bytes.Buffer)
			continue
		}
		decodedb, err := hex.DecodeString(token)
		if err != nil {
			log.Println("This shouldn't happen")
			continue
		}
		// Will panic if fails
		_, _ = buffer.Write(decodedb)
	}
}

type BeaconRecord struct {
	BeaconData
	Datetime time.Time
	Rssi     int16
}

func ProcessIBeacons(client *clientinfo, brs chan BeaconRecord) {
	bleadv := make(chan *bytes.Buffer, 128)
	go ProduceBLEAdv(bleadv)

	for {
		bytesb := <-bleadv
		buffer := bytesb.Bytes()
		index := bytes.Index(buffer, []byte{0x4C, 0x00, 0x02})
		if index == -1 {
			continue
		}
		buffer = buffer[index+4:] // There is one byte we wanna skip
		if len(buffer) < 22 {
			log.Println("Buffer was not long enough for", buffer)
			continue
		}
		var beaconRecord BeaconRecord
		copy(beaconRecord.Uuid[:], buffer[:16])
		buffer = buffer[16:]
		reader := bytes.NewReader(buffer)
		if err := binary.Read(reader, binary.BigEndian, &beaconRecord.Major); err != nil {
			log.Printf("Failed to parse major: %s", err)
			continue
		}
		if err := binary.Read(reader, binary.BigEndian, &beaconRecord.Minor); err != nil {
			log.Printf("Failed to parse Minor: %s", err)
			continue
		}

		{
			client.Lock()
			uid := fmt.Sprintf("%s,%d,%d", beaconRecord.Uuid.String(), beaconRecord.Major, beaconRecord.Minor)
			if _, ok := client.nodes[uid]; !ok {
				client.Unlock()
				continue
			}
			client.Unlock()
		}

		var rssi int8
		// NOTE: we throw away the 21st bit, which is the send power
		binary.Read(reader, binary.BigEndian, &rssi);
		// Read the real RSSI
		if err := binary.Read(reader, binary.BigEndian, &rssi); err != nil {
			log.Printf("Failed to parse Rssi: %s", err)
			continue
		}
		beaconRecord.Rssi = int16(rssi)
		beaconRecord.Datetime = time.Now()
		brs <- beaconRecord
	}
}
