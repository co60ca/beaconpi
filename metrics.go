package beaconpi

import (
	"net/http"
	"log"
	"encoding/json"
	"time"
	"github.com/lib/pq"
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
		log.Println("Received invalid request")
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
	rows, err := db.Query(`
		select datetime, edgenodeid, rssi
		where edgenodeid in ($1) 
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
		Datetime time.Time
		Edge int
		Rssi int
	}

	var results []Result

	for rows.Next() {
		var row Result
		if err = rows.Scan(&row.Datetime, &row.Edge, &row.Rssi); err != nil {
			log.Println("Error scanning rows", err)
			http.Error(w, "Server failure", 500)
			return
		}
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
	http.HandleFunc("/history/short", beaconShortHistory)
	log.Fatal(http.ListenAndServe(":" + metrics.Port, nil))
}
