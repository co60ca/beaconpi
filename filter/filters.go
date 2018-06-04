package filters

import (
	kalman "github.com/ChristopherRabotin/gokalman"
	"gonum.org/v1/gonum/mat"
)

// XY PDR Filter
type XYPDRFilter interface {
	// Update takes in a measurement and applies the filter returning an estimate
	Update(measurement []float64) (estimate []float64)
	// Returns just the measurement
	Estimate() (estimate []float64)
}

type XYPFEKF struct {
//	kf kalman.HybridKF 	
}

// Creates a new XYPFEKF with Δt which is the time of each step and our start
// loc
func NewXYPFEKF(Δt float64, startloc []float64) (out *XYPFEKF) {
	out = new(XYPFEKF)
	prevXHat := mat.NewVecDense(4, nil)
	prevXHat.SetVec(0, startloc[0])
	prevXHat.SetVec(1, startloc[1])

	prevP := mat.NewSymDense(4, nil)

	// Values taken from Iannis paper
	errorcovar := 100.0
	for i := 0; i < 4; i++ {
		prevP.SetSym(i, i, errorcovar)
	}
//	Q := mat.NewSymDense(4, nil)
//	R := mat.NewSymDense(2, []float64{0.1, 0.0, 0.0, 0.1})
//	noiseKF := kalman.NewNoiseless(Q, R)
//	out.kf, _, err := NewHybridKF(prevXHat, prevP, noiseKF, 2)
//	out.kf.EnableEKF()
	return out
}
