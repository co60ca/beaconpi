// Beacon Pi, a edge node system for iBeacons and Edge nodes made of Pi
// Copyright (C) 2017  Maeve Kennedy
// 
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// 
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// 
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//
package beaconpi

import (
	"database/sql"
	_ "github.com/lib/pq"
	"errors"
	"time"
	"log"
	"encoding/hex"
	"strings"
	"strconv"
)

type dbHandler struct {
	Drivername string
	DataSourceName string
}

func (dbh *dbHandler) openDB() (*sql.DB, error) {
	return sql.Open(dbh.Drivername, dbh.DataSourceName)
}

func dbAddLogsForBeacons(pack *BeaconLogPacket, edgeid int, db *sql.DB) error {
	beaconids, err := dbGetIDForBeacons(pack, db)
	if err != nil {
		return err
	}

	data := make([]struct {
		Datetime time.Time
		Rssi int
		Beaconid int
	}, len(pack.Logs))
	for i, logv := range pack.Logs {
		data[i].Datetime = logv.Datetime
		data[i].Rssi = int(logv.Rssi)
		// TODO(mae) additional error logging here for ids that don't exist
		data[i].Beaconid = beaconids[logv.BeaconIndex]
	}
	log.Printf("Inserting %d records", len(data))
	for _, row := range data {
		rows, err := db.Query(`
			insert into beacon_log
			(datetime, beaconid, edgenodeid, rssi)
			VALUES
			($1, $2, $3, $4)
		`, row.Datetime, row.Beaconid, edgeid, row.Rssi)
		if err != nil {
			return errors.New("Failed to insert into DB: " + err.Error())
		}
		rows.Close()
	}
	log.Printf("Completed inserting %d records", len(data))
	return nil
}

func dbGetIDForBeacons(pack *BeaconLogPacket, db *sql.DB) ([]int, error) {
	rval := make([]int, len(pack.Beacons))
	for i, b := range pack.Beacons {
		var tempid int
		err := db.QueryRow(`
			select id
			from beacon_list
			where uuid = $1`, b.Uuid.String()).Scan(&tempid)
		if err != nil {
			return []int{}, errors.New("Failed while scanning beacon ids: " + err.Error())
		}
		rval[i] = tempid
	}
	return rval, nil
}

func dbGetBeacons(db *sql.DB) ([]BeaconData, error) {
	rval := make([]BeaconData, 0, 8)

	rows, err := db.Query(`
		select uuid, major, minor
		from beacon_list
	`)
	if err != nil {
		return rval, errors.New("Failed while executing beacon_list query:" + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var (
			uuid string
			major uint16
			minor uint16
		)
		if err := rows.Scan(&uuid, &major, &minor); err != nil {
			return rval, errors.New("Failed while scanning beacon_list: " + err.Error())
		}
		uuid = strings.Replace(uuid, "-", "", -1)
		hexb, err := hex.DecodeString(uuid)
		if err != nil {
			return rval, errors.New("Failed while decoding hex: " +  err.Error())
		}
		bdtemp := BeaconData{Major: major, Minor: minor}
		copy(bdtemp.Uuid[:], hexb[:16])
		rval = append(rval, bdtemp)
	}
	return rval, nil
}

func dbInsertControlLog(edgenodeid int, packet *BeaconLogPacket, db *sql.DB) error {
	rows, err := db.Query(`
		insert into control_log (edgenodeid, data)
		values ($1, $2)
	`, edgenodeid, packet.ControlData);
	if err != nil {
		return errors.New("Failed to insert control log: " + err.Error())
	}
	rows.Close()
	return nil
}

func dbCompleteControl(packet *BeaconLogPacket, db *sql.DB) error {
	edgeid, err := dbCheckUuid(packet.Uuid, db)
	if err != nil {
		return errors.New("Failed to update control because: " + err.Error())
	}
	pdata := strings.SplitN(packet.ControlData, "\n", 2)
	if len(pdata) != 2 {
		return errors.New("Failed to update control because split invalid")
	}
	controlid, err := strconv.Atoi(pdata[0])
	if err != nil {
		return errors.New("Failed to update control because: " + err.Error())
	}
	rows, err := db.Query(`
		update control_commands
		set COMPLETED = TRUE
		where edgenodeid = $1 and id = $2
	`, edgeid, controlid)
	if err != nil {
		return errors.New("Failed to update control because: " + err.Error())
	}
	rows.Close()
	rows, err = db.Query(`
		insert into control_log 
		(edgenodeid, controlid, data) VALUES
		($1, $2, $3)
	`, edgeid, controlid, pdata[1])
	if err != nil {
		return errors.New("Failed to update control because: " + err.Error())
	}
	rows.Close()
	return nil
}

func dbGetControl(packet *BeaconLogPacket, db *sql.DB) (string, error) {
	edgeid, err := dbCheckUuid(packet.Uuid, db)
	if err != nil {
		return "", errors.New("Failed to get control because: " + err.Error())
	}
	var data string
	var id int
	err = db.QueryRow(`
		select id, data
		from control_commands
		where edgenodeid = $1 and completed = FALSE
		order by datetime
		limit 1
	`, edgeid).Scan(&id, &data)

	if err != nil {
		return "", errors.New("Failed to get control because: " + err.Error())
	}
	return strconv.Itoa(id) + "\n" + data , nil
}

// Returns the ID of the edge
func dbCheckUuid(uuid Uuid, db *sql.DB) (int, error) {
	var edgeid int
	err := db.QueryRow(`
		select id 
		from edge_node 
		where uuid = $1`, uuid.String()).Scan(&edgeid)
	if err != nil {
		return 0, errors.New("Error occured while attempting to fetch Uuid: " + err.Error())
	}
	return edgeid, nil
}
