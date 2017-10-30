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
	"os"
	"crypto/tls"
	"log"
	"net"
	"flag"
	"io/ioutil"
	//"database/sql"
)

var db *dbHandler

type ServerConfig struct {
	X509cert string
	X509key string
	Drivername string
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
	flag.Parse()
	if out.X509cert == "" || out.X509key == "" ||
			out.Drivername == "" || out.DSN == "" {
		flag.Usage()
		os.Exit(1)
	}
	return
}

func StartServer(x509cert, x509key, drivername, dsn string, end chan struct{}) {
	log.SetFlags(log.Lshortfile)

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

	for {
		select {
			case <- end:
				log.Println("Recieved end message, stopping...")
				return
			default:
		}
		conn, err := ln.Accept()

		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func writeResponseAndClose(conn net.Conn, resp *BeaconResponsePacket, close bool) {
		respbytes, err := resp.MarshalBinary()
		if err != nil {
			log.Println("Failed to marshal data to response:", err)
		}
		n, err := conn.Write(respbytes)
		if n != len(respbytes) || err != nil {
			log.Printf("Failed to write response. Len written: %d of %d" +
				". Error was %s", err)
		}
		if close {
			conn.Close()
		}
}

func handleConnection(conn net.Conn) {
	// TODO(mae)  We use ioutil.ReadAll here but in the future we should use
	// bytes.Buffer.ReadFrom with handling to prevent a buffer over the max size
	buff, err := ioutil.ReadAll(conn)
	log.Println("Read from connection")
	var resp BeaconResponsePacket
	resp.Flags = CURRENT_VERSION
	if err != nil {
		log.Println("Message failed to read with:" , err)
		resp.Flags |= RESPONSE_INVALID
		// TODO different flags based on the errors
		writeResponseAndClose(conn, &resp, true)
		return
	}

	var message BeaconLogPacket
	err = message.UnmarshalBinary(buff)
	if err != nil {
		log.Println("Failed to parse message with:", err)
		resp.Flags |= RESPONSE_INVALID
		writeResponseAndClose(conn, &resp, true)
		return
	}


	// Client request beacon updates
	if message.Flags & REQUEST_BEACON_UPDATES != 0 {
		db, err := db.openDB()
		if err != nil {
			log.Println("Failed to open DB", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		defer db.Close()
		beacons, err := dbGetBeacons(db)
		if err != nil {
			log.Println("Failed to get Beacons", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		for _, b := range beacons {
			resp.Data += "\n" + b.String()
		}
		if len(resp.Data) > 1 {
			resp.Data = resp.Data[1:]
		}

		resp.Flags |= RESPONSE_BEACON_UPDATES
		writeResponseAndClose(conn, &resp, true)
		return
	}
	if message.Flags & REQUEST_CONTROL_LOG != 0 {
		db, err := db.openDB()
		if err != nil {
			log.Println("Failed to open DB", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		defer db.Close()
		edgeid, err := dbCheckUuid(message.Uuid, db)
		if err != nil {
			log.Printf("Error occured edgeid \"%s\" was not found in db: %s", message.Uuid, err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		if err = dbInsertControlLog(edgeid, &message, db); err != nil {
			log.Printf("Error occured %s", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
		} else {
			resp.Flags |= RESPONSE_OK
		}
		writeResponseAndClose(conn, &resp, true)
		return
	} else if message.Flags & REQUEST_CONTROL_COMPLETE != 0 {
		db, err := db.openDB()
		if err != nil {
			log.Println("Failed to open DB", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		defer db.Close()
		err = dbCompleteControl(&message, db)
		if err != nil {
			log.Println("Failed to update control")
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		resp.Flags |= RESPONSE_OK
		writeResponseAndClose(conn, &resp, true)
	} else {
		db, err := db.openDB()
		if err != nil {
			log.Println("Failed to open DB", err)
			resp.Flags |= RESPONSE_INTERNAL_FAILURE
			writeResponseAndClose(conn, &resp, true)
			return
		}
		defer db.Close()
		control, err := dbGetControl(&message, db)
		if err != nil {
			log.Printf("DEBUG: Failed to get control, passing: %s", err)
		} else {
			resp.Data = control
			resp.Flags |= RESPONSE_SYSTEM
		}
		resp.Flags |= RESPONSE_OK
		writeResponseAndClose(conn, &resp, true)
	}

	handlePacket(&message)
}

func handlePacket(pack *BeaconLogPacket) {
	log.Println("Handling packet")
	var edgeid int
	db, err := db.openDB()
	if err != nil {
		log.Println("Failed to open DB", err)
		return
	}
	defer db.Close()

	edgeid, err = dbCheckUuid(pack.Uuid, db)
	if err != nil {
		log.Printf("Error occured edgeid \"%s\" was not found in db: %s", pack.Uuid, err)
		return
	}
	log.Println("Packet from ", pack.Uuid, edgeid)
	if err = dbAddLogsForBeacons(pack, edgeid, db); err != nil {
		log.Println("Error when checking in logs for beacon", err)
		return
	}

}
