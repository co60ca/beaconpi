package beaconpi

import (
	"fmt"
	"encoding/hex"
	"encoding/binary"
	"bytes"
	"io"
	"encoding/json"
	"time"
	"errors"
	"math/rand"
	crand "crypto/rand"
	"gonum.org/v1/gonum/stat/distuv"
	"math"
	log "github.com/sirupsen/logrus"
)
// This module is designed to be able to mock hcidump with realistic
// looking data

const (
	MODE_MODEL = iota
	MODE_RANDOM
)

type MockConfig struct {
	Beacons []MockBeacon
	Edges []MockEdge
	// For simplicity I provide all Edges data but give a selector, this
	// lets multiple running producers share the same config file
	EdgeSelected int
	PathLossModel PathmodelParams
	// Noise added to regular operation at RSSI, or distance for MODE_MODEL
	StdDevNoise float64
	// Events added that change the RSSI dramatically Poisson mean in seconds
	RandomEventK float64
	// Event offset sample StdDev
	RandomEventStdDev float64
	Mode int
}

type MockBeacon struct {
	BeaconRecord
	// In ms
	Delay float64
	Location []float64
}

type MockEdge struct {
	Location []float64
}

func ReadMockConfiguration(r io.Reader) (*MockConfig, error) {
	// We need to read it twice
	r1 := new(bytes.Buffer)
	r2 := new(bytes.Buffer)
	wmulti := io.MultiWriter(r1, r2)
	if _, err := io.Copy(wmulti, r); err != nil {
		return nil, err
	}

	var output MockConfig
	decoder := json.NewDecoder(r1)
	err := decoder.Decode(&output)
	if err != nil {
		return nil, errors.New("Failed to read configuration " + err.Error())
	}
	// Because Go cannot unmarshal into array (or []byte) we need to decode
	// the uuid has base64 hex strings
	var uuidstruct struct {
		Beacons []struct {
			Uuid string
		}
	}
	decoder = json.NewDecoder(r2)
	
	if err := decoder.Decode(&uuidstruct) ; err != nil {
		return nil, errors.New("Failed to read configuration UUID decode " + err.Error())
	}
	for i, v := range uuidstruct.Beacons {
		data, err := hex.DecodeString(v.Uuid)
		if err != nil {
			return nil, errors.New("Failed to hex decode string " + v.Uuid)
		}
		copy(output.Beacons[i].Uuid[:], data)
	}
	
	return &output, nil
}

func euclideanDist(v1, v2 []float64) float64 {
	if len(v1) != len(v2) {
		panic("|v1| != |v2|")
	}
	sum := 0.0
	for i, _ := range v1 {
		sum += math.Pow(v1[i] - v2[i], 2)
	}
	return math.Sqrt(sum)
}

func (conf *MockConfig) generateRecord(rng *rand.Rand, beacon, edge int, eventbias float64) *BeaconRecord {
	out := &BeaconRecord{}
	copy(out.Uuid[:], conf.Beacons[beacon].Uuid[:])
	out.Major, out.Minor = conf.Beacons[beacon].Major, conf.Beacons[beacon].Minor
	switch conf.Mode { 
		case MODE_RANDOM:
			// Random data around -50dbm
			out.Rssi = int16(-50 + rng.NormFloat64() * conf.StdDevNoise)
		case MODE_MODEL:
			dist := euclideanDist(conf.Beacons[beacon].Location, conf.Edges[edge].Location)
			log.Infof("Actual distance: %v\n", dist)
			dist += eventbias 
			// TODO(mae) redo this out.Rssi = int16(pathLossExpectedRssi(dist, conf.PathLossModel.Bias, conf.PathLossModel.Gamma) + rng.NormFloat64() * conf.StdDevNoise)
			log.Infof("Testing RSSI: %v\n", out.Rssi)
	} 
	out.Datetime = time.Now()
	return out
}

func eventNextTime(k float64) time.Duration {
	// Uses its own rng
	dist := distuv.Poisson{Lambda: k}
	return time.Duration(int64(float64(time.Second)*dist.Rand()))
}

func (conf *MockConfig) generateBeaconRecords(out chan *BeaconRecord) {
	// Setup RNG
	randbyte := make([]byte, 4)
	_, err := crand.Read(randbyte)
	if err != nil {
		panic("Getting random seed from system failed: " + err.Error())
	}
	seed, _ := binary.Varint(randbyte)
	rng := rand.New(rand.NewSource(seed))
	var timers []*time.Ticker
	for _, v := range conf.Beacons {
		timers = append(timers, time.NewTicker(time.Duration(int64(float64(time.Millisecond) * v.Delay))))
	}

	// Event
	eventtimer := time.After(eventNextTime(conf.RandomEventK))
	var eventendtimer <- chan time.Time
	var offsetevent float64

	for {
		time.Sleep(time.Millisecond * 10)

		// Check for event changes
		select {
		case <- eventtimer:
			// Event end timer
			seconds := float64(time.Second)*(rng.NormFloat64() * 3 + 10)
			// Minimum 3 seconds
			seconds = math.Max(seconds, float64(time.Second) * 3)

			eventendtimer = time.After(time.Duration(int64(seconds)))

			// Offset calculation, always +tv
			offsetevent = rng.NormFloat64()*conf.RandomEventStdDev 
			offsetevent = math.Abs(offsetevent)
			log.Infof("Event started %vs, offset: %v\n", time.Duration(int64(seconds)).Seconds(), offsetevent)
		case <- eventendtimer:
			fmt.Println("Event ended")
			eventtimer = time.After(eventNextTime(conf.RandomEventK))
			offsetevent = 0.0
		default:
		}

		for i, v := range timers {
			select {
				case <- v.C:
					out <- conf.generateRecord(rng, i, conf.EdgeSelected, offsetevent)
				default:
			}
		}
	}
}

func (conf *MockConfig) HCIDump() {	
	generator := make(chan *BeaconRecord, 128)
	go conf.generateBeaconRecords(generator)
	for {
		br := <- generator 
		fmt.Println(br.String())
	}
}

func (b *BeaconRecord) Bytes() (o []byte) {
	// I don't really care about this fix data so I just grabbed some online
	// for testing purposes
	fixed := "\x04\x3E\x2A\x02\x01\x03\x00\xB3\xF1\xC6\x72\x02\x00\x1E\x02\x01\x1A\x1A\xFF\x4C\x00\x02\x15"
	buf := new(bytes.Buffer)
	o = make([]byte, 46)
	// 0-22
	copy(o[:23], []byte(fixed))
	// 23-39 uuid
	copy(o[23:40], b.Uuid[:])
	// 40-41 major
	if err := binary.Write(buf, binary.BigEndian, b.Major); err != nil {
		panic(err)
	}
	copy(o[40:42], buf.Bytes()[:2])
	buf.Reset()
	// 42-43 minor
	if err := binary.Write(buf, binary.BigEndian, b.Minor); err != nil {
		panic(err)
	}
	copy(o[42:44], buf.Bytes()[:2])
	buf.Reset()
	// 44 tx power
	// 45 rx power (rssi)
	// Warning: b.Rssi is two bytes so we use byte 1
	if err := binary.Write(buf, binary.BigEndian, b.Rssi); err != nil {
		panic(err)
	}
	o[45] = buf.Bytes()[1]
	return
}

// Prints the beacon record the same way hcidump does
func (b *BeaconRecord) String() (o string) {

	// every twentyith byte is followed by new line and two spaces
	o = ">"
	for i, v := range(b.Bytes()) {
		if i != 0 && (i % 20) == 0 {
			o += "\n  "
		} else {
			o += " "
		}
		o += fmt.Sprintf("%02X", v) 
	}
	return
}

