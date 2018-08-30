package beaconpi

import (
  crand "crypto/rand"
  "math/rand"
  "encoding/binary"
  "encoding/base64"
)

func initRand() (*rand.Rand) {
	randbyte := make([]byte, 4)
	_, err := crand.Read(randbyte)
	if err != nil {
		panic("Getting random seed from system failed: " + err.Error())
	}
	seed, _ := binary.Varint(randbyte)
	rng := rand.New(rand.NewSource(seed))
  return rng
}

func getRand() (*rand.Rand) {
  rng := initRand()
  return rng
}

// Size is orders of 3 bytes to make sure there is no padding for convience
// bytes is 3*sets 
func RandBase64(rng *rand.Rand, sets int) string {
  b := make([]byte, sets*3)
  rng.Read(b)
  return base64.StdEncoding.EncodeToString(b)
}
