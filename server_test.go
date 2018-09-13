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
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	// Always skip this
	t.Skip()

	logfname := "test.log"
	logf, err := os.Create(logfname)
	if err != nil {
		t.Fatal(err)
	}
	log.SetOutput(logf)
	channelend := make(chan struct{}, 1)
	prog := "/mnt/storage/Source_Code/Projects/beacons"
	config := ServerConfig{X509cert: prog + "/x509/server.crt",
		X509key: prog + "/x509/server.key", Drivername: "postgres",
		DSN: "user=postgres password=notapassword dbname=beacons sslmode=disable"}
	t.Log("Starting server")
	go StartServer(config.X509cert, config.X509key,
		config.Drivername, config.DSN, channelend)
	time.Sleep(1 * time.Second)
	channelend <- struct{}{} // End after one request

	t.Log("Server started")
	packet := BeaconLogPacket{Flags: REQUEST_BEACON_UPDATES}
	encpacket, _ := packet.MarshalBinary()
	resbytes := sendBytes(encpacket)
	t.Logf("%#v\n", resbytes)
	var rp BeaconResponsePacket
	err = rp.UnmarshalBinary(resbytes)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v\n", rp)
	time.Sleep(1 * time.Second)
}

func sendBytes(b []byte) []byte {
	var servcertfile, clientcertfile, clientkeyfile string
	prog := "/mnt/storage/Source_Code/Projects/beacons"
	servcertfile = prog + "/x509/server.crt"
	clientcertfile = prog + "/x509/client.crt"
	clientkeyfile = prog + "/x509/client.key"

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
	port := DEFAULT_PORT
	log.Println("Dialing localhost")
	conn, err := tls.Dial("tcp", "localhost:"+port, conf)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Write(b)
	if err != nil {
		log.Fatal(err)
	}
	conn.CloseWrite()

	data, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Fatal(err)
	}
	return data
}
