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
	"encoding/json"
	"github.com/co60ca/webauth"
	"github.com/lib/pq"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

// MetricsParameters represents required data for a metrics server
type MetricsParameters struct {
	Port           string
	DriverName     string
	DataSourceName string
	AllowedOrigin  string
}

var mp MetricsParameters

// beaconShortHistory is used to do the plotting of rssi in the web interface
func beaconShortHistory() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)
		requestData := struct {
			Edges  []int
			Beacon int
			// Datetime formatted with isoUTC datetime
			Since  string
			Before string
		}{}
		if err := decoder.Decode(&requestData); err != nil {
			log.Infof("Received invalid request %s", err)
			http.Error(w, "Invalid request", 400)
			return
		}
		// Backwards compat for tools that didn't use "Before"
		if requestData.Before == "" {
			// Gosh I hope no one is using this in 2050
			requestData.Before = "2050-01-01T01:00:00Z"
		}

		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

		rows, err := db.Query(`
			select datetime, edgenodeid, rssi
			from beacon_log
			where edgenodeid = any($1::int[]) 
			and beaconid = $2 
			and datetime > $3 and datetime < $4
			order by datetime
		`, pq.Array(requestData.Edges), requestData.Beacon, requestData.Since, requestData.Before)
		if err != nil {
			log.Infof("Error getting query results", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer rows.Close()

		type result struct {
			Datetime string
			Edge     int
			Rssi     int
		}

		var results []result

		for rows.Next() {
			var row result
			var date time.Time
			if err = rows.Scan(&date, &row.Edge, &row.Rssi); err != nil {
				log.Infof("Error scanning rows", err)
				http.Error(w, "Server failure", 500)
				return
			}
			row.Datetime = date.Format("2006-01-02T15:04:05")
			results = append(results, row)
		}

		encoder := json.NewEncoder(w)
		if err = encoder.Encode(results); err != nil {
			log.Infof("Failed to encode results", err)
			http.Error(w, "Server failure", 500)
			return
		}
	})
}

// MetricStart is the main entry point of the metrics server
func MetricStart(metrics *MetricsParameters) {
	mp = *metrics

	mux := http.NewServeMux()

	// Logging
	log.SetLevel(log.DebugLevel)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	wa, err := webauth.OpenAuthDB(mp.DriverName, mp.DataSourceName)
	if err != nil {
		log.Fatalf("Failed to open DB for auth %s", err)
	}
	wc := webauth.AuthDBCookie{
		Authdb:        wa,
		RedirectLogin: "",
		RedirectHome:  "",
	}
	cookieAction := webauth.FAIL_COOKIE_UNAUTHORIZED

	mux.Handle("/auth/login", wc.AuthAndSetCookie())
	mux.Handle("/auth/user", wc.ReturnUserForCookie())
	mux.Handle("/auth/logout", wc.ClearCookie())
	mux.Handle("/auth/allusers", wc.CheckCookie(cookieAction)(wc.GetUsers()))
	mux.Handle("/auth/moduser", wc.CheckCookie(cookieAction)(wc.ModUser()))

	mux.Handle("/config/modbeacon", wc.CheckCookie(cookieAction)(modBeacon()))
	mux.Handle("/config/modedge", wc.CheckCookie(cookieAction)(modEdge()))
	mux.Handle("/config/allbeacons", wc.CheckCookie(cookieAction)(getBeacons()))
	mux.Handle("/config/alledges", wc.CheckCookie(cookieAction)(getEdges()))
	mux.Handle("/stats/quick", wc.CheckCookie(cookieAction)(quickStats()))

	mux.Handle("/history/short", wc.CheckCookie(cookieAction)(beaconShortHistory()))
	//TODO(mae) restore cookie
	mux.Handle("/history/maptracking", wc.CheckCookie(cookieAction)(filteredMapLocation(mp)))
	mux.Handle("/maps/allmaps", wc.CheckCookie(cookieAction)(allMaps(mp)))
	mux.Handle("/maps/mapimage", wc.CheckCookie(cookieAction)(fetchImage(mp)))

	mux.Handle("/history/export", wc.CheckCookie(cookieAction)(getCSV()))

	origins := strings.Split(mp.AllowedOrigin, ",")
	log.Infof("Allowed domains: %#v", origins)
	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	// Start
	log.Infof("Starting metrics server on %v", metrics.Port)
	log.Fatal(http.ListenAndServe(":"+metrics.Port, handler))
}
