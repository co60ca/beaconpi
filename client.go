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
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TIMEOUT_BEACON_REFRESH = 60
	TIMEOUT_BEACON         = 2
)

type clientinfo struct {
	sync.Mutex

	tlsconf *tls.Config
	// Key to nothing, key is "UUID,major,minor"
	nodes map[string]struct{}
	host  string
	uuid  Uuid
	// Time to re request beacons from server
	timeoutBeaconRefresh time.Duration
	// Time to force the beacons sightings to the server
	timeoutBeacon time.Duration
}

func StartClient() {

	//log.SetFlags(log.Lshortfile)
	var (
		servcertfile         string
		clientcertfile       string
		clientkeyfile        string
		servhost             string
		servport             string
		clientuuid           string
		timeoutBeaconRefresh int
		timeoutBeacon        int
	)

	flag.StringVar(&servcertfile, "serv-cert-file", "", "Has trusted keys")
	flag.StringVar(&clientcertfile, "client-cert-file", "", "")
	flag.StringVar(&clientkeyfile, "client-key-file", "", "")
	flag.StringVar(&clientuuid, "client-uuid", "", "Uuid for this node, no dashes")
	flag.StringVar(&servhost, "serv-host", "localhost", "")
	flag.StringVar(&servport, "serv-port", DEFAULT_PORT, "")
	flag.IntVar(&timeoutBeaconRefresh, "timeout-beacon-refresh", TIMEOUT_BEACON_REFRESH, "timeout for beacon data rerequest from server to keep freshness")
	flag.IntVar(&timeoutBeacon, "timeout-beacon", TIMEOUT_BEACON, "timeout for beacon sightings before pushing to the server")
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

	client := clientinfo{
		tlsconf:              conf,
		host:                 servhost + ":" + servport,
		nodes:                make(map[string]struct{}),
		timeoutBeaconRefresh: time.Second * time.Duration(timeoutBeaconRefresh),
		timeoutBeacon:        time.Second * time.Duration(timeoutBeacon),
	}

	uuiddec, err := hex.DecodeString(clientuuid)
	if err != nil {
		log.Fatal("uuid is not valid hex, do not include -")
	}
	copy(client.uuid[:], uuiddec)
	log.Printf("Client arguments %#v", client)
	clientLoop(&client)
}

func clientLoop(client *clientinfo) {
	timeruuid := time.NewTicker(client.timeoutBeaconRefresh)
	timerbeacon := time.NewTicker(client.timeoutBeacon)
	brs := make(chan BeaconRecord, 256)
	go ProcessIBeacons(client, brs)
	log.Println("Init request beacons")

	var conn *tls.Conn

	requestBeacons(client, conn)

	datapacket := new(BeaconLogPacket)
	copy(datapacket.Uuid[:], client.uuid[:])
	datapacket.Flags = CURRENT_VERSION

	// Map from uuid,major,minor to offset
	currentbeacons := make(map[string]int)

	log.Println("Start loop")
	for {
		var err error
		for conn == nil {
			conn, err = tls.Dial("tcp", client.host, client.tlsconf)
			if err != nil {
				// TODO backoff
				log.Printf("Failed to open socket, abandoning: %s", err)
				return
			}
		}

		select {
		case _ = <-timeruuid.C:
			requestBeacons(client, conn)

		case _ = <-timerbeacon.C:
			log.Println("Sending data to server due to timeout")
			// Send and reset
			sendData(client, conn, datapacket)
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
				i = len(datapacket.Beacons) - 1
				currentbeacons[beaconstr] = i
			}
			datapacket.Logs = append(datapacket.Logs, BeaconLog{
				Datetime:    tempbr.Datetime,
				Rssi:        tempbr.Rssi,
				BeaconIndex: uint16(i)})
		}
		if len(datapacket.Beacons) == MAX_LOGS {
			log.Println("Sending data to server due to full queue")
			sendData(client, conn, datapacket)
			// Reset data
			currentbeacons = make(map[string]int)
			datapacket = new(BeaconLogPacket)
			copy(datapacket.Uuid[:], client.uuid[:])
			datapacket.Flags = CURRENT_VERSION
		}
	}

}

func handleFatalError(conn *tls.Conn, msg string, err error) error {
	conn.Close()
	if err != nil {
		return errors.Wrap(err, msg)
	}
	return errors.New(msg)
}

func writeLengthLE32(conn *tls.Conn, buff *bytes.Buffer) error {
	err := binary.Write(conn, binary.LittleEndian, uint32(buff.Len()))
	if err != nil {
		conn.Close()
		return err
	}
	return nil
}

// Will reset your buffer
func readFromRemoteOrClose(conn *tls.Conn, buff *bytes.Buffer) error {
	_, err := io.CopyN(buff, conn, 4)
	if err != nil {
		conn.Close()
		return errors.Wrap(err, "Failed to read length")
	}
	var length uint32
	err = binary.Read(buff, binary.LittleEndian, &length)
	if err != nil {
		conn.Close()
		return errors.Wrap(err, "Failed to decode length")
	}
	buff.Reset()
	_, err = io.CopyN(buff, conn, int64(length))
	if err != nil {
		conn.Close()
		return errors.Wrap(err, "Failed to read data packet")
	}
	return nil
}

func sendData(client *clientinfo, conn *tls.Conn, datapacket *BeaconLogPacket) error {
	bytespacket, err := datapacket.MarshalBinary()
	if err != nil {
		return handleFatalError(conn, "Failed to marshal binary", err)
	}
	buff := bytes.NewBuffer(bytespacket)
	err = writeLengthLE32(conn, buff)
	if err != nil {
		return err
	}

	_, err = buff.WriteTo(conn)
	if err != nil {
		return handleFatalError(conn, "Failed to write to socket", err)
	}
	//conn.CloseWrite()
	buff.Reset()
	//readFromRemoteOrClose(conn *tls.Conn, buff *bytes.Buffer) error {
	err = readFromRemoteOrClose(conn, buff)
	if err != nil {
		return errors.Wrap(err, "Failed to read response to sendData")
	}

	return readUpdates(client, conn, buff)
}

func requestBeacons(client *clientinfo, conn *tls.Conn) error {
	var blp BeaconLogPacket
	blp.Flags |= REQUEST_BEACON_UPDATES
	copy(blp.Uuid[:], client.uuid[:])
	buffer, err := blp.MarshalBinary()
	if err != nil {
		return handleFatalError(conn, "Failed to marshal request message", err)
	}

	writer := bytes.NewBuffer(buffer)
	err = writeLengthLE32(conn, writer)
	if err != nil {
		return err
	}

	_, err = writer.WriteTo(conn)
	if err != nil {
		return handleFatalError(conn, "Failed to write to connection abandoning", err)
	}
	//conn.CloseWrite()
	reader := writer
	reader.Reset()

	if _, err = reader.ReadFrom(conn); err != nil {
		return handleFatalError(conn, "Failed to read from connection, abandoning", err)
	}
	return readUpdates(client, conn, reader)
}

// For any handling of client responses
func readUpdates(client *clientinfo, conn *tls.Conn, buff *bytes.Buffer) error {
	log.Println("Recieved response from server")
	var brp BeaconResponsePacket
	if err := brp.UnmarshalBinary(buff.Bytes()); err != nil {
		return handleFatalError(conn, "Failed to Unmarshal response packet", err)
	}
	if brp.Flags&RESPONSE_BEACON_UPDATES != 0 {
		splitnl := strings.Split(brp.Data, "\n")
		client.Lock()
		defer client.Unlock()
		client.nodes = make(map[string]struct{})
		for _, line := range splitnl {
			client.nodes[line] = struct{}{}
		}
		log.Printf("New beacon list: \n%#v", client.nodes)
		log.Println("Completed parsing response from server")
	} else if brp.Flags&RESPONSE_SYSTEM != 0 {
		return handleSystem(client, conn, &brp)
	}
	return nil
}

func handleSystem(client *clientinfo, conn *tls.Conn, brp *BeaconResponsePacket) error {
	cd := strings.SplitN(brp.Data, "\n", 2)
	if len(cd) != 2 {
		return handleFatalError(conn, "Sent control is invalid", nil)
	}
	_, err := strconv.Atoi(cd[0])
	if err != nil {
		return handleFatalError(conn, "Sent control is invalid not integer", nil)
	}
	command := cd[1]
	var cmd []string
	if err = json.Unmarshal([]byte(command), &cmd); err != nil {
		return handleFatalError(conn, "Sent control: Failed to unmarshal: ", err)
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
	return sendData(client, conn, &datapacket)
}
