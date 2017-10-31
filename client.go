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
	"crypto/tls"
	"flag"
	"log"
	"sync"
	"time"
	"bytes"
	"strings"
	"os/exec"
	"strconv"
	"encoding/json"
	"encoding/hex"
)

const (
	TIMEOUT_BEACON_REFRESH time.Duration = 1*time.Minute
	TIMEOUT_BEACON time.Duration = 10*time.Second
)

type clientinfo struct {
	sync.Mutex

	tlsconf *tls.Config
	// Key to nothing, key is "UUID,major,minor"
	nodes map[string]struct{}
	host string
	uuid Uuid
}

func StartClient() {

	log.SetFlags(log.Lshortfile)
	var servcertfile, clientcertfile, clientkeyfile, servhost string
	var servport string
	var clientuuid string

	flag.StringVar(&servcertfile, "serv-cert-file", "", "Has trusted keys")
	flag.StringVar(&clientcertfile, "client-cert-file", "", "")
	flag.StringVar(&clientkeyfile, "client-key-file", "", "")
	flag.StringVar(&clientuuid, "client-uuid", "", "Uuid for this node, no dashes")
	flag.StringVar(&servhost, "serv-host", "localhost", "")
	flag.StringVar(&servport, "serv-port", DEFAULT_PORT, "")
	flag.Parse()

	certpool := LoadFileToCert(servcertfile)
	if certpool == nil {
		log.Fatal("Something happened while loading the certfile")
	}

	clientcert, err := tls.LoadX509KeyPair(clientcertfile, clientkeyfile)
	if err != nil {
		log.Fatal("Failed to open x509 keypair", err)
	}

	conf := &tls.Config{
		RootCAs:      certpool,
		Certificates: []tls.Certificate{clientcert},
	}

	client := clientinfo{}
	client.nodes = make(map[string]struct{})
	client.tlsconf = conf
	client.host = servhost+":"+servport
	uuiddec, err := hex.DecodeString(clientuuid)
	if err != nil {
		log.Fatal("uuid is not valid hex, do not include -")
	}
	copy(client.uuid[:], uuiddec)
	clientLoop(&client)
}

func clientLoop(client *clientinfo) {
	timeruuid := time.NewTicker(TIMEOUT_BEACON_REFRESH)
	timerbeacon := time.NewTicker(TIMEOUT_BEACON)
	brs := make(chan BeaconRecord, 256)
	go ProcessIBeacons(client, brs)
	requestBeacons(client)

	datapacket := new(BeaconLogPacket)
	copy(datapacket.Uuid[:], client.uuid[:])
	datapacket.Flags = CURRENT_VERSION

	// Map from uuid,major,minor to offset
	currentbeacons := make(map[string]int)

	for {
		select {
			case _ = <-timeruuid.C:
				go requestBeacons(client)

			case _ = <-timerbeacon.C:
				log.Println("Sending data to server due to timeout")
				// Send and reset
				go sendData(client, datapacket)
				// Reset data
				currentbeacons = make(map[string]int)
				datapacket = new(BeaconLogPacket)
				datapacket.Flags = CURRENT_VERSION
				copy(datapacket.Uuid[:], client.uuid[:])

			case tempbr := <-brs:
				beaconstr := tempbr.BeaconData.String()
				var i int
				var ok bool
				if i, ok = currentbeacons[beaconstr]; !ok {
					datapacket.Beacons = append(datapacket.Beacons, tempbr.BeaconData)
					i = len(datapacket.Beacons)-1
					currentbeacons[beaconstr] = i
				}
				datapacket.Logs = append(datapacket.Logs, BeaconLog{
					Datetime: tempbr.Datetime,
					Rssi: tempbr.Rssi,
					BeaconIndex: uint16(i)})
		}
		if len(datapacket.Beacons) == MAX_LOGS {
			log.Println("Sending data to server due to full queue")
			go sendData(client, datapacket)
			// Reset data
			currentbeacons = make(map[string]int)
			datapacket = new(BeaconLogPacket)
			copy(datapacket.Uuid[:], client.uuid[:])
			datapacket.Flags = CURRENT_VERSION
		}
	}

}

func sendData(client *clientinfo, datapacket *BeaconLogPacket) {
	conn, err := tls.Dial("tcp", client.host, client.tlsconf)
	if err != nil {
		log.Printf("Failed to open socket, abandoning: %s", err)
		return
	}
	defer conn.Close()
	bytespacket, err := datapacket.MarshalBinary()
	if err != nil {
		log.Printf("Failed to marshal binary: %s", err)
		return
	}
	buff := bytes.NewBuffer(bytespacket)
	_, err = buff.WriteTo(conn)
	if err != nil {
		log.Printf("Failed to write to socket: %s", err)
		return
	}
	conn.CloseWrite()
	buff.Reset()
	_, err = buff.ReadFrom(conn)
	if err != nil {
		log.Printf("Failed to read response from server: %s", err)
		return
	}

	readUpdates(client, buff)
}

func requestBeacons(client *clientinfo) {
	var blp BeaconLogPacket
	blp.Flags |= REQUEST_BEACON_UPDATES
	copy(blp.Uuid[:], client.uuid[:])
	buffer, err := blp.MarshalBinary()
	if err != nil {
		log.Fatal("Failed to marshal request message", err)
		return
	}

	conn, err := tls.Dial("tcp", client.host, client.tlsconf)
	if err != nil {
		log.Printf("Failed to request beacons, abandoning: %s", err)
		return
	}
	defer conn.Close()
	writer := bytes.NewBuffer(buffer)
	_, err = writer.WriteTo(conn)
	if err != nil {
		log.Printf("Failed to write to connection abandoning: %s", err)
		return
	}
	conn.CloseWrite()
	reader := writer
	reader.Reset()

	if _, err = reader.ReadFrom(conn); err != nil {
		log.Printf("Failed to read from connection, abandoning: %s", err)
		return
	}
	readUpdates(client, reader)
}

// For any handling of client responses
func readUpdates(client *clientinfo, buff *bytes.Buffer) {
	log.Println("Recieved response from server")
	var brp BeaconResponsePacket
	if err := brp.UnmarshalBinary(buff.Bytes()); err != nil {
		log.Printf("Failed to Unmarshal response packet: %s", err)
		return
	}
	if brp.Flags & RESPONSE_BEACON_UPDATES != 0 {
		splitnl := strings.Split(brp.Data, "\n")
		client.Lock()
		defer client.Unlock()
		client.nodes = make(map[string]struct{})
		for _, line := range splitnl {
			client.nodes[line] = struct{}{}
		}
		log.Printf("New beacon list: \n%#v", client.nodes)
		log.Println("Completed parsing response from server")
	} else if brp.Flags & RESPONSE_SYSTEM != 0 {
		handleSystem(client, &brp)
	}
}

func handleSystem(client *clientinfo, brp *BeaconResponsePacket) {
	cd := strings.SplitN(brp.Data, "\n", 2)
	if len(cd) != 2 {
		log.Println("Sent control is invalid", cd, brp.Data)
		return
	}
	_, err := strconv.Atoi(cd[0])
	if err != nil {
		log.Println("Sent control is invalid not integer:", cd[0])
		return
	}
	command := cd[1]
	var cmd []string
	if err = json.Unmarshal([]byte(command), &cmd); err != nil {
		log.Println("Sent control: Failed to unmarshal: ", err)
		return
	}
	var com *exec.Cmd
	if len(cmd) == 1 {
		com = exec.Command(cmd[0])
	} else if len(cmd) > 1 {
		com = exec.Command(cmd[0], cmd[1:]...)
	}
	output, err := com.CombinedOutput()
	log.Println("Combined output:", output)
	outputstr := cd[0] + "\n" + string(output)
	end := len(outputstr)
	if end > MAX_CTRL {
		end = MAX_CTRL
	}
	outputstr = outputstr[:end]
	var datapacket BeaconLogPacket
	datapacket.Flags = CURRENT_VERSION
	datapacket.Flags |= REQUEST_CONTROL_COMPLETE
	copy(datapacket.Uuid[:], client.uuid[:])
	datapacket.ControlData = outputstr
	sendData(client, &datapacket)
}
