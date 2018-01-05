#!/bin/bash
set -eu
n=${1:-""}

if ! [ $n ] ; then
  echo "Please provide a number"
  exit 1
fi

of="client-options.cfg"
certfold="../x509"
certgen="./generate-keys.sh"
uuid=`cat /proc/sys/kernel/random/uuid`
uuidnodash=`echo "$uuid" | sed 's/-//g'`
# Client name
cn="client-`printf \"%x\" \"$RANDOM$RANDOM$RANDOM\"`"

outdir="$PWD/$cn-$n"
mkdir "$outdir"
cd "$certfold"
"$certgen" newclient "$cn-$n"
mv $cn* "$outdir"
cp server.crt "$outdir"
cd -

logfile=beacon.log


cat > "${outdir}/client-options.cfg" <<EOF
CLIENT_CERT="${cn}-${n}.crt"
CLIENT_KEY="${cn}-${n}.key"
CLIENT_UUID="$uuidnodash"
SERVER_CERT="server.crt"
SERVER="3508data.soe.uoguelph.ca"
PORT="32969"
EOF
echo "Pi ${n} has UUID:" | tee -a $logfile
echo "$uuid" | tee -a $logfile
