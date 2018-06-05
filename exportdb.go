package beaconpi

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	//"github.com/pkg/errors"
	"encoding/json"
	"time"
	"github.com/lib/pq"
	"io"
	"io/ioutil"
	"os"
	"fmt"
)

func getCSV() http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		input := struct{
			Edges []int
			Beacons []int
			Before string
			After string
		}{}
		dec := json.NewDecoder(req.Body)
		if err := dec.Decode(&input); err != nil {
			log.Infof("Failed to parse: %s", err)
			http.Error(w, "Invalid Request", 400)
			return
		}

		before, err := time.Parse(time.RFC3339, input.Before)
		var after time.Time 
		if err == nil {
			after, err = time.Parse(time.RFC3339, input.After)
		}
		if err != nil {
			log.Infof("Failed to parse time: %s", err)
			jsonResponse(w, map[string]interface{}{
				"Error": "Failed to parse before or after",
			})
			return
		}
		if after.After(before) {
			log.Infof("Before earlier than after")
			jsonResponse(w, map[string]interface{} {
				"Error": "Before must be after After",
			})
			return
		}

		if len(input.Edges) == 0 || len(input.Beacons) == 0 {
			log.Infof("One of edges or beacons was empty")
			jsonResponse(w, map[string]interface{} {
				"Error": "Both edges and beacons must be non zero length",
			})
			return
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
			select datetime, beaconid, edgenodeid, rssi from beacon_log
			where edgenodeid = any($1::int[]) and beaconid = any($2::int[])
			and datetime < $3 and datetime > $4
		`, pq.Array(input.Edges), pq.Array(input.Beacons), before, after)

		if err != nil {
			log.Infof("Error querying", err)
			http.Error(w, "Server failure", 500)
			return
		}

		defer rows.Close()
		filecsv, err := ioutil.TempFile("", "beaconpi-csv-")
		if err != nil {
			log.Infof("Failure to open file for csv cache", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer func() {
			filecsv.Close()
			os.Remove(filecsv.Name())
		}()

		if _, err = fmt.Fprintf(filecsv, 
				"\"datetime\"\t\"beacon\"\t\"edge\"\t\"rssi\""); err != nil {
			log.Infof("Failure to write to file csv", err)
			http.Error(w, "Server failure", 500)
			return
		}

		for rows.Next() {
			var (
				date time.Time
				beacon int
				edge int
				rssi int
			)
			if err = rows.Scan(&date, &beacon, &edge, &rssi); err != nil {
				log.Infof("Error scanning", err)
        http.Error(w, "Server failure", 500)
        return
			}
			if _, err = fmt.Fprintf(filecsv, 
					"\"%v\"\t%d\t%d\t%d",
					date.Format(time.RFC3339), beacon, edge, rssi); err != nil {
				log.Infof("Failure to write to file csv", err)
				http.Error(w, "Server failure", 500)
				return
			}
		}

		// Seek to the begining of the file and write it to the output
		if _, err = filecsv.Seek(0, os.SEEK_SET); err != nil {
				log.Infof("Failed to seek to pos 0", err)
				http.Error(w, "Server failure", 500)
				return
		}

		var header http.Header
		header.Add("Content-Type", "octet-stream")
		if err = header.Write(w); err != nil {
				log.Infof("Failed to write header", err)
				http.Error(w, "Server failure", 500)
				return
		}
		if _, err = io.Copy(w, filecsv); err != nil {
			log.Infof("Failed to copy data to output", err)
			http.Error(w, "Server failure", 500)
		}
	})
}
