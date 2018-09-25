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
	// Milliseconds
	TIMEOUT_BEACON_REFRESH = 60000
	TIMEOUT_BEACON         = 50
	BACKOFF_MAX            = 30 * time.Second
	BACKOFF_MIN            = 50 * time.Millisecond
	BACKOFF_MULTIPLIER     = 2
)

// Encapsulates all client data
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

// Main entry point for the client app that is run on the edge devices
func StartClient() {

	var (
		servcertfile         string
		clientcertfile       string
		clientkeyfile        string
		servhost             string
		servport             string
		clientuuid           string
		timeoutBeaconRefresh int
		timeoutBeacon        int
		logDebug             bool
	)

	flag.StringVar(&servcertfile, "serv-cert-file", "", "Has trusted keys")
	flag.StringVar(&clientcertfile, "client-cert-file", "", "")
	flag.StringVar(&clientkeyfile, "client-key-file", "", "")
	flag.StringVar(&clientuuid, "client-uuid", "", "Uuid for this node, no dashes")
	flag.StringVar(&servhost, "serv-host", "localhost", "")
	flag.StringVar(&servport, "serv-port", DEFAULT_PORT, "")
	flag.IntVar(&timeoutBeaconRefresh, "timeout-beacon-refresh", TIMEOUT_BEACON_REFRESH, "timeout for beacon data rerequest from server to keep freshness")
	flag.IntVar(&timeoutBeacon, "timeout-beacon", TIMEOUT_BEACON, "timeout for beacon sightings before pushing to the server")
	flag.BoolVar(&logDebug, "debug", false, "enable more logging")
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
		timeoutBeaconRefresh: time.Millisecond * time.Duration(timeoutBeaconRefresh),
		timeoutBeacon:        time.Millisecond * time.Duration(timeoutBeacon),
	}

	uuiddec, err := hex.DecodeString(clientuuid)
	if err != nil {
		log.Fatal("uuid is not valid hex, do not include -")
	}
	copy(client.uuid[:], uuiddec)
	if logDebug {
		log.SetLevel(log.DebugLevel)
	}

	clientLoop(&client)
}

// clientLoop is the hot loop for the beacons it handles communicating with
// the remote server and creating packets to send over the channel
func clientLoop(client *clientinfo) {
	timeruuid := time.NewTicker(client.timeoutBeaconRefresh)
	timerbeacon := time.NewTicker(client.timeoutBeacon)
	brs := make(chan BeaconRecord, 256)
	go processIBeacons(client, brs)
	first := true

	var conn *tls.Conn

	datapacket := new(BeaconLogPacket)
	copy(datapacket.Uuid[:], client.uuid[:])
	datapacket.Flags = CURRENT_VERSION

	// Map from uuid,major,minor to offset
	currentbeacons := make(map[string]int)

	var backoff time.Duration = BACKOFF_MIN
	log.Println("Start loop")
	for {
		var err error
		for conn == nil {
			log.Infof("Creating new connection: host: %s", client.host)
			conn, err = tls.Dial("tcp", client.host, client.tlsconf)
			if err != nil {
				log.Info("Back off ", backoff)
				time.Sleep(backoff)
				backoff *= BACKOFF_MULTIPLIER
				if backoff > BACKOFF_MAX {
					backoff = BACKOFF_MAX
				}
				log.Printf("Failed to open socket, abandoning: %s", err)
				continue
			}
			backoff = BACKOFF_MIN
			vbuff := bytes.NewBuffer([]byte{byte(CURRENT_VERSION)})
			_, err = io.CopyN(conn, vbuff, 1)
			if err != nil {
				log.Printf("Failed to write current version to remote %s", err)
				conn.Close()
				conn = nil
				return
			}
			vbuff.Reset()
			_, err = io.CopyN(vbuff, conn, 1)
			if err != nil {
				log.Print("Failed to get server version")
				conn.Close()
				conn = nil
				return
			}
			if uint8(vbuff.Bytes()[0]) != CURRENT_VERSION {
				log.Println("Server failed to answer with same version")
				conn.Close()
				conn = nil
				return
			}
		}

		if first {
			log.Println("Init request beacons")
			first = false
			if err = requestBeacons(client, conn); err != nil {
				log.Printf("Error occured, connection killed %s", err)
				conn = nil
				continue
			}
		}

		select {
		case tempbr := <-brs:
			// Block gets Beacons from beacon log producer
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
		case _ = <-timeruuid.C:
			if err = requestBeacons(client, conn); err != nil {
				log.Printf("Error occured, connection killed %s", err)
				conn = nil
			}
		case _ = <-timerbeacon.C:
			if len(datapacket.Logs) == 0 {
				continue
			}
			// Send and reset
			if err = sendData(client, conn, datapacket); err != nil {
				log.Printf("Error occured, connection killed %s", err)
				conn = nil
			}
			// Reset data
			currentbeacons = make(map[string]int)
			datapacket = new(BeaconLogPacket)
			datapacket.Flags = CURRENT_VERSION
			copy(datapacket.Uuid[:], client.uuid[:])

		}
		if len(datapacket.Beacons) == MAX_LOGS {
			log.Println("Sending data to server due to full queue")
			if err = sendData(client, conn, datapacket); err != nil {
				log.Printf("Error occured, connection killed %s", err)
				conn = nil
			}
			// Reset data
			currentbeacons = make(map[string]int)
			datapacket = new(BeaconLogPacket)
			copy(datapacket.Uuid[:], client.uuid[:])
			datapacket.Flags = CURRENT_VERSION
		}
	}

}

// handleFatalError returns and error (if available, err == nil just makes
// a new error with msg) it will also close the tls.Conn
func handleFatalError(conn *tls.Conn, msg string, err error) error {
	conn.Close()
	if err != nil {
		return errors.Wrap(err, msg)
	}
	return errors.New(msg)
}

// writes to the conn the length of the buff before sending, returning
// and error and closing the channel
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

// sendData sends the current datapacket over the connection handling errors
// and responses
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

	err = readFromRemoteOrClose(conn, buff)
	if err != nil {
		return errors.Wrap(err, "Failed to read response to sendData")
	}

	return readUpdates(client, conn, buff)
}

// requestBeacons sends a request for the registered beacons from the server
func requestBeacons(client *clientinfo, conn *tls.Conn) error {
	var blp BeaconLogPacket
	blp.Flags = CURRENT_VERSION
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

	err = readFromRemoteOrClose(conn, reader)
	if err != nil {
		return handleFatalError(conn, "Failed to read response to sendData", err)
	}

	return readUpdates(client, conn, reader)
}

// For any handling of client responses
func readUpdates(client *clientinfo, conn *tls.Conn, buff *bytes.Buffer) error {
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

// handleSystem processes any response from the server that
// contains RESPONSE_SYSTEM flag
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
