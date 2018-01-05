# Beaconpi
A system for monitoring Beacons using Raspberry Pis. Created as a part of a Percom 2018 Demo Paper submission. Linked here (Not yet available). The intention is to build on this system to make location predictions more accurate.

## Components
Beaconpi requires many physical and software components. We give a high level overview of what each component is in this section.
- Beacons
  Currently supported beacons include only the Apple iBeacon of any brand.
- Edges
  Edges receive the announcements from the beacons and relay them onto the beacon server. This is accomplished using Raspberry Pis.
- Beacon Client
  The edges use the beacon client, which communicates with the beacon  
  server to get control commands and relay the beacon sightings
- Beacon Server
  Manages the data from the edge units and the database.
- Database
  SQL database the efficiently indexs and stores all beacon data.
- Metrics Server
  Serves data from the database
- Web Client
  Serves data from the Metrics Server into visual format

## Requirements
Requirements are software/hardware requirements for the entire system. The software should be installed before proceeding.
- Hardware Requirements
  - Raspberry Pis, with bluetooth and some network connectivity, Raspberry Pi 3 works best as it has built in Bluetooth and WiFi.
  - Centralized server (application server) with network open to Raspberry Pis, cloud is acceptable.
- Software Requirements
  - [Go compiler](https://golang.org/doc/install) (1.7+) suggested on both the application server and clients (Raspberry Pi), we can cross compile but updates are easier if the client has the compiler too.
  - [Raspbian](https://www.raspberrypi.org/downloads/raspbian/) OS for the Raspberry Pis, you should use the minimal image (lite)
  - [bluez](https://packages.debian.org/stretch/bluez) Install on Raspberry Pis to provide `hcitool`
  - [bluez-hcidump](https://packages.debian.org/stretch/bluez-hcidump)  
Install on Raspberry Pis to provide `hcidump` which is depreciated in bluez; this requirement will be removed in future versions
  - Application Server can run any Unix-like OS that supports Go
  - [Postgres Sql 9.5.10+](https://www.postgresql.org/) for data storage and persistant configuration. Install to application server only.


## Configuration Requirements
  - Raspberry Pis must be altered to allow non root users to use `hcitool`the following command  
    ```bash
    sudo setcap 'cap_net_raw,cap_net_admin+eip' $(which hcitool)
    ```
    satisifies this requirement.

## Build Requirements
  - GNU Make (recommended install requirement)
  - git (recommended install requirement)
  - OpenSSL
  - Node and NPM (optional for webinterface)

## Install
  The instructions below are suggestions. Make any directories that don't already exist. These instructions do not walk you through basic Linux commands so intermediate understanding of Unix-like systems is required. Some helper programs assume `$GOPATH` is set, you should set it if not already set. It should match your Go path. By default Go uses `$HOME/go`. The database schema(tables) must be created before the application is run.
  
1. Postgres must be running, check with your OS vendor how to do this. In some cases `sudo systemctl start postgres` is sufficient but will vary across vendors.
2. `CREATE DATABASE` command or `createdb` program can be used to make the database. You can use any authentication that [lib/pq](https://github.com/lib/pq) supports but my suggestion is to use `trust` authentication for simplicity and should be as secure as your user on the server. Use `createdb beacons` or any other name you would like.
3. Once your database is created you need to source the schema. Source the commands from the files in `etc/db/` in order. To update the database run the migrations you haven't done thus far. For example you can do 
`\s etc/db/mig_0001.sql`
interactively from `psql beacons`
or in batch from command line using 
`psql -f etc/db/mig_0001.sql beacons`
4. Compile the Go programs. Copy this repo to your `$GOPATH/src/github.com/co60ca/beaconpi` directory. Copying this repo can also be done with `go get github.com/co60ca/beaconpi`. Then run `make all` from the beaconpi directory. Make will build the programs to `builj/` relative to the `beaconpi` directory. It will also cross compile the `beaconclient` program to `arm64` which is the target platform for Raspberry Pi 3.
5. Copy `beaconclient` to the Raspberry Pi, you can put this anywhere you like as the application relies on no relative pathes. A possible suggestion is to put it in the `$GOPATH/bin` directory as some helper scripts are designed to expect that.
6. Copy `beaconserv` to the application server. Again the location is not important but some helper scripts expect `$GOPATH/bin`
Copy `etc/start-server.sh` to the application server as well. The configuration for the database is located in there and defaults to password based authentication. Check [lib/pq](https://github.com/lib/pq) for more options.
7. Copy `metricsserv` to the application server. Suggested location is the same.
8. Generate security keys. Change directory to `etc/x509`. You may generate the client and server keys by using the `generate-keys.sh` helper script however you should only use the `./generate-keys.sh newserver` mode of the script. Client keys are made in the next step automatically. Generating a new server key will invalidate all prior keys.
9. Generate client keys. Change directory to `etc/client-maker`. Use `setup-client.sh` from the same machine you generated the new server keys. Generate new clients by running `./setup-client.sh <name of client>` the name of the client can be anything. For simplicity you should just use numbers. To generate many at once you can use a bash for loop.
    ```bash
    # Creates 20 clients
    for i in {0..19} ; do ./setup-client.sh ${i} ; done
    ```
    The helper script creates folders with the required configure files for the client start helper script `start-client.sh`.
10. Copy the client files to each of the edge units. Ensure the files in each folder and the start-client.sh are in the same directory on each of the edge units.
## Optional Install
The whole system can be operated only using the sql database for reporting however the JavaScript/Web interface is build using Node/npm. Before running the below command(s) change the target directory to the host/port of your application server system. Use https:// if available.
Open `etc/client-triangulation/src/index.js` and change 
`var target = '<url>'`
to 
`http(s)://<fully qualified domain name>:<port>`

```bash
# Get required packages and build bundle
npm install 
```
then serve this file with any webserver (even Github Pages!) or use `file:///home/mae/beaconpi/etc/client-triangulation/triangulation.html` from any modern web browser.
## Required Configuration
The database has allowlists tables for edge units and beacons. 
1. Insert iBeacon details into `beacon_list`. All fields are required and must be correct for the edge units to record the sightings. iBeacons that are not in the database will be ignored by the edge units. For example, run the following sql from `psql beacons` interactively
    ```sql
    insert into beacon_list (label, uuid, major, minor) 
    values ('beacon a', '6b4f6d90-95b1-497d-be84-a12760413a3b', 0, 0)
    ``` 
    Note that the uuid, major, minor must match the fields from the iBeacon in question.

2. Insert edge details into `edge_node`. The edge nodes have a UUID that they identify themselves to the server with. The server will drop any sightings from beacons they don't identify. The client UUIDs will be located in `etc/client-maker/beacon.log` with the format: `"Pi ${n} has UUID:\n${uuid}"` Insert the rows as follows:
    ```sql
    insert into edge_node (uuid, title, room, location, description)
    values ('<uuid from beacon.log>', '<helpful title>', '<room name>', '<location name>', '<description>')
    ``` 
    the important field is the uuid, but for record keeping the other fields are provided. Title is restricted to 60 characters but the other fields are unstructured. Consider using location or description to provide specific details in the location or type of edge unit.
## Starting Everything
The clients will fail and retry to connect to the server so as long as the database is up any order is permitted. However this the supported startup sequence.
1. Start the database. Check with your OS vendor on how to do this. You probably want this to start with the application server OS.
2. Start the beaconserv. Use `./start-server.sh` with the correct configuration set in this file or use `beaconserv` directly with the correct command line arguments. See `beaconserv --help` for the arguments.
3. Start each of the clients. You probably want to configure this to start with each of the clients. Use `./start-client.sh` to do so.
4. Start the metricserv. Simply run `./metricsserv` the metrics server currently has no arguments and the port is fixed serving on port `32967`
  
  
  
  
  

