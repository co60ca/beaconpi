package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"github.com/co60ca/beaconpi"
	"io/ioutil"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)
	var servcertfile, clientcertfile, clientkeyfile string
	flag.StringVar(&servcertfile, "serv-cert-file", "", "Has trusted keys")
	flag.StringVar(&clientcertfile, "client-cert-file", "", "")
	flag.StringVar(&clientkeyfile, "client-key-file", "", "")
	flag.Parse()

	certpool := beaconpi.LoadFileToCert(servcertfile)
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
	port := beaconpi.DEFAULT_PORT
	conn, err := tls.Dial("tcp", "localhost:"+port, conf)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("OwO\n"))
	if err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 100)
	_, err = conn.Read(buf)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(buf))
}
