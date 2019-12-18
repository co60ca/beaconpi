#!/bin/bash

prog=/mnt/storage/Source_Code/Projects/beacons

$GOPATH/bin/beaconserv --serv-cert "${prog}/x509/server.crt" --serv-key "${prog}/x509/server.key" \
    --db-driver-name "postgres" --db-datasource-name "user=postgres password=Meowmix9 dbname=beacons sslmode=disable"
