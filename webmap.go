package beaconpi

import (
    "net/http"
    "encoding/json"
    log "github.com/sirupsen/logrus"
    "github.com/pkg/errors"
    "time"
    "bytes"
    "database/sql"
    "github.com/co60ca/indoorfilters"
    "github.com/co60ca/trilateration"
    "sync"
    "github.com/lib/pq"
    "io"
)

type MapConfig struct {
    Id int
    Title string
    // Image stored seperatly
    Image int
    // Json part below

    CoordBiasX int
    CoordBiasY int
    // Can be negative to flip coordinate system
    CoordScaleX float64
    CoordScaleY float64
    // x1, x2, y1, y2
    Limits []float64
    Edges []int
}

type TrackingData struct {
    FilterID string
    RequestTime time.Time
    // Beacon IDs
    Beacons []int
    Edges []int
    Series []TimeSeriesPoint
    MapConfig *MapConfig
}

type TimeSeriesPoint struct {
    Beacon int
    Time time.Time
    // 2d location
    Location []float64
}

type FilteredMapLocationRequest struct {
      // Previously assigned filter ID
      FilterID string
      Beacons []int
      Edges []int
      MapID int
      RequestTime time.Time
      Algorithm string
}

type filterIdSet struct {
  pfs map[int]*indoorfilters.PF
  timeout time.Time
}

type filterManager struct {
    sync.Mutex
    filters map[string]*filterIdSet
}

func fetchLO(db *sql.DB, i int) ([]byte, error) {
  var res []byte
  if err := db.QueryRow(`
    select string_agg(a.data, '') 
    from (select data 
        from pg_largeobject 
        where loid = $1 
        order by pageno) as a`, i).Scan(&res); err != nil {
      return nil, err
    }
  return res, nil
}

func fetchImage(mp MetricsParameters) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

    var request struct {
      ImageID int
    }
    dec := json.NewDecoder(req.Body)
    if err := dec.Decode(&request); err != nil {
			log.Infof("Failed to parse request", err)
			http.Error(w, "Invalid Request", 400)
			return
    }

    var image int

		err = db.QueryRow(`
			select image 
			from webmap_configs
      where id = $1`, request.ImageID).Scan(&image)
		if err != nil {
				log.Infof("Failed while quering configs %s", err)
				http.Error(w, "Server failure", 500)
				return
		}
    data, err := fetchLO(db, image)
    if err != nil {
				log.Infof("Failed while fetching the image %s", err)
				http.Error(w, "Server failure", 500)
				return
    }
    //TODO(mae) more mimetypes?
    w.Header().Set("Content-Type", "image/png")
    buf := bytes.NewBuffer(data)
    if _, err = io.Copy(w, buf); err != nil {
      log.Infof("Failed to copy buffer %s", err)
      http.Error(w, "Server failure", 500)
      return
    }
    return
  })
}

func allMaps(mp MetricsParameters) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
		defer db.Close()

		rows, err := db.Query(`
			select id, title, image, config 
			from webmap_configs
			order by id`)
		if err != nil {
				log.Infof("Failed while quering configs %s", err)
				http.Error(w, "Server failure", 500)
				return
		}

    var configs []MapConfig
    for rows.Next() {
      var (
        id int
        title string
        mapid int
        config string
      )
      if err = rows.Scan(&id, &title, &mapid, &config); err != nil {
				log.Infof("Failed while quering configs %s", err)
				http.Error(w, "Server failure", 500)
				return
      }

      var res MapConfig
      buf := bytes.NewBufferString(config)
      dec := json.NewDecoder(buf)
      if err := dec.Decode(&res); err != nil {
				log.Infof("Failed while scanning configs %s", err)
				http.Error(w, "Server failure", 500)
				return
      }
      res.Id, res.Title = id, title
      configs = append(configs, res)
    }
		jsonResponse(w, map[string]interface{}{
			"Maps": configs,
		})
    return
  })
}

type filterFunction func(*sql.DB, *MapConfig, *FilteredMapLocationRequest) (TrackingData, error)

func filteredMapLocation(mp MetricsParameters) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, req *http.Request) {
    var request FilteredMapLocationRequest
    dec := json.NewDecoder(req.Body)
    if err := dec.Decode(&request); err != nil {
			log.Infof("Failed to parse request", err)
			http.Error(w, "Invalid Request", 400)
			return
    }

		dbconfig := dbHandler{mp.DriverName, mp.DataSourceName}
		db, err := dbconfig.openDB()
		if err != nil {
			log.Infof("Error opening DB", err)
			http.Error(w, "Server failure", 500)
			return
		}
    //create table webmap_configs (
		defer db.Close()
    var mc *MapConfig
    if mc, err = fetchMapConfig(db, request.MapID); err != nil {
        log.Infof("Failed to fetch map for given Id", err)
        http.Error(w, "Invalid Request", 400)
        return
    }
    var algo filterFunction
    switch request.Algorithm {
    case "particle-filter-velocity":
        algo = particleFilterVelocity
    }
    td, err := algo(db, mc, &request)
    if err != nil {
			log.Infof("Error applying filter", err)
			http.Error(w, "Server failure", 500)
			return
    }
    enc := json.NewEncoder(w)
    if err = enc.Encode(td); err != nil {
			log.Infof("Error encoding results", err)
			http.Error(w, "Server failure", 500)
			return
    }
  })
}

var clampedPFs filterManager
func particleFilterVelocity(db *sql.DB, mp *MapConfig,
    mlr *FilteredMapLocationRequest) (TrackingData, error) {
    var res TrackingData
    if clampedPFs.filters == nil {
      clampedPFs.filters = make(map[string]*filterIdSet)
    }
    clampedPFs.Lock()
    defer clampedPFs.Unlock()

    rng := getRand()

    // Initalize filters
    for {
      if _, ok := clampedPFs.filters[mlr.FilterID]; ok {
        break
      }
      // Filter not set, make new
      mlr.FilterID = RandBase64(rng, 6)

      // Already exists check
      if _, ok := clampedPFs.filters[mlr.FilterID]; ok {
        continue
      }
      // Create a new set
      clampedPFs.filters[mlr.FilterID] = &filterIdSet{timeout: time.Now().Add(time.Second * 30)}
      clampedPFs.filters[mlr.FilterID].pfs = make(map[int]*indoorfilters.PF)
      for _, v := range mlr.Beacons {
        clampedPFs.filters[mlr.FilterID].pfs[v] = indoorfilters.NewClampedFilter(
            mp.Limits[0], mp.Limits[1], mp.Limits[2], mp.Limits[3],
            200, 0.5, 0.01, 5.0)
      }
    }

    curfilter := clampedPFs.filters[mlr.FilterID]
    // Advance timeout
    curfilter.timeout = time.Now().Add(time.Second * 30)

    // Fetch required locations and edge
    edgeloc, err := fetchEdgeLocations(db, mlr.Edges)
    if err != nil {
      return TrackingData{}, errors.Wrap(err, "Failed to fetch edges")
    }
    log.Infof("mlr.RequestTime %s", mlr.RequestTime)
    rssi, err := fetchAverageRSSI(db, mlr.Beacons, mlr.Edges, mlr.RequestTime)
    log.Infof("Rssi %#v", rssi)
    if err != nil {
      return TrackingData{}, errors.Wrap(err, "Failed to fetch RSSI")
    }

    series, err := trilatMultiBeacons(rssi, edgeloc, mlr.Beacons, mlr.Edges, mlr.RequestTime)
    if err != nil {
      return TrackingData{}, errors.Wrap(err, "Failed in trilat")
    }
    log.Infof("Series %#v", series)
    series, err = filterClampPFsApply(series, curfilter)
    if err != nil {
      return TrackingData{}, errors.Wrap(err, "Failed in filter")
    }
    res.Series = series
    res.FilterID = mlr.FilterID
    res.Beacons = mlr.Beacons
    res.Edges = mlr.Edges
    res.RequestTime = mlr.RequestTime
    res.MapConfig = mp

    return res, nil
}

func filterClampPFsApply(series []TimeSeriesPoint, filters *filterIdSet) ([]TimeSeriesPoint, error) {
  for i, _ := range series {
    b := series[i].Beacon
    pf, ok := filters.pfs[b]
    if !ok {
      return nil, errors.New("Beacon does not exist in list of particle filters")
    }
    x, y, err := pf.Round(series[i].Location[0], series[i].Location[1]); if err != nil {
      return nil, errors.Wrap(err, "Failed during particle filter run")
    }
    series[i].Location[0], series[i].Location[1] = x, y
  }
  return series, nil
}

type rssiTuples struct {
  Beacon int
  Edge int
  // Rssi is an average here so we use float64
  Rssi float64
  // Dist in metres
  Dist float64
}

// edges matches loc for id of edge
// rssi must be ordered by beacon, edge (as per the results of fetchAverageRSSI
func trilatMultiBeacons(rssi []rssiTuples, loc [][]float64, beacons []int,
                        edges []int, time time.Time) (series []TimeSeriesPoint, err error) {
  edgeToIdx := make(map[int]int)
  for i, v := range edges {
      edgeToIdx[v] = i
  }

  bi := 0
  b := beacons[bi]

  var tloc []trilateration.Point3
  var tdist []float64

  for _, v := range rssi {
    if v.Beacon != b {
      bi += 1
      if bi < len(beacons) {
        // Do trilateration and reset
        log.Infof("tloc %v, tdis %v", tloc, tdis)
        params := trilateration.Parameters3{Loc: tloc, Dis: tdist}
        trilatloc, err := params.SolveTrilat3()
        if err != nil {
          return nil, errors.Wrapf(err, "Failed to solve trilateration with Loc %v and Dist %v", tloc, tdist)
        }
        series = append(series, TimeSeriesPoint{
          Beacon: b,
          Time: time,
          Location: trilatloc,
        })
        // Reset
        b = beacons[bi]
        tloc = nil
        tdist = nil
      } else {
        break
      }
    }
    // Append record from rssi
    e := v.Edge
    tdist = append(tdist, v.Dist)
    var p3 trilateration.Point3
    // TODO(mae)  is this element 0 and 1
    copy(p3[0:3], loc[edgeToIdx[e]][0:3])
    tloc = append(tloc, p3)
  }

  // Finally do the last trilat
  params := trilateration.Parameters3{Loc: tloc, Dis: tdist}
  trilatloc, err := params.SolveTrilat3()
  if err != nil {
    return nil, errors.Wrapf(err, "Failed to solve trilateration with Loc %v and Dist %v", tloc, tdist)
  }
  series = append(series, TimeSeriesPoint{
    Beacon: b,
    Time: time,
    Location: trilatloc,
  })

  return
}

// Returns the average RSSI ordered by Beacon, Edge
func fetchAverageRSSI(db *sql.DB, beacons []int, edges []int,
    ts time.Time) ([]rssiTuples, error) {
    rows, err := db.Query(`select beacon, edge, rssi, distance
        from average_stamp_and_prev($1) 
        where beacon = any ($2::int[])
        and edge = any ($3::int[])
    `, ts, pq.Array(beacons), pq.Array(edges))
    if err != nil {
        return nil, errors.Wrap(err, "Failed to fetch RSSI with query")
    }
    var result []rssiTuples
    for rows.Next() {
      var t rssiTuples
      if err = rows.Scan(&t.Beacon, &t.Edge, &t.Rssi, &t.Dist); err != nil {
        return nil, errors.Wrap(err, "Failed to fetch RSSI when scanning")
      }
      result = append(result, t)
    }
    return result, nil
}

func fetchEdgeLocations(db *sql.DB, edges []int) (loc [][]float64, err error) {
    rows, err := db.Query(`select x, y, z
        from edge_locations
        where id = any ($1::int[])
        order by id
    `, pq.Array(edges))

    if err != nil {
        return nil, errors.Wrap(err, "Failed to fetch Edges with query")
    }
    for rows.Next() {
      t := make([]float64, 3)
      if err = rows.Scan(&t[0], &t[1], &t[2]); err != nil {
        return nil, errors.Wrap(err, "Failed to fetch Edges when scanning")
      }
      loc = append(loc, t)
    }
    if len(loc) != len(edges) {
      return nil, errors.New("Failed to get enough edges given input")
    }
    return
}

func fetchMapConfig(db *sql.DB, id int) (*MapConfig, error) {
  var (
    title string
    image int
    config string
  )
  var res MapConfig
  if err := db.QueryRow(`select title, image, config 
      from webmap_configs
      where id = $1`, id).Scan(&title, &image, &config); err != nil {
        return nil, errors.Wrapf(err, "Failed to query config with id = %d", id)
  }
  buf := bytes.NewBufferString(config)
  dec := json.NewDecoder(buf)
  if err := dec.Decode(&res); err != nil {
    return nil, err
  }
  res.Id, res.Title, res.Image = id, title, image
  return &res, nil
}
