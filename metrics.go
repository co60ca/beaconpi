package beaconpi

import (
	"net/http"
	"log"
	"encoding/json"
	"time"
	_ "github.com/lib/pq"
	"github.com/lib/pq"
	"github.com/rs/cors"
)

type MetricsParameters struct {
	Port string	
	DriverName string
	DataSourceName string
}

var mp MetricsParameters

func beaconShortHistory(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	requestData := struct {
		Edges []int
		Beacon int
		Since string
	}{}
	if err := decoder.Decode(&requestData); err != nil {
		log.Println("Received invalid request", err)
		http.Error(w, "Invalid request", 400)
		return
	}
	dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
	db, err := dbconfig.openDB()
	if err != nil {
		log.Println("Error opening DB", err)
		http.Error(w, "Server failure", 500)
		return
	}
	defer db.Close()
	rows, err := db.Query(`
		select datetime, edgenodeid, rssi
		from beacon_log
		where edgenodeid = any($1::int[]) 
		and beaconid = $2 
		and datetime > $3
		order by datetime
	`, pq.Array(requestData.Edges), requestData.Beacon, requestData.Since)
	if err != nil {
		log.Println("Error getting query results", err)
		http.Error(w, "Server failure", 500)
		return
	}
	defer rows.Close()

	type Result struct {
		Datetime string
		Edge int
		Rssi int
	}

	var results []Result

	for rows.Next() {
		var row Result
		var date time.Time
		if err = rows.Scan(&date, &row.Edge, &row.Rssi); err != nil {
			log.Println("Error scanning rows", err)
			http.Error(w, "Server failure", 500)
			return
		}
		row.Datetime = date.Format("2006-01-02T15:04:05")
		results = append(results, row)
	}

	encoder := json.NewEncoder(w)
	if err = encoder.Encode(results); err != nil{
		log.Println("Failed to encode results", err)
		http.Error(w, "Server failure", 500)
		return
	}
}

func MetricStart(metrics *MetricsParameters) {
	mp = *metrics
	mux := http.NewServeMux()
	mux.HandleFunc("/history/short", beaconShortHistory)

	handler := cors.Default().Handler(mux)
	log.Fatal(http.ListenAndServe(":" + metrics.Port, handler))
}
