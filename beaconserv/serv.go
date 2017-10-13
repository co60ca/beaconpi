package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/co60ca/beaconpi"
	"io/ioutil"
	"log"
	"net"
	//	"bufio"
	"flag"
)

func main() {
	log.SetFlags(log.Lshortfile)

	var x509cert, x509key string
	flag.StringVar(&x509cert, "serv-cert", "", "x509 server public certificate")
	flag.StringVar(&x509key, "serv-key", "", "x509 server private key")
	flag.Parse()

	if x509cert == "" || x509key == "" {
		log.Fatal("Both cert and key must be provided")
	}

	cerpoolrootca := beaconpi.LoadFileToCert(x509cert)

	cer, err := tls.LoadX509KeyPair(x509cert, x509key)
	if err != nil {
		log.Fatal(err)
	}
	port := beaconpi.DEFAULT_PORT
	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
		ClientCAs:    cerpoolrootca,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	ln, err := tls.Listen("tcp", ":"+port, config)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	j := make([]byte, 64)
	conn.Read(j)
	log.Println(string(j))
	defer conn.Close()
}
