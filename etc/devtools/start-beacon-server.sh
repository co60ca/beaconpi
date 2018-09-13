#!/bin/bash
# Usage of /home/bkennedy/go/src/github.com/co60ca/beaconpi/build/metricsserv:
#   -allowed-origin string
#         Origin, including http(s) for valid domains that may access the resource, * is invalid for our application. (default "http://localhost:3000")
#   -db-datasource-name string
#         Required: The database datasource name, may be multiple tokes
#   -db-driver-name string
#         Required: The database driver name
#   -port string
#         Required: Port for serving http
set -eu

source environment.cfg

cd $HOME

app=$GOPATH/src/github.com/co60ca/beaconpi/build/beaconserv

"${app}" -db-datasource-name "${DBDATASOURCENAME}" \
        -db-driver-name "${DBDRIVER}" \
        -serv-cert "${APPCERTSERVER}" \
        -serv-key "${APPKEYSERVER}"
