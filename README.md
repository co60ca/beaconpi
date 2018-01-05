# Beaconpi

A system for monitoring Beacons using Raspberry Pis

## Components
### Beacons
  Currently supported beacons include only the Apple iBeacon of any  
  brand.
### Edges
  Edges receive the announcements from the beacons and relay them onto  
  the beacon server. This is accomplished using Raspberry Pis.
### Beacon Client
  The edges use the beacon client, which communicates with the beacon  
  server to get control commands and relay the beacon sightings
### Beacon Server
  Manages the data from the edge units and the database.
### Database
  SQL database the efficiently indexs and stores all beacon data.
### Metrics Server
  Serves data from the database

## Requirements
- Hardware Requirements
  - Raspberry Pis, with bluetooth and some network connectivity,  
Raspberry Pi 3 works best as it has built in Bluetooth and WiFi.
  - Centralized server (application server) with network open to  
Raspberry Pis, cloud is acceptable.
- Software Requirements
  - [Go compiler](https://golang.org/doc/install) (1.7+) suggested  
on both the application server and clients (Raspberry Pi), we can  
cross compile but updates are easier if the client has the compiler too.
  - [Raspbian](https://www.raspberrypi.org/downloads/raspbian/)  
OS for the Raspberry Pis, you should use the minimal image (lite)
  - [bluez](https://packages.debian.org/stretch/bluez) Install on  
Raspberry Pis to provide `hcitool`
  - [bluez-hcidump](https://packages.debian.org/stretch/bluez-hcidump)  
Install on Raspberry Pis to provide `hcidump` which is depreciated  
in bluez this requirement will be removed in future versions  
  - Application Server can run any Unix-like OS
  - [Postgres Sql 9.5.10+](https://www.postgresql.org/) for data storage  
and persistant configuration. Install to application server only.
  - 

## Configuration Requirements
  - Raspberry Pis must be altered to allow non root users to use `hcitool`  
the following command  
`sudo setcap 'cap_net_raw,cap_net_admin+eip' $(which hcitool)`  
satisifies this requirement.

## Build Requirements
  - GNU Make (recommended install requirement)
  - git (recommended install requirement)
  - OpenSSL
  - Node and NPM (optional)

