package beaconpi

import (
	"net/http"
	"log"
	"encoding/json"
	"time"
	"github.com/lib/pq"
	"github.com/co60ca/trilateration"
)

type MetricsParameters struct {
	Port string	
	DriverName string
	DataSourceName string
}

type locationResults struct {
  Bracket time.Time
  Beacon int
  Edge []int
  Distance []int
  Confidence int
}

type result struct {
  Bracket time.Time
  Datetime time.Time
  Edge int
  Rssi int
}

var mp MetricsParameters


func beaconTrilateration(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	requestData := struct {
		Edges []int
    // Location of edges in order of Edges member above second dim x,y,z
    EdgeLocations [][]float64
		Beacon int
		Since string
		Before string
    Filter string
    // Randomized string for keeping history
    Clientid string
    // TODO(mae) use this
    BracketSeconds int
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
	rows, err := db.Query(`
		select date_trunc('second', min(datetime)) as time_bracket, 
    datetime, edgenodeid, rssi
		where edgenodeid in ($1) 
		and beaconid = $2 and datetime > $3 and datetime < $4
    group by floor(extract(epoch from beacon_log.datetime)/ 1), datetime, beaconid, edgenodeid, rssi
		order by time_bracket, datetime, edgenodeid
	`, pq.Array(requestData.Edges), requestData.Beacon, requestData.Since,
  requestData.Before)
	if err != nil {
		log.Println("Error getting query results", err)
		http.Error(w, "Server failure", 500)
		return
	}
	defer rows.Close()

	var results []result

	for rows.Next() {
		var row result
		if err = rows.Scan(&row.Bracket, &row.Datetime, &row.Edge, &row.Rssi); err != nil {
			log.Println("Error scanning rows", err)
			http.Error(w, "Server failure", 500)
			return
		}
		results = append(results, row)
	}

  // TODO(mae) filter
  switch requestData.Filter {
    case "average":
      results = filterAverage(results)
    default:
      log.Println("Received invalid request, unknown filter")
      http.Error(w, "Invalid request", 400)
      return
  }
	//TODO(mae) dbm
	// TODO(mae) replace 0x0 with dbm
  trilatresults := trilat(results, requestData.EdgeLocations, 0x0)

	encoder := json.NewEncoder(w)
	if err = encoder.Encode(trilatresults); err != nil{
		log.Println("Failed to encode results", err)
		http.Error(w, "Server failure", 500)
		return
	}
}

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

	type result struct {
		Datetime time.Time
		Edge int
		Rssi int
	}

	var results []result

	for rows.Next() {
		var row result
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

// Trilateration assumes that the
func trilat(results []result, edgeloc [][]float64, dbm int) []locationResults {
	// TODO(mae)
	return nil
}

func filterAverage(results []result) []result {
	var out []result
	var counts []int
	codetoint := make(map[uint64]int)
	timetoint := make(map[time.Time]uint32)
	edgetoint := make(map[int]uint32)
	for _, r := range results {
		var ok bool
		var tint uint32
		var edgeint uint32
		var codeint int
		if tint, ok = timetoint[r.Bracket]; !ok {
			timetoint[r.Bracket] = uint32(len(timetoint))
			tint = uint32(len(timetoint) - 1)
		}
		if edgeint, ok = edgetoint[r.Edge]; !ok {
			edgetoint[r.Edge] = uint32(len(edgetoint))
			edgeint = uint32(len(edgetoint) - 1)
		}
		code := uint64(tint) | uint64(edgeint) << 32
		if codeint, ok = codetoint[code] ; !ok {
			codetoint[code] = len(codetoint)
			codeint = codetoint[code]
			// Make a new entry in out
			out = append(out, r)
			counts = append(counts, 1)
			out[codeint].Datetime = r.Bracket
		} else {
			// Just add the RSSI value
			out[codeint].Rssi += r.Rssi
			counts[codeint] += 1
		}
	}
	// compute mean
	for i, _ := range out {
		out[i].Rssi /= counts[i]
	}
	return out
}

func MetricStart(metrics *MetricsParameters) {
	http.HandleFunc("/history/short", beaconShortHistory)
	log.Fatal(http.ListenAndServe(":" + metrics.Port, nil))
}
