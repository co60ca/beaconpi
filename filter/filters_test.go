package filters

import (
	"testing"
)

func TestNewXYPFEKF(t *testing.T) {
	_ = NewXYPFEKF(0.0, []float64{1, 1})
}
