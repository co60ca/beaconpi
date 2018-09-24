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
	"crypto/tls"
	"flag"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	//"database/sql"
	"bytes"
	"encoding/binary"
	"time"
)

var db *dbHandler

// Required BeaconServer config
type ServerConfig struct {
	X509cert string
	X509key  string
	// Database drivername, must be psql
	Drivername string
	// Database data source name
	DSN string
}

func GetFlags() (out ServerConfig) {

	flag.StringVar(&out.X509cert, "serv-cert", "",
		"Required: x509 server public certificate file path")
	flag.StringVar(&out.X509key, "serv-key", "",
		"Required: x509 server private key file path")
	flag.StringVar(&out.Drivername, "db-driver-name", "",
		"Required: The database driver name")
	flag.StringVar(&out.DSN, "db-datasource-name", "",
		"Required: The database datasource name, may be multiple tokes")
	debug := flag.Bool("debug", false, "extra logging")
	flag.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	if out.X509cert == "" || out.X509key == "" ||
		out.Drivername == "" || out.DSN == "" {
		flag.Usage()
		os.Exit(1)
	}
	return
}

// StartServer is the main interface for the BeaconServer
func StartServer(x509cert, x509key, drivername, dsn string, end chan struct{}) {
	// Logging
	log.SetLevel(log.DebugLevel)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	db = new(dbHandler)
	db.Drivername = drivername
	db.DataSourceName = dsn

	cerpoolrootca := LoadFileToCert(x509cert)

	cer, err := tls.LoadX509KeyPair(x509cert, x509key)
	if err != nil {
		log.Fatal(err)
	}
	port := DEFAULT_PORT
	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
		ClientCAs:    cerpoolrootca,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	ln, err := tls.Listen("tcp", ":"+port, config)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Now listening on tcp \":\"" + port)
	defer ln.Close()
	go func() {
		// Passes when closed
		_, _ = <-end
		log.Println("Recieved end message, stopping...")
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()

		if err != nil {
			log.Println(err)
			continue
		}
		// TODO manage cancelation
		go handleConnection(conn, end)
	}
}

// writeResponseAndClose writes a reponse using the transport protocol
// which passes version - length delimited messages in version >0
// and just a message with a close in version 0
//
// Therefore the stream is in version >0
// 1 byte of version, 4 bytes of length, then the remainded is
// a BeaconResponsePacket followed by length then packet ect..
func writeResponseAndClose(conn net.Conn, resp *BeaconResponsePacket, close bool, version uint8) {
	respbytes, err := resp.MarshalBinary()
	if err != nil {
		log.Println("Failed to marshal data to response:", err)
	}
	buff := new(bytes.Buffer)
	defer func() {
		if close {
			log.Debug("Connection closed")
			conn.Close()
		}
	}()

	if version != 0 {
		// In version >0 we only print the version once per connection
		// other than in the flags
		//		_, _ = buff.Write([]byte{uint8(version)})
		err = binary.Write(buff, binary.LittleEndian, uint32(len(respbytes)))
		buff.WriteTo(conn)
		if err != nil {
			log.Printf("Failed to write len of response %s", err)
			return
		}
	}
	n, err := conn.Write(respbytes)
	if n != len(respbytes) || err != nil {
		log.Printf("Failed to write response. Len written: %d of %d"+
			". Error was %s", n, len(respbytes), err)
	}
}

// handleConnection handles the connection once established
func handleConnection(conn net.Conn, end chan struct{}) {
	databuff := new(bytes.Buffer)
	// First we need to find out if the packet is version 0 or >0
	// version 0 starts out with flags, the first byte therefore should
	// have 0000 in the second half of the first byte in contrast, version >0
	// has the version as the first byte (lower half) followed by length

	// Copy the first byte
	_, err := io.CopyN(databuff, conn, 1)
	if err != nil {
		var resp BeaconResponsePacket
		log.Println("Message failed to read with:", err)
		resp.Flags |= RESPONSE_INVALID
		// TODO different flags based on the errors
		writeResponseAndClose(conn, &resp, true /*temp*/, 0)
		return
	}
	version := uint8(databuff.Bytes()[0] & VERSION_MASK)

	if version == uint8(0) {
		var resp BeaconResponsePacket
		resp.Flags |= uint16(0)
		var message BeaconLogPacket
		// Version 0 reads all
		if _, err = databuff.ReadFrom(conn); err != nil {
			log.Println("Failed to read from connection:", err)
			resp.Flags |= RESPONSE_INVALID
			writeResponseAndClose(conn, &resp, true, 0)
			return
		}

		err = message.UnmarshalBinary(databuff.Bytes())
		if err != nil {
			log.Println("Failed to parse message with:", err)
			resp.Flags |= RESPONSE_INVALID
			writeResponseAndClose(conn, &resp, true, 0)
			return
		}
		// We respond with a version the same as the sender

		handlePacket(conn, &resp, &message)
		return
	}
	// Write the version back
	_, err = conn.Write([]byte{uint8(version)})
	if err != nil {
		log.Printf("Failed to write version", err)
	}
	// else use streaming
	for {
		var resp BeaconResponsePacket
		resp.Flags |= uint16(version)
		buff, err := readBytesOrCancel(conn, 4, &resp, version, end)
		if err != nil {
			log.Printf("Recieved error while reading length %s", err)
			return
		}

		var length uint32
		if err = binary.Read(buff, binary.LittleEndian, &length); err != nil {
			log.Println("Unexpected error when reading from buffer ", err)
			resp.Flags |= RESPONSE_INVALID
			writeResponseAndClose(conn, &resp, true, version)
			return
		}

		buff, err = readBytesOrCancel(conn, int64(length), &resp, version, end)
		if err != nil {
			log.Printf("Recieved error while reading packet %s", err)
			resp.Flags |= RESPONSE_INVALID
			writeResponseAndClose(conn, &resp, true, version)
			return
		}

		var message BeaconLogPacket
		err = message.UnmarshalBinary(buff.Bytes())
		if err != nil {
			log.Printf("Recieved error while unmarshalling %s", err)
			resp.Flags |= RESPONSE_INVALID
			writeResponseAndClose(conn, &resp, true, version)
			return

		}
		// We do not handle packets in parallel because we need to send
		// back data to the connection in order of arrival
		handlePacket(conn, &resp, &message)
	}
}

// readBytesOrCancel will read the specified bytes from connection and return
// an error if there was a problem, it handles timeouts to allow you to cancel
// the connection by closing the end channel
func readBytesOrCancel(conn net.Conn, n int64,
	resp *BeaconResponsePacket, version uint8,
	end chan struct{}) (*bytes.Buffer, error) {

	buff := new(bytes.Buffer)
	// Set a deadline for 5 seconds from now
	var timeoutcount int
	for n > 0 {
		conn.SetReadDeadline(time.Now().Add(time.Second * 2))
		copyn, err := io.CopyN(buff, conn, n)
		if err != nil {
			switch v := err.(type) {
			case net.Error:
				// pass
				if v.Timeout() {
					timeoutcount += 1
					if timeoutcount > 10 {
						resp.Flags |= RESPONSE_INVALID
						writeResponseAndClose(conn, resp, true, version)
						return nil, errors.New("Failed to read from conn in 20 seconds")
					}
					// If timeout check if the channel is closed, if so
					// return
					select {
					case _, _ = <-end:
						log.Println("Shutdown requested")
						resp.Flags |= RESPONSE_INTERNAL_FAILURE
						writeResponseAndClose(conn, resp, true, version)
						return nil, errors.New("Shutdown requested")
					default:
					}
					log.Println("DEBUG: timeout")
					n -= copyn
					continue
				}
				log.Println("Unexpected error when reading from buffer ", v)
				resp.Flags |= RESPONSE_INVALID
				writeResponseAndClose(conn, resp, true, version)
				return nil, err
			default:
				log.Println("Unexpected error when reading from buffer ", v)
				resp.Flags |= RESPONSE_INVALID
				writeResponseAndClose(conn, resp, true, version)
				return nil, err
			}
		}
		n -= copyn
	}
	return buff, nil
}

// handlePacket operates on a single packet inserting data
// and sending back status and commands
func handlePacket(conn net.Conn, resp *BeaconResponsePacket,
	pack *BeaconLogPacket) {
	version := pack.Flags & VERSION_MASK
	errorClose := true
	// Version 0 should close on success, Version > 0 uses stream connections
	successClose := version == 0 || false

	db, err := db.openDB()
	if err != nil {
		log.Println("Failed to open DB", err)
		resp.Flags |= RESPONSE_INTERNAL_FAILURE
		writeResponseAndClose(conn, resp, errorClose, version)
		return
	}
	defer db.Close()

	// Client request beacon updates
	if pack.Flags&REQUEST_BEACON_UPDATES != 0 {
		beacons, err := dbGetBeacons(db)
		if err != nil {
			log.Println("Failed to get Beacons", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, resp, errorClose, version)
			return
		}
		for _, b := range beacons {
			resp.Data += "\n" + b.String()
		}
		if len(resp.Data) > 1 {
			resp.Data = resp.Data[1:]
		}

		resp.Flags |= RESPONSE_BEACON_UPDATES
		writeResponseAndClose(conn, resp, successClose, version)
		return
	}

	// Client requested command and control
	if pack.Flags&REQUEST_CONTROL_LOG != 0 {
		edgeid, err := dbCheckUuid(pack.Uuid, db)
		if err != nil {
			log.Printf("Error occured edgeid \"%s\" was not found in db: %s", pack.Uuid, err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, resp, errorClose, version)
			return
		}
		if err = dbInsertControlLog(edgeid, pack, db); err != nil {
			log.Printf("Error occured %s", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, resp, errorClose, version)
		} else {
			resp.Flags |= RESPONSE_OK
			writeResponseAndClose(conn, resp, successClose, version)
		}
		return

		// Client is phoning home to give the results of the command and control
	} else if pack.Flags&REQUEST_CONTROL_COMPLETE != 0 {
		err = dbCompleteControl(pack, db)
		if err != nil {
			log.Println("Failed to update control", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, resp, errorClose, version)
			return
		}
		resp.Flags |= RESPONSE_OK
		writeResponseAndClose(conn, resp, successClose, version)
	} else {
		control, err := dbGetControl(pack, db)
		if err != nil {
			// log.Printf("DEBUG: Failed to get control, passing: %s", err)
		} else {
			log.Debugf("Sending control %s", control)
			resp.Data = control
			resp.Flags |= RESPONSE_SYSTEM
		}
		resp.Flags |= RESPONSE_OK
		writeResponseAndClose(conn, resp, successClose, version)
	}

	var edgeid int

	edgeid, err = dbCheckUuid(pack.Uuid, db)
	if err != nil {
		log.Printf("Error occured edgeid \"%s\" was not found in db: %s", pack.Uuid, err)
		return
	}
	// Update the time of the given edge that we have confirmed
	updateEdgeLastUpdate(pack.Uuid, db)
	log.Debug("Packet from ", pack.Uuid, edgeid)
	if err = dbAddLogsForBeacons(pack, edgeid, db); err != nil {
		log.Println("Error when checking in logs for beacon", err)
		resp.Flags |= RESPONSE_INTERNAL_FAILURE
		writeResponseAndClose(conn, resp, errorClose, version)
		return
	}
	resp.Flags |= RESPONSE_OK
	writeResponseAndClose(conn, resp, successClose, version)

}
