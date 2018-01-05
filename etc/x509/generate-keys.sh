#!/bin/bash

mode=$1
clientname=$2

if ! [ $mode ] ; then
  echo "Usage: $0 newserver|newclient [clientname]"
  exit 1
fi

if [ "$mode" == "newserver" ] ; then
  openssl genrsa -out server.key 2048
  openssl ecparam -genkey -name secp384r1 -out server.key
  openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
fi

if [ "$mode" == "newclient" ] ; then
  if ! [ $clientname ] ; then
    echo "Please provide client name"
    exit 1
  fi
  openssl genrsa -out "$clientname".key 2048
  yes "" | head -n 9 | \
  openssl req -new -key "$clientname".key -out "$clientname".csr
  # Sign the csr

  openssl x509 -req -days 3650 -in "$clientname".csr -CA server.crt -CAkey server.key \
    -CAcreateserial -CAserial ca.seq -out "$clientname".crt 
fi
