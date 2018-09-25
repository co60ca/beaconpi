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

	if err = binary.Write(buff, binary.LittleEndian, uint32(len(respbytes))); err != nil {
		log.Printf("Failed to encode len of response")
		return
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	_, err = buff.WriteTo(conn)
	if err != nil {
		log.Printf("Failed to write len of response %+v", errors.WithStack(err))
		return
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
	n, err := conn.Write(respbytes)
	if n != len(respbytes) || err != nil {
		log.Printf("Failed to write response. Len written: %d of %d"+
			". Error was %+v", n, len(respbytes), err)
	}
	conn.SetWriteDeadline(time.Time{})
}

// handleConnection handles the connection once established
func handleConnection(conn net.Conn, end chan struct{}) {
	var resp BeaconResponsePacket
	version := uint8(CURRENT_VERSION)
	log.Infof("New connection from %s", conn.RemoteAddr())

	raiseErr := func(flags uint16, err error) {
		log.Printf("handleConnection failed with %s", err)
		resp.Flags |= flags
		writeResponseAndClose(conn, &resp, true, version)
	}

	// Copy the first byte
	buff, err := readBytesOrCancel(conn, 1, &resp, CURRENT_VERSION, end)
	if err != nil {
		raiseErr(RESPONSE_INVALID, err)
		return
	}
	version = uint8(buff.Bytes()[0] & VERSION_MASK)
	resp.Flags |= uint16(version)

	// Write the version back
	if err = writeBytesOrCancel(conn, bytes.NewBuffer([]byte{uint8(version)}),
		&resp, version, end); err != nil {
		raiseErr(RESPONSE_INVALID,
			errors.Wrap(err, "Failed to write back version"))
		return

	}
	// else use streaming
	for {
		buff, err := readBytesOrCancel(conn, 4, &resp, version, end)
		if err != nil {
			raiseErr(RESPONSE_INVALID, errors.Wrap(err, "Failed to read length"))
			return
		}

		var length uint32
		if err = binary.Read(buff, binary.LittleEndian, &length); err != nil {
			err = errors.Wrap(err, "Unexpected error when reading from buffer")
			raiseErr(RESPONSE_INVALID, err)
			return
		}

		buff, err = readBytesOrCancel(conn, int64(length), &resp, version, end)
		if err != nil {
			err = errors.Wrap(err, "Recieved error while reading packet %s")
			raiseErr(RESPONSE_INVALID, err)
			return
		}

		var message BeaconLogPacket
		err = message.UnmarshalBinary(buff.Bytes())
		if err != nil {
			err = errors.Wrap(err, "Recieved error while unmarshalling %s")
			raiseErr(RESPONSE_INVALID, err)
			return

		}
		// We do not handle packets in parallel because we need to send
		// back data to the connection in order of arrival
		handlePacket(conn, &resp, &message)
	}
}

// handlePacket operates on a single packet inserting data
// and sending back status and commands
func handlePacket(conn net.Conn, resp *BeaconResponsePacket,
	pack *BeaconLogPacket) {
	version := pack.Flags & VERSION_MASK
	errorClose := true
	// Version 0 should close on success, Version > 0 uses stream connections
	successClose := false

	responseHandle := func(flags uint16, err error) {
		resp.Flags |= flags
		if err != nil {
			log.Println("handlePacket failed with: %s", err)
			writeResponseAndClose(conn, resp, errorClose, version)
			return
		}
		writeResponseAndClose(conn, resp, successClose, version)
	}

	db, err := db.openDB()
	if err != nil {
		responseHandle(RESPONSE_INTERNAL_FAILURE, errors.Wrap(err, "Failed to open DB"))
		return
	}
	defer db.Close()

	// Client request beacon updates
	if pack.Flags&REQUEST_BEACON_UPDATES != 0 {
		log.Info("Client requested beacon updates")
		beacons, err := dbGetBeacons(db)
		if err != nil {
			responseHandle(RESPONSE_INTERNAL_FAILURE, errors.Wrap(err, "Failed to get beacons"))
			return
		}
		for _, b := range beacons {
			resp.Data += "\n" + b.String()
		}
		if len(resp.Data) > 1 {
			resp.Data = resp.Data[1:]
		}
		responseHandle(RESPONSE_BEACON_UPDATES, nil)
		return
	}

	// Client requested command and control
	if pack.Flags&REQUEST_CONTROL_LOG != 0 {
		edgeid, err := dbCheckUuid(pack.Uuid, db)
		if err != nil {
			err = errors.Wrapf(err, "Error occured edgeid \"%s\" was not found in db", pack.Uuid)
			responseHandle(RESPONSE_INTERNAL_FAILURE, err)
			return
		}
		if err = dbInsertControlLog(edgeid, pack, db); err != nil {
			responseHandle(RESPONSE_INTERNAL_FAILURE, err)
		} else {
			responseHandle(RESPONSE_OK, nil)
		}
		return

		// Client is phoning home to give the results of the command and control
	} else if pack.Flags&REQUEST_CONTROL_COMPLETE != 0 {
		err = dbCompleteControl(pack, db)
		if err != nil {
			responseHandle(RESPONSE_INTERNAL_FAILURE, errors.Wrap(err, "Failed to update control"))
			return
		}
		responseHandle(RESPONSE_OK, nil)
	} else {
		control, err := dbGetControl(pack, db)
		if err != nil {
			// log.Printf("DEBUG: Failed to get control, passing: %s", err)
		} else {
			log.Infof("Sending control %s", control)
			resp.Data = control
			resp.Flags |= RESPONSE_SYSTEM
		}
	}

	var edgeid int

	edgeid, err = dbCheckUuid(pack.Uuid, db)
	if err != nil {
		err = errors.Wrapf(err, "Error occured edgeid \"%s\" was not found in db", pack.Uuid)
		responseHandle(RESPONSE_INVALID, err)
		return
	}

	// Update the time of the given edge that we have confirmed
	updateEdgeLastUpdate(pack.Uuid, db)
	log.Debug("Packet from ", pack.Uuid, edgeid)
	if err = dbAddLogsForBeacons(pack, edgeid, db); err != nil {
		err = errors.Wrap(err, "Error when checking in logs for beacon")
		responseHandle(RESPONSE_INTERNAL_FAILURE, err)
		return
	}
	responseHandle(RESPONSE_OK, nil)
}

// writeBytesOrCancel
func writeBytesOrCancel(conn net.Conn, buff *bytes.Buffer, resp *BeaconResponsePacket, version uint8, end chan struct{}) error {
	var timeoutcount int
	n := int64(buff.Len())

	raiseErr := func(flags uint16, err error) error {
		resp.Flags |= flags
		writeResponseAndClose(conn, resp, true, version)
		return err
	}

	for n > 0 {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		copyn, err := io.CopyN(conn, buff, n)

		if err == nil {
			break
		}
		switch v := err.(type) {
		case net.Error:
			// pass
			if !v.Timeout() {
				return raiseErr(RESPONSE_INVALID, err)
			}
			timeoutcount += 1
			if timeoutcount > 5 {
				err = errors.New("Failed to write to conn in 10 seconds")
				return raiseErr(RESPONSE_INVALID, err)
			}

			// If timeout check if the channel is closed, if so return
			select {
			case _, _ = <-end:
				err = errors.New("Shutdown requested")
				return raiseErr(RESPONSE_INTERNAL_FAILURE, err)
			default:
			}
			log.Println("DEBUG: timeout")
		default:
			return raiseErr(RESPONSE_INVALID, err)
		}
		n -= copyn
	}
	conn.SetWriteDeadline(time.Time{})
	return nil
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

	raiseErr := func(flags uint16, err error) (*bytes.Buffer, error) {
		resp.Flags |= flags
		writeResponseAndClose(conn, resp, true, version)
		return nil, err
	}

	for n > 0 {
		conn.SetReadDeadline(time.Now().Add(time.Second * 2))
		copyn, err := io.CopyN(buff, conn, n)

		if err == nil {
			break
		}
		switch v := err.(type) {
		case net.Error:
			// pass
			if !v.Timeout() {
				return raiseErr(RESPONSE_INVALID, err)
			}
			timeoutcount += 1
			if timeoutcount > 5 {
				err = errors.New("Failed to read from conn in 10 seconds")
				return raiseErr(RESPONSE_INVALID, err)
			}

			// If timeout check if the channel is closed, if so return
			select {
			case _, _ = <-end:
				err = errors.New("Shutdown requested")
				return raiseErr(RESPONSE_INTERNAL_FAILURE, err)
			default:
			}
			log.Println("DEBUG: timeout")
		default:
			return raiseErr(RESPONSE_INVALID, err)
		}
		n -= copyn
	}
	conn.SetReadDeadline(time.Time{})
	return buff, nil
}
