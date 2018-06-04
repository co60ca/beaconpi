package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"os"
	crand "crypto/rand"
	"encoding/binary"
)

func TestPFOne(t *testing.T) {
	av := 0.7
	im := 100.0
	gv := 2400
	model := NewPFMotionModel(0.1, SUGGESTED_ACCELERATION_VAR * av,
		SUGGESTED_IMPORTANCE_VAR * im, gv)
	// model is all zeros, which is what the prior is anyway
	data := ReadCSV("out_long.csv")
	outfile, err := os.Create("filtered.csv")
	if err != nil {
		panic(err.Error())
	}
	defer outfile.Close()
	rng := rand.New(rand.NewSource(0))
	msedata := 0.0
	msemodel := 0.0
	edata := 0.0
	emodel := 0.0
	fmt.Fprintf(outfile,"realx,realy,noisex,noisey,estx,esty\n")
	for i, _ := range data[0:1200] {
		noise := []float64{0.0, 0.0}
		copy(noise, data[i])
		noise[0] += rng.NormFloat64() * 9
		noise[1] += rng.NormFloat64() * 9

		estimate := model.SISRound(rng, noise)
		msedata += math.Pow(norm(data[i], noise), 2) / float64(len(data))
		msemodel += math.Pow(norm(data[i], estimate), 2) / float64(len(data))
		edata += norm(data[i], noise) / float64(len(data))
		emodel += norm(data[i], estimate) / float64(len(data))
	  t.Logf("\nInput %v,\nNoised %v,\nOutput %v", data[i], noise, estimate)
		fmt.Fprintf(outfile, "%f, %f, %f, %f, %f, %f\n", data[i][0], data[i][1], noise[0], noise[1], estimate[0], estimate[1])
	}
	t.Logf("MSE noise: %f, MSE model: %f", msedata, msemodel)
	t.Logf("E noise: %f, E model: %f", edata, emodel)
}
