package beaconpi

import (
	"math"
	"database/sql"
	"sync"
	"errors"
	log "github.com/sirupsen/logrus"
)

func pathLossFunction(bias, gamma float64) func(float64) float64 {
	return func(rssi float64) float64 {
		log.Info(bias, gamma, rssi)
		return math.Pow(10, (bias - rssi)/ (10 * gamma))
	}
}

func PathLoss(rssi, bias, gamma float64) float64 {
	return pathLossFunction(bias, gamma)(rssi)
}

type PathmodelParams struct {
	Bias float64
	Gamma float64
}

func getModelByEdge(edge int, db*sql.DB) (PathmodelParams, error) {
	var params PathmodelParams
	err := db.QueryRow(`
		select bias, gamma
		from models
		where id = $1 
	`, edge).Scan(&params.Bias, &params.Gamma)
	if err != nil {
		return params, err
	}
	return params, nil
}

var modelcache map[int]PathmodelParams
var modellock sync.Mutex

func distanceModel(rssi, edge int, db *sql.DB) (float64, error) {
	v, err := getModelByEdge(edge, db)
	if err != nil {
		return 0.0, errors.New("Failed to get model from DB " +
			err.Error())
	}
	return PathLoss(float64(rssi), v.Bias, v.Gamma), nil
}
