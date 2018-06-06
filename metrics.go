package beaconpi

import (
	"encoding/json"
	"github.com/co60ca/trilateration"
	"github.com/co60ca/webauth"
	"github.com/lib/pq"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"time"
	"database/sql"
)

type MetricsParameters struct {
	Port           string
	DriverName     string
	DataSourceName string
	AllowedOrigin   string
}

type locationResults struct {
	Bracket    time.Time
	Beacon     int
	Loc        []float64
	Edge       []int
	Distance   []float64
	Confidence int
}

type result struct {
	Bracket  time.Time
	Datetime time.Time
	Edge     int
	Rssi     int
}

var mp MetricsParameters

func getDBMForBeacon(beacon int) (int, error) {
	dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
	db, err := dbconfig.openDB()
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var rxpow int
	err = db.QueryRow(`
		select txpower
		from ibeacons
		where id = $1`, beacon).Scan(&rxpow)
	if err != nil {
		return 0, err
	}
	return rxpow, nil
}

// Used for sorting edge/edge location by id
type sortableedge struct {
	edges []int
	locs  [][]float64
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

func beaconTrilateration() http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		decoder := json.NewDecoder(req.Body)
		requestData := struct {
			Edges []int
			// Location of edges in order of Edges member
			// above second dim x,y,z
			EdgeLocations [][]float64
			Beacon        int
			Since         string
			Before        string
			Filter        string
			// Randomized string for keeping history
			Clientid string
			BracketSeconds int
		}{}
		if err := decoder.Decode(&requestData); err != nil {
			log.Infof("Received invalid request %s", err)
			http.Error(w, "Invalid request", 400)
			return
		}
		log.Infof("Request data: %v", requestData)

		sortedges := sortableedge{
			requestData.Edges,
			requestData.EdgeLocations,
		}
		sort.Sort(sortedges)

		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()
		rows, err := db.Query(`
			select to_timestamp(
				floor(extract(epoch from beacon_log.datetime)/$5)*$5)
				as time_bracket,
				datetime, edgenodeid, rssi
				from beacon_log
				where edgenodeid = any ($1::int[])
				and beaconid = $2 and datetime > $3 and datetime < $4
				order by time_bracket, datetime, edgenodeid;
		`, pq.Array(requestData.Edges), requestData.Beacon, requestData.Since,
			requestData.Before, requestData.BracketSeconds)
		if err != nil {
			log.Infof("Error getting query results", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer rows.Close()

		var results []result

		for rows.Next() {
			var row result
			if err = rows.Scan(&row.Bracket, &row.Datetime,
				&row.Edge, &row.Rssi); err != nil {
				log.Infof("Error scanning rows", err)
				http.Error(w, "Server failure", 500)
				return
			}
			results = append(results, row)
		}

		switch requestData.Filter {
		case "average":
			results = filterAverage(results)
		default:
			log.Infof("Received invalid request, unknown filter")
			http.Error(w, "Invalid request", 400)
			return
		}
		log.Debugf("Results into trilat: %#v", results)
		trilatresults := trilat(results, requestData.EdgeLocations, db)
		log.Debugf("Results out of trilat: %#v", trilatresults)
		for i, _ := range trilatresults {
			trilatresults[i].Beacon = requestData.Beacon
		}

		encoder := json.NewEncoder(w)
		if err = encoder.Encode(trilatresults); err != nil {
			log.Infof("Failed to encode results", err)
			http.Error(w, "Server failure", 500)
			return
		}
	})
}

func beaconShortHistory() http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
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
		if (requestData.Before == "") {
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

func trilatCollect(tempresults []result, edgeloc [][]float64, db *sql.DB) locationResults {
	var task trilateration.Parameters3
	copy(task.Loc[0][:], edgeloc[0][:])
	copy(task.Loc[1][:], edgeloc[1][:])
	copy(task.Loc[2][:], edgeloc[2][:])
	var edges []int
	var distances []float64

	for i, r := range tempresults {
		dist, err := distanceModel(r.Rssi, r.Edge, db)
		if err != nil {
			// TODO(mae) don't panic here
			log.Panicf("Failed to get model, possible missing edge %s", err);
		}
		task.Dis[i] = dist
		edges = append(edges, r.Edge)
		distances = append(distances, dist)
	}
	loc, err := task.SolveTrilat3()
	if err != nil {
		log.Panicf("Error occured in Trilat3 %s", err)
	}
	return locationResults{
		Bracket:    tempresults[0].Bracket,
		Loc:        loc,
		Edge:       edges,
		Distance:   distances,
		Confidence: 0,
	}
}

func trilat(results []result, edgeloc [][]float64, db *sql.DB) []locationResults {
	var output []locationResults
	// Sort such that the time brackets are in order and the edge nodes
	// thereafter, this should ensure the edgeloc slice is also in
	// the correct order.
	sort.Slice(results, func(i, j int) bool {
		earlier := results[i].Bracket.Before(results[j].Bracket)
		return earlier || (results[i].Bracket.Equal(results[j].Bracket) && results[i].Edge < results[j].Edge)
	})
	log.Debugf("Results, sorted: %#v", results)
	var tempresults []result
	donext := false
	currtime := results[0].Bracket

	log.Debugf("Results into bracketing loop: %#v", results)

	for i := 0; i < len(results); i++ {
		if results[i].Bracket.Equal(currtime) {
			// Add to current set
			tempresults = append(tempresults, results[i])
		} else {
			donext = true
		}
		currtime = results[i].Bracket
		if donext || i == len(results)-1 {
			donext = false
			// Do trilat for this tempresults set
			tempLocResults := trilatCollect(tempresults, edgeloc, db)
			output = append(output, tempLocResults)
			// New tempresults, for the last loop it wont matter
			tempresults = nil
			// Add the one that was skipped for this round into the next set
			tempresults = append(tempresults, results[i])
		}
	}
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
		code := uint64(tint) | uint64(edgeint)<<32
		if codeint, ok = codetoint[code]; !ok {
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
	mp = *metrics

	mux := http.NewServeMux()

	wa, err := webauth.OpenAuthDB(mp.DriverName, mp.DataSourceName)
	if err != nil {
		log.Fatalf("Failed to open DB for auth %s", err)
	}
	wc := webauth.AuthDBCookie{
		Authdb: wa,
		RedirectLogin: "",
		RedirectHome: "",
	}
	cookieAction := webauth.FAIL_COOKIE_UNAUTHORIZED

	mux.Handle("/auth/login", wc.AuthAndSetCookie())
	mux.Handle("/auth/user", wc.ReturnUserForCookie())
	mux.Handle("/auth/logout", wc.ClearCookie())
	mux.Handle("/auth/allusers", wc.CheckCookie(cookieAction)(wc.GetUsers()))
	mux.Handle("/auth/moduser", wc.CheckCookie(cookieAction)(wc.ModUser()))

	mux.Handle("/config/modbeacon", wc.CheckCookie(cookieAction)(ModBeacon()))
	mux.Handle("/config/modedge", wc.CheckCookie(cookieAction)(ModEdge()))
	mux.Handle("/config/allbeacons", wc.CheckCookie(cookieAction)(GetBeacons()))
	mux.Handle("/config/alledges", wc.CheckCookie(cookieAction)(GetEdges()))
	mux.Handle("/stats/quick", wc.CheckCookie(cookieAction)(quickStats()))

	mux.Handle("/history/short", wc.CheckCookie(cookieAction)(beaconShortHistory()))
	mux.Handle("/history/trilateration", wc.CheckCookie(cookieAction)(beaconTrilateration()))

	mux.Handle("/history/export", wc.CheckCookie(cookieAction)(getCSV()))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{mp.AllowedOrigin},
		AllowCredentials: true,
	})
	handler := c.Handler(mux)

	// Logging
	log.SetLevel(log.DebugLevel)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	// Start
	log.Infof("Starting metrics server on %v", metrics.Port)
	log.Fatal(http.ListenAndServe(":"+metrics.Port, handler))
}
