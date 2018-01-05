#!/bin/bash

if ! [ ${GOPATH:-""} ] ; then
        source .bash_profile
fi

set -eu

if [ -f client-options.cfg ] ; then
        source client-options.cfg
else
        echo "Please use setup-client.sh first"
        exit 1
fi

"$GOPATH/bin/beaconclient" -client-cert-file "${CLIENT_CERT}" -client-key-file "${CLIENT_KEY}" -client-uuid "${CLIENT_UUID}" -serv-cert-file "${SERVER_CERT}" -serv-host "${SERVER}" -serv-port "${PORT}"

