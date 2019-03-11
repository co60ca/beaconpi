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

package beaconpi

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"math"
	"strconv"
	"strings"
	"time"
)

// dbHandler stores data for use in opening a connection to the DB
type dbHandler struct {
	Drivername     string
	DataSourceName string
}

// openDB is a helper to open a connection to the DB
func (dbh *dbHandler) openDB() (*sql.DB, error) {
	return sql.Open(dbh.Drivername, dbh.DataSourceName)
}

// dbAddLogsForBeacons given a packet and edge add the logs for the packet
// into the database
func dbAddLogsForBeacons(pack *BeaconLogPacket, edgeid int, db *sql.DB) error {
	if len(pack.Logs) == 0 {
		return nil
	}

	beaconids, err := dbGetIDForBeacons(pack, db)
	if err != nil {
		return err
	}

	data := make([]struct {
		Datetime time.Time
		Rssi     int
		Beaconid int
	}, len(pack.Logs))

	// Line is gaurunteed by guard at top
	firstlog := pack.Logs[0]
	log.Debug("Time on beacon recieved ", firstlog)

	maxtimediff := 5.0
	diff := firstlog.Datetime.Sub(time.Now()).Seconds()
	if math.Abs(diff) > 5.0 {
		errorstr := fmt.Sprintf("Time between server and client is greater than %f, (%f)", maxtimediff, diff)
		log.Info(errorstr)
		dbInsertError(ERROR_DESYNC, ERROR_INFO, errorstr, edgeid, "2 minutes", db)
		return errors.New(errorstr)
	}
	for i, logv := range pack.Logs {
		data[i].Datetime = logv.Datetime
		data[i].Rssi = int(logv.Rssi)
		// TODO(mae) additional error logging here for ids that don't exist
		data[i].Beaconid = beaconids[logv.BeaconIndex]
	}
	for _, row := range data {
		rows, err := db.Query(`
			insert into beacon_log
			(datetime, beaconid, edgenodeid, rssi)
			VALUES
			($1, $2, $3, $4)
		`, row.Datetime.UTC(), row.Beaconid, edgeid, row.Rssi)
		if err != nil {
			return errors.New("Failed to insert into DB: " + err.Error())
		}
		rows.Close()
	}
	if len(data) != 0 {
		log.Debugf("Completed inserting %d records", len(data))
	}
	return nil
}

// dbGetIDForBeacons converts the ID references in the request to integer
// ids in the DB
func dbGetIDForBeacons(pack *BeaconLogPacket, db *sql.DB) ([]int, error) {
	//TODO(mae) optimize this
	rval := make([]int, len(pack.Beacons))
	for i, b := range pack.Beacons {
		var tempid int
		err := db.QueryRow(`
			select id
			from ibeacons
			where uuid = $1`, b.Uuid.String()).Scan(&tempid)
		if err != nil {
			return []int{}, errors.New("Failed while scanning beacon ids: " + err.Error())
		}
		rval[i] = tempid
	}
	return rval, nil
}

// dbGetBeacons returns all Beacons in the database
func dbGetBeacons(db *sql.DB) ([]BeaconData, error) {
	rval := make([]BeaconData, 0, 8)

	rows, err := db.Query(`
		select uuid, major, minor
		from ibeacons
	`)
	if err != nil {
		return rval, errors.New("Failed while executing ibeacons query:" + err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var (
			uuid  string
			major uint16
			minor uint16
		)
		if err := rows.Scan(&uuid, &major, &minor); err != nil {
			return rval, errors.New("Failed while scanning ibeacons: " + err.Error())
		}
		uuid = strings.Replace(uuid, "-", "", -1)
		hexb, err := hex.DecodeString(uuid)
		if err != nil {
			return rval, errors.New("Failed while decoding hex: " + err.Error())
		}
		bdtemp := BeaconData{Major: major, Minor: minor}
		copy(bdtemp.Uuid[:], hexb[:16])
		rval = append(rval, bdtemp)
	}
	return rval, nil
}

// dbInsertControlLog accepts a packet containing a Control Log from the edge
// in the DB with ID edgenodeid and inserts the log
func dbInsertControlLog(edgenodeid int, packet *BeaconLogPacket, db *sql.DB) error {
	rows, err := db.Query(`
		insert into control_log (edgenodeid, data)
		values ($1, $2)
	`, edgenodeid, packet.ControlData)
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

// dbGetControl sets that the Control Message was completed
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
	return strconv.Itoa(id) + "\n" + data, nil
}

// updateEdgeLastUpdate updates the last time the edge has been seen for the
// application for audit purposes
func updateEdgeLastUpdate(uuid Uuid, db *sql.DB) {
	_, err := db.Exec(`update edge_node set lastupdate = current_timestamp
			where uuid = $1`, uuid.String())
	if err != nil {
		log.Infof("Failed to update edge_node lastupdate to current_timestamp %s", err)
	}
}

// dbCheckUuid returns the ID of the edge or returns an error if it doesn't exist
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

const (
	ERROR_TRACE = 0
	ERROR_DEBUG = 1
	ERROR_INFO  = 2
	ERROR_WARN  = 3
	ERROR_ERROR = 4
	ERROR_FATAL = 5
)

const (
	ERROR_NULL = iota
	ERROR_DESYNC
)

//
// every is a postgres interval
func dbInsertError(errorid, errorlevel int, errortext string, edgenodeid int, every string, db *sql.DB) {

	query := `insert into system_errors (error_id, error_level, error_text, edgenodeid)
		values ($1, $2, $3, $4)`

	erroridp := &errorid
	edgenodeidp := &edgenodeid

	if *erroridp == 0 {
		erroridp = nil
	}
	if *edgenodeidp == 0 {
		edgenodeidp = nil
	}

	var count int
	err := db.QueryRow(`select count(*) from system_errors 
        where edgenodeid=$1 and error_id=$2 and 
        current_timestamp - datetime < `+every, edgenodeidp, erroridp).Scan(&count)
	if count > 0 {
		return
	}

	_, err = db.Exec(query, erroridp, errorlevel, errortext, edgenodeidp)
	if err != nil {
		log.Warnf("Failed to insert error \"%s\" due to error: %s", errortext, err)
	}
}
