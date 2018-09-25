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

// metricsserv builds cause problems with other binaries due to inclusion
// of packages that require python3
// +build metrics

package beaconpi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// jsonResponse helper for sending simple JSON objects
func jsonResponse(w http.ResponseWriter, results map[string]interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(results)
	if err != nil {
		log.Error("Failed to write jsonResponse", err)
		http.Error(w, "Server error", 500)
	}
	return
}

// quickStats returns a few simple statistics useful for system administration
func quickStats() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Errorf("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

		// Active edges in last 10 minutes
		var (
			countedges   int
			countbeacons int
		)

		type edges struct {
			Title       string
			Room        string
			Location    string
			Description string
		}
		var inactEdges []edges
		rowsedge, err := db.Query(`
			select * from inactive_edges()
		`)
		if err != nil {
			log.Errorf("Failed while getting inactive edges %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer rowsedge.Close()
		for rowsedge.Next() {
			var t edges
			var desc sql.NullString
			if err := rowsedge.Scan(&t.Title, &t.Room, &t.Location,
				&desc); err != nil {
				log.Errorf("Failed while scanning edges %s", err)
				http.Error(w, "Server failure", 500)
				return
			}
			t.Description = desc.String
			inactEdges = append(inactEdges, t)
		}

		if err = db.QueryRow(`select count(*) 
				from edge_node 
				`).Scan(&countedges); err != nil {
			log.Printf("Failed while getting total count %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		if err = db.QueryRow(`select count(*) 
				from ibeacons 
				`).Scan(&countbeacons); err != nil {
			log.Printf("Failed while getting total beacon count %s", err)
			http.Error(w, "Server failure", 500)
			return
		}

		type ibeacons struct {
			Label string
			Uuid  string
			Major int
			Minor int
		}
		var inactivebeacons []ibeacons
		rows, err := db.Query(`
			select * from inactive_beacons()
		`)
		defer rows.Close()
		if err != nil {
			log.Printf("Failed to get inactive beacons %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		for rows.Next() {
			var t ibeacons
			if err = rows.Scan(&t.Label, &t.Uuid, &t.Major, &t.Minor); err != nil {
				log.Errorf("Failed while scanning beacons %s", err)
				http.Error(w, "Server failure", 500)
				return
			}
			inactivebeacons = append(inactivebeacons, t)
		}
		jsonResponse(w, map[string]interface{}{
			"InactiveBeacons": inactivebeacons,
			"InactiveEdges":   inactEdges,
			"EdgeCount":       countedges,
			"InaEdgeCount":    len(inactEdges),
			"BeaconCount":     countbeacons,
			"InaBeaconCount":  len(inactivebeacons),
		})
		return
	})
}

// getBeacons returns all beacons to the requestor
func getBeacons() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

		rows, err := db.Query(`
			select id, label, uuid, major, minor
			from ibeacons
			order by label`)
		if err != nil {
			log.Infof("Failed while quering beacons %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		type ibeacon struct {
			Id    int
			Label string
			Uuid  string
			Major int
			Minor int
		}

		var outdata []ibeacon

		for rows.Next() {
			var b ibeacon
			if err = rows.Scan(&b.Id, &b.Label, &b.Uuid, &b.Major,
				&b.Minor); err != nil {
				log.Errorf("Failed to scan beacons in GetBeacons %s", err)
				http.Error(w, "Server failure", 500)
				return
			}
			outdata = append(outdata, b)
		}
		jsonResponse(w, map[string]interface{}{
			"Beacons": outdata,
		})
		return
	})
}

// getEdges returns all the edges to the caller
func getEdges() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

		rows, err := db.Query(`
			select id, uuid, title, room, location, description, bias, gamma
			from edge_node
			order by title`)
		if err != nil {
			log.Errorf("Failed while quering edges %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		type edge struct {
			Id          int
			Uuid        string
			Title       string
			Room        string
			Location    string
			Description string
			Bias        float64
			Gamma       float64
		}
		var outdata []edge

		for rows.Next() {
			var edge edge
			var description sql.NullString
			if err = rows.Scan(&edge.Id, &edge.Uuid, &edge.Title,
				&edge.Room, &edge.Location, &description,
				&edge.Bias, &edge.Gamma); err != nil {
				log.Errorf("Failed to scan edges in GetEdges %s", err)
				http.Error(w, "Server failure", 500)
				return
			}
			edge.Description = description.String
			outdata = append(outdata, edge)
		}
		jsonResponse(w, map[string]interface{}{
			"Edges": outdata,
		})
		return
	})
}

// Function that checks if the field is greater than the minlen
// if not it will return an error. This is chainable however,
// the first error is the only one that is returned
func validateLen(pass error, field interface{}, fieldn string, minlen int) error {
	if pass != nil {
		return pass
	}
	switch v := field.(type) {
	case string:
		if len(v) < minlen {
			return errors.New(
				fmt.Sprintf("Field %s was too short %d < %d, (value %v)",
					fieldn, len(v), minlen, field))
		}
		return nil
	case []interface{}:
		if len(v) < minlen {
			return errors.New(
				fmt.Sprintf("Field %s was too short %d < %d, (value %v)",
					fieldn, len(v), minlen, field))
		}
		return nil
	}
	return errors.New(fmt.Sprintf("Field %s is unknown type, (value %v)", fieldn, field))
}

// modEdge allows the caller to modify edges through the administrative panel
func modEdge() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		input := struct {
			Id          int
			Uuid        string
			Title       string
			Room        string
			Location    string
			Description string
			Bias        float64
			Gamma       float64
			Option      string
		}{}
		dec := json.NewDecoder(req.Body)
		err := dec.Decode(&input)
		if err != nil {
			log.Infof("Failed to decode json request in ModEdge %s", err)
			http.Error(w, "Invalid Request", 400)
			return
		}
		if input.Option != "rem" {
			err = validateLen(nil, input.Uuid, "Uuid", 16)
			err = validateLen(err, input.Title, "Title", 1)
			err = validateLen(err, input.Room, "Room", 1)
			err = validateLen(err, input.Location, "Location", 1)
			err = validateLen(err, input.Option, "Option", 3)
			if err != nil {
				log.Infof("Failed validation %s", err)
				http.Error(w, "Invalid Request", 400)
				return
			}
		}

		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Errorf("Error opening DB %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()
		switch input.Option {
		case "new":
			_, err = db.Exec(`insert into edge_node (uuid, title, room, location,
					description, bias, gamma) values ($1, $2, $3, $4, $5, $6, $7)`,
				input.Uuid, input.Title, input.Room, input.Location,
				input.Description, input.Bias, input.Gamma)
		case "mod":
			_, err = db.Exec(`update edge_node set 
					(uuid, title, room, location, description, bias, gamma) = 
					($1, $2, $3, $4, $5, $6, $7) where id = $8`, input.Uuid, input.Title,
				input.Room, input.Location, input.Description, input.Bias,
				input.Gamma, input.Id)
		case "rem":
			_, err = db.Exec(`delete from edge_node
					where id = $1`, input.Id)
		default:
			log.Infof("Option invalid given \"%s\"", input.Option)
			http.Error(w, "Invalid Request", 400)
			return
			// Mod
		}
		if err != nil {
			log.Infof("Failed operation on DB %s", err)
			http.Error(w, "Invalid Request", 400)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"Success": true,
		})
		return
	})
}

// modBeacon allows users to modify beacons through the admin interface
func modBeacon() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		input := struct {
			Id     int
			Label  string
			Uuid   string
			Major  int
			Minor  int
			Option string
		}{}
		dec := json.NewDecoder(req.Body)
		err := dec.Decode(&input)
		if err != nil {
			log.Infof("Failed to decode json request in ModBeacon %s", err)
			http.Error(w, "Invalid Request", 400)
			return
		}

		if len(input.Option) != 3 {
			log.Infof("Option invalid in ModBeacon given \"%s\"", input.Option)
			http.Error(w, "Invalid Request", 400)
			return
		}

		if input.Option != "rem" {
			err = validateLen(nil, input.Label, "Label", 1)
			err = validateLen(err, input.Option, "Option", 3)
			if err != nil {
				log.Infof("Failed validation %s", err)
				http.Error(w, "Invalid Request", 400)
				return
			}
		}
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Errorf("Error opening DB %s", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()
		switch input.Option {
		case "new":
			_, err = db.Exec(`insert into ibeacons
				(label, uuid, major, minor) values
				($1, $2, $3, $4)`, input.Label, input.Uuid,
				input.Major, input.Minor)

		case "mod":
			_, err = db.Exec(`update ibeacons
				set (label, uuid, major, minor) =
				($1, $2, $3, $4) where id = $5`, input.Label, input.Uuid,
				input.Major, input.Minor, input.Id)
		case "rem":
			_, err = db.Exec(`delete from ibeacons
				where id = $1`, input.Id)
		default:
			log.Infof("Option invalid given \"%s\"", input.Option)
			http.Error(w, "Invalid Request", 400)
			return
			// Mod
		}
		if err != nil {
			log.Infof("Failed operation on DB %s", err)
			http.Error(w, "Invalid Request", 400)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"Success": true,
		})
		return
	})
}

// syncCheck returns the maximum time difference between the last 10 beacon_logs
// entered and the current time, this is used to see if any edges are misbehaving
// if no error occurs int will be populated with
func syncCheck() (timedeltaseconds float64, edgenodeid int, err error) {
	dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
	db, err := dbconfig.openDB()
	if err != nil {
		return 0.0, 0, errors.Errorf("Error opening DB %s", err)
	}
	defer db.Close()
	query := `select a.edgenodeid as edge, 
            abs(extract(epoch from a.td)) as diff 
            from (select edgenodeid, datetime - current_timestamp as td 
                from beacon_log order by id desc limit 10) as a 
            order by diff desc limit 1`

	err = db.QueryRow(query).Scan(&edgenodeid, &timedeltaseconds)
	return
}
