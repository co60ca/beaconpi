package beaconpi

import (
	"math"
	"sort"
	"net/http"
	"log"
	"encoding/json"
	"time"
	"github.com/lib/pq"
	"github.com/co60ca/trilateration"
)

const (
	SIGNAL_PROP_CONSTANT = 2
)

type MetricsParameters struct {
	Port string	
	DriverName string
	DataSourceName string
}

type locationResults struct {
  Bracket time.Time
	Loc []float64
  Edge []int
  Distance []float64
  Confidence int
}

type result struct {
  Bracket time.Time
  Datetime time.Time
  Edge int
  Rssi int
}

var mp MetricsParameters

func getDBMForBeacon(beacon int) (int, error) {
	dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
	db, err := dbconfig.openDB()
	if err != nil {
		return 0, err
	}
	var rxpow int
	err = db.QueryRow(`
		select txpower
		from ibeacons
		where id = ?`, beacon).Scan(&rxpow)
	if err != nil {
		return 0, err
	}
	return rxpow, nil
}

// Used for sorting edge/edge location by id
type sortableedge struct {
	edges []int
	locs [][]float64
}
func (s sortableedge) Len() int {
	return len(s.edges)
}
func (s sortableedge) Less(i, j int) bool {
	return s.edges[i] < s.edges[j]
}
func (s sortableedge) Swap(i, j int) {
	// Swap edges
	s.edges[i], s.edges[j] = s.edges[j], s.edges[i]
	// Swap locs
	s.locs[i], s.locs[j] = s.locs[j], s.locs[i]
}


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

	sortedges := sortableedge{
		requestData.Edges,
		requestData.EdgeLocations,
	}
	sort.Sort(sortedges)

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

  switch requestData.Filter {
    case "average":
      results = filterAverage(results)
    default:
      log.Println("Received invalid request, unknown filter")
      http.Error(w, "Invalid request", 400)
      return
  }

	// Resort incase filtering has changed sorting order

	power, err := getDBMForBeacon(requestData.Beacon)
	if err != nil {
		log.Println("Error fetching beacon power by id", err)
		http.Error(w, "Server failure", 500)
		return
	}

  trilatresults := trilat(results, requestData.EdgeLocations, power)

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

func trilatCollect(tempresults []result, edgeloc [][]float64, dbm int) locationResults {
	var task trilateration.Parameters3 
	copy(task.Loc[0][:], edgeloc[0][:])
	copy(task.Loc[1][:], edgeloc[1][:])
	copy(task.Loc[2][:], edgeloc[2][:])
	var edges []int
	var distances []float64

	for i, r := range tempresults {
		dist := math.Pow(10, float64(dbm - r.Rssi) / float64(10 * SIGNAL_PROP_CONSTANT))
		task.Dis[i] = dist
		edges = append(edges, r.Edge)
		distances = append(distances, dist)
	}
	loc, conf := task.SolveTrilat3()
	return locationResults{
		Bracket: tempresults[0].Bracket,
		Loc: loc,
		Edge: edges,
		Distance: distances,
		Confidence: conf,
	}
}

func trilat(results []result, edgeloc [][]float64, dbm int) []locationResults {
	var output []locationResults
	// Sort such that the time brackets are in order and the edge nodes
	// thereafter, this should ensure the edgeloc slice is also in
	// the correct order.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Bracket.Before(results[j].Bracket) && results[i].Edge < results[j].Edge
	})
	var tempresults []result
	donext := false
	currtime := results[0].Bracket
	for i := 0; i < len(results); i++ {
		if results[i].Bracket.Equal(currtime) {
			// Add to current set
			tempresults = append(tempresults, results[i])
		} else {
			donext = true
		}
		if donext || i == len(results) - 1 {
			// Do trilat for this tempresults set
			tempLocResults := trilatCollect(tempresults, edgeloc, dbm)
			output = append(output, tempLocResults)
			// New tempresults, for the last loop it wont matter
			tempresults = nil
			// Add the one that was skipped for this round into the next set
			tempresults = append(tempresults, results[i])
		}
	}
	log.Printf("results %#v\n", results)
	// TODO(mae)
	return output
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
