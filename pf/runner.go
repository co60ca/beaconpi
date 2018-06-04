package main

import (
	"os"
	"math/rand"
	"math"
	"flag"
	"encoding/json"
	"log"
	"fmt"
)

type Config struct{
	InputFile string
	OutputFile string
	AccelerationSTDD float64
	ImportanceSTDD float64
//	NumParticles int
	WindowSize int
	NoiseInjectionSTDD float64
	RunNumber int
}

var particleSet = []int{200, 400, 800, 1600, 3200, 6400, 12800} 
func PFRunner(cfg Config) {
	data := readCSV(cfg.InputFile)

	rng := rand.New(rand.NewSource(randInt64()))
	for _, numParticles := range particleSet {
		for n := 0; n < cfg.RunNumber; n++ {
			log.Printf("Run %d/%d with %d", n+1, cfg.RunNumber, numParticles)
			msedata := 0.0
			msemodel := 0.0
			edata := 0.0
			emodel := 0.0
			model := NewPFMotionModel(0.1, cfg.AccelerationSTDD,
				cfg.ImportanceSTDD, numParticles)
			for i, _ := range data {
				noise := []float64{0.0, 0.0}
				copy(noise, data[i])
				noise[0] += rng.NormFloat64() * cfg.NoiseInjectionSTDD
				noise[1] += rng.NormFloat64() * cfg.NoiseInjectionSTDD

				estimate := model.SISRound(rng, noise)
				msedata += math.Pow(norm(data[i], noise), 2) / float64(len(data))
				msemodel += math.Pow(norm(data[i], estimate), 2) / float64(len(data))
				edata += norm(data[i], noise) / float64(len(data))
				emodel += norm(data[i], estimate) / float64(len(data))
			}

			outfile, err := os.OpenFile(cfg.OutputFile,
					os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err.Error())
			}
			fmt.Fprintf(outfile, "%s,%f,%f,%d,%f,%f,%f,%f,%f\n",
				cfg.InputFile, cfg.AccelerationSTDD, cfg.ImportanceSTDD,
				numParticles, cfg.NoiseInjectionSTDD, msedata, msemodel,
				edata, emodel)
			outfile.Close()
		}
	}
}

type SlidingWindowModel struct {
	data []float64
	stride int
	// pointer to next empty element
	pointer int
	// Number of elements used
	usage int
	maxusage int
	maxele int
}

func NewSlidingWindowModel(windowsize int, vsize int) (swm *SlidingWindowModel) {
	swm = new(SlidingWindowModel)
	swm.data = make([]float64, windowsize * vsize)
	swm.stride = vsize
	swm.maxusage = windowsize*vsize
	return
}

func (s *SlidingWindowModel) SlidingWindowRound(rng *rand.Rand, measurement []float64) (est []float64) {
	if len(measurement) != s.stride {
		panic("Invalid measurement size")
	}
	copy(s.data[s.pointer:s.pointer+s.stride], measurement)
	s.pointer = (s.pointer + s.stride) % s.maxusage

	if s.usage < s.maxusage {
		s.usage += s.stride
		s.maxele += 1
	}
	est = make([]float64, s.stride)
	for l := 0; l < s.stride; l++ {
		for i := 0; i < s.maxele; i++ {
			est[l] = s.data[i * s.stride + l] 
		}
		est[l] /= float64(s.maxele)
	}
	return
}

func WindowRunner(cfg Config) {
	data := readCSV(cfg.InputFile)

	rng := rand.New(rand.NewSource(randInt64()))
	for n := 0; n < cfg.RunNumber; n++ {
		log.Printf("Run %d/%d", n+1, cfg.RunNumber)
		msedata := 0.0
		msemodel := 0.0
		edata := 0.0
		emodel := 0.0
		model := NewSlidingWindowModel(cfg.WindowSize, 2)
		for i, _ := range data {
			noise := []float64{0.0, 0.0}
			copy(noise, data[i])
			noise[0] += rng.NormFloat64() * cfg.NoiseInjectionSTDD
			noise[1] += rng.NormFloat64() * cfg.NoiseInjectionSTDD

			estimate := model.SlidingWindowRound(rng, noise)
			msedata += math.Pow(norm(data[i], noise), 2) / float64(len(data))
			msemodel += math.Pow(norm(data[i], estimate), 2) / float64(len(data))
			edata += norm(data[i], noise) / float64(len(data))
			emodel += norm(data[i], estimate) / float64(len(data))
		}

		outfile, err := os.OpenFile(cfg.OutputFile,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err.Error())
		}
		fmt.Fprintf(outfile, "%s,%d,%f,%f,%f,%f,%f\n",
			cfg.InputFile, cfg.WindowSize, cfg.NoiseInjectionSTDD, msedata, msemodel,
			edata, emodel)
		outfile.Close()
	}
}

func main() {
	config := flag.String("config", "", "Config file")
	usewindow := flag.Bool("window", false, "Windowing instead of pf")
	flag.Parse()
	if *config == "" {
		flag.Usage()
		return
	}
	f, err := os.Open(*config)
	if err != nil {
		log.Fatal("Failed to open config", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var cfg Config
	if err = dec.Decode(&cfg); err != nil {
		log.Fatal("Failed to decode json", err)
	}
	if *usewindow {
		WindowRunner(cfg)
	} else {
		PFRunner(cfg)
	}
}
