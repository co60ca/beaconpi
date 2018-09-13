package beaconpi

import (
	"bytes"
	"testing"
)

func TestBytes(t *testing.T) {
	var testbecon BeaconRecord
	testbecon.Major = 1
	testbecon.Minor = 2
	testbecon.Rssi = -70
	t.Logf("%#v", testbecon.Bytes())
	t.Logf("%#v", testbecon.String())
}

func TestOpenConfig(t *testing.T) {
	testconf := bytes.NewBufferString(`
    {"Beacons" :
      [
        {
          "Uuid": "52d8064e08ca474fb471e3610e609141",
          "Major": 1,
          "Minor": 1,
          "Delay": 100,
          "Location": [0.1, 0.2, 0.3]
        }
      ],
    "Edges" :
      [
        {
          "Location": [0.3, 0.3, 0.3]
        }
      ],
    "EdgeSelected": 0,
    "PathLossModel":
      {
        "Bias": -30.0564,
        "K": 9.73594e-05,
        "Gamma": 0.536281
      },
    "StdDevNoise": 10,
    "RandomEventK": 50,
    "RandomEventStdDev": 10,
    "Mode": 0
    }`)
	conf, err := ReadMockConfiguration(testconf)
	if err != nil {
		t.Fatalf("Error when reading Mock configuration: %s", err)
	}
	t.Logf("Conf: %#v", conf)
}
