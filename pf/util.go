package main

import (
	"math"
	"os"
	"bufio"
	"strconv"
	"strings"
	"encoding/binary"
	crand "crypto/rand"
)

func randInt64() int64 {
	var out int64
	err := binary.Read(crand.Reader, binary.LittleEndian, &out)
	if err != nil {
		panic("Error getting random int, " + err.Error())
	}
	return out
}

// TODO(mae)  maybe try using the measurement to calculate velocity
func readCSV(filen string) [][]float64 {
	file, err := os.Open(filen)
	if err != nil {
		panic(err.Error())
	}
	defer file.Close()
	reader := bufio.NewScanner(file)
	reader.Split(bufio.ScanLines)
	output := make([][]float64, 0, 300)
	for i := 0; reader.Scan(); i++ {
		text := reader.Text()
		sections := strings.Split(text, ",")
		output = append(output, []float64{0.0, 0.0})
		output[i][0], _ = strconv.ParseFloat(sections[1], 64)
		output[i][1], _ = strconv.ParseFloat(sections[2], 64)
	}
	return output
}

func norm(a, b []float64) float64 {
	total := 0.0
	for i := range a {
		total += math.Pow(b[i] - a[i], 2)
	}
	return math.Sqrt(total)
}

func generateGrid(top, bottom float64, count int) []float64 {
	step := (top - bottom) / float64(count)
	var res []float64
	for ; bottom + step < top; bottom += step {
		res = append(res, bottom)
	}
	res = append(res, top)
	return res
}

func generateGridInt(top, bottom float64, count int) []int {
	step := (top - bottom) / float64(count)
	if count == 1 {
		return []int{int(bottom)}
	}
	if step < 1.0 {
		panic("Step must be greater than 1")
	}
	var res []int
	for ; bottom + step < top; bottom += step {
		res = append(res, int(bottom))
	}
	res = append(res, int(top))
	return res
}

