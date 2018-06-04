package main

import (
	"github.com/co60ca/sirpf"
	"math/rand"
)

const (
	SUGGESTED_ACCELERATION_STDD = 1.0
	SUGGESTED_IMPORTANCE_STDD = 1.0
)

// MovementModel returns a movement transition function, it is expected
// that the passed state is a vector of size 4, representing
// [x, y, Δx, Δy]
// all units in m/s (but just keep things consistent)
func MovementModel(stepSeconds, accelStdd float64) func(prev []float64) {
	return func(prev []float64) {
		// Apply velocity to location
		prev[2] += rand.NormFloat64() * accelStdd * stepSeconds
		prev[3] += rand.NormFloat64() * accelStdd * stepSeconds
		prev[0] += prev[2] * stepSeconds
		prev[1] += prev[3] * stepSeconds
		// Perterb acceleration
	}

}

// Imporance model, in the future will incorperate the prior
func ImportanceModel(stdd float64) func(particle, given []float64) (prob float64) {
	fun := sirpf.NewMVGaussian(stdd)
	return func(particle, given []float64) float64 {
		return fun(particle[0:2], given)
	}
}

func NewPFMotionModel(stepSeconds, accelVar, importancestdd float64, particles int) (model *sirpf.State) {
	model = sirpf.NewPF(particles, 4)
	model.Transition = MovementModel(stepSeconds, accelVar)
	model.Observation = ImportanceModel(importancestdd)
	return model
}
