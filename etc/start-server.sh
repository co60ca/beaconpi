#!/bin/bash
set -eu
DSN="user=postgres password=notapassword dbname=beacons sslmode=disable"
DN=postgres
CERT=x509/server.crt
KEY=x509/server.key

$GOPATH/bin/beaconserv -db-datasource-name "${DSN}" -db-driver-name ${DN} \
  -serv-cert ${CERT} -serv-key ${KEY}
