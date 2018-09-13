# Beaconpi
A system for monitoring Beacons using Raspberry Pis. Created as a part of a Percom 2018 Demo Paper submission. [Linked here](https://co60.ca/blog/ble-beacon-based-patient-tracking-smart-care-facilities). The intention is to build on this system to make location predictions more accurate.

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
  - [Nodejs & npm](https://nodejs.org/) used for the web interface
  - [Raspbian](https://www.raspberrypi.org/downloads/raspbian/) OS for the Raspberry Pis, you should use the minimal image (lite)
  - [bluez](https://packages.debian.org/stretch/bluez) Install on Raspberry Pis to provide `hcitool`
  - [bluez-hcidump](https://packages.debian.org/stretch/bluez-hcidump)  
Install on Raspberry Pis to provide `hcidump` which is depreciated in bluez; this requirement will be removed in future versions
  - Application Server can run any Unix-like OS that supports Go
  - [Postgres Sql 9.5.10+](https://www.postgresql.org/) for data storage and persistant configuration. Install to application server only.
  - Python 3.5+ for metricsserver components tracking components

## Configuration Requirements
  - Raspberry Pis must be altered to allow non root users to use `hcitool` the following command  
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
3. Once your database is created you need to apply the schema. Use the script `etc/devtools/setup-db.sh` to apply migrations, it uses the `environment.cfg` file in order to connect to the database
4. Compile the Go programs. Copy this repo to your `$GOPATH/src/github.com/co60ca/beaconpi` directory. Copying this repo can also be done with `go get github.com/co60ca/beaconpi`. Then run `make all` from the beaconpi directory. Make will build the programs to `build/` relative to the `beaconpi` directory. It will not `beaconclient` program to `arm64` which is the target platform for Raspberry Pi 3. You will need to do this on your target edge nodes.
5. Copy `beaconclient` to your cluster of Raspberry Pis, you can put this anywhere you like as the application relies on no relative pathes. A possible suggestion is to put it in the `$GOPATH/bin` directory as some helper scripts are designed to expect that.
6. Copy `beaconserver` to the application server. 
Copy `etc/devtools/start-beacon-server.sh` and `etc/devtools/environment.cfg` to the application server as well. The configuration for the database is located in `environment.cfg` and defaults to password based authentication. Check [lib/pq](https://github.com/lib/pq) for more options.
7. Copy `metricsserver` to the application server. Suggested location is the same. Additionally copy the `etc/devtools/start-web.sh` and `etc/devtools/environment.cfg`
8. Generate security keys. Change directory to `etc/x509`. You may generate the client and server keys by using the `generate-keys.sh` helper script however you should only use the `./generate-keys.sh newserver` mode of the script. Client keys are made in the next step automatically. Generating a new server key will invalidate all prior keys.
9. Generate client keys. Change directory to `etc/client-maker`. Use `setup-client.sh` from the same machine you generated the new server keys. Generate new clients by running `./setup-client.sh <name of client>` the name of the client can be anything. For simplicity you should just use numbers. To generate many at once you can use a bash for loop.
    ```bash
    # Creates 20 clients
    for i in {0..19} ; do ./setup-client.sh ${i} ; done
    ```
    The helper script creates folders with the required configure files for the client start helper script `start-client.sh`.
10. Copy the client files to each of the edge units. Ensure the files in each folder and the start-client.sh are in the same directory on each of the edge units.

# Web Install
The whole system can be operated only using the sql database for reporting however the JavaScript/Web interface is build using Node/npm. Before running the below command(s) change the target directory to the host/port of your application server system. Use https:// if available.
Open `etc/beaconpi-react/src/config.js` and change 

`const home = '<url>'`

to 

`const home = 'http(s)://<fully qualified domain name>:<port>'`

to the root of where your web server will be hosted.
Then change 

`const app = 'http(s)://<fully qualified domain name>:<port>'`

to the path where your app server will be hosted.

Finally install required 3rd party libraries and build the bundle
```bash
# Get required packages and build bundle
npm install && npm run build
```
then serve this bundle in `build/` with any webserver (even Github Pages!)

## Required Configuration
Version 2 has introduced beaconpi-react which is a very easy to use web interface, it should allow you to add beacons, edges, and users to the system without using SQL. For the Lateration tab SQL is still required as no system admin page has been made for the MapConfigs yet.

All users in the current version are admins and have full access to the system, the first user must be made in SQL unfortunatly. To do so:
```
insert into webauth_users 
  (displayname, email, password, active) values
  ('<your displayname>', '<your email>', <password from next step>, 1)
```
the password must be hashed in advance, since we use `github.com/co60ca/webauth` we can use the password entry tool from that application. So compile 
`go install src/github.com/co60ca/webauth/passgen` then run passgen from `$GOPATH/bin` which will ask you for your password then give you the exact text to put in the blank in the step prior.

Once you have the user created you can login to the webinterface once started. The system can operate with no data in the database. 
 
1. Beacons - Simply add the iBeacon settings under the Admin -> Beacon tab
2. Edges - Simply add the Edge settings under the Admin -> Edge tab, the UUID for each edge is given in the `/client-options.cfg` for each edge

## Starting Everything
The clients will fail and retry to connect to the server so as long as the database is up any order is permitted. However this the supported startup sequence.
1. Start the database. Check with your OS vendor on how to do this. You probably want this to start with the application server OS.
2. Start the beaconserver. Use `./start-beacon-server.sh` with the correct configuration set in this file or use `beaconserver` directly with the correct command line arguments. See `beaconserver --help` for the arguments.
3. Start each of the clients. You probably want to configure this to start with each of the clients. Use `./start-client.sh` to do so.
4. Start the metricserver. Simply run `./start-metrics-server.sh`.
5. Start your webserver for the client facing code.
  
  
  
  
  

