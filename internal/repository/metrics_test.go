package repository_test

import (
	"fmt"
	"strconv"

	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

func ExampleGauge_String() {
	var g repository.Gauge = 0.01
	fmt.Print(g.String())

	// Output:
	// 0.01
}

func ExampleGauge_Type() {
	var g repository.Gauge = 0.01
	fmt.Print(g.Type())

	// Output:
	// gauge
}

func ExampleGauge_GetMetrics() {

	mType := "gauge"
	id := "TestGauge"
	hashKey := "TestHash"

	var g repository.Gauge = 0.01
	value := float64(g)
	msg := fmt.Sprintf("%s:%s:%f", id, mType, value)
	heshVal := cryptohash.HeshSHA256(msg, hashKey)
	mt := encoding.Metrics{ID: id, MType: mType, Value: &value, Hash: heshVal}
	fmt.Print(fmt.Sprintf("%s,%s,%v,%s", mt.MType, mt.ID, mt.Value, mt.Hash))

	// Output:
	// gauge,TestGauge,0xc00000f010,4e5d8a0e257dd12355b15f730591dddd9e45e18a6ef67460a58f20edc12c9465
}

func ExampleGauge_Set() {
	var g repository.Gauge
	var f float64 = 0.01
	v := encoding.Metrics{
		ID:    "",
		MType: "",
		Value: &f,
		Hash:  "",
	}
	g.Set(v)
	fmt.Print(g)

	// Output:
	// 0.01
}

func ExampleGauge_SetFromText() {

	metValue := "0.01"

	predVal, _ := strconv.ParseFloat(metValue, 64)
	g := repository.Gauge(predVal)

	fmt.Print(g)

	// Output:
	// 0.01
}

////////////////////////////////////////////////////////////

func ExampleCounter_String() {
	var c repository.Counter = 58
	fmt.Println(c.String())

	// Output:
	// 58
}

func ExampleCounter_Type() {
	var c repository.Counter = 58
	fmt.Print(c.Type())

	// Output:
	// counter
}

func ExampleCounter_GetMetrics() {

	mType := "counter"
	id := "TestGauge"
	hashKey := "TestHash"

	var c repository.Counter = 58
	value := float64(c)
	msg := fmt.Sprintf("%s:%s:%f", id, mType, value)
	heshVal := cryptohash.HeshSHA256(msg, hashKey)
	mt := encoding.Metrics{ID: id, MType: mType, Value: &value, Hash: heshVal}
	fmt.Print(fmt.Sprintf("%s,%s,%v,%s", mt.MType, mt.ID, mt.Value, mt.Hash))

	// Output:
	// counter,TestGauge,0xc00000f030,85c5563bc3b6c007d2b4c069f0b293d3f6a30764dbbc3c0952965a31954947e6
}

func ExampleCounter_Set() {
	var c repository.Counter
	var i int64 = 58
	v := encoding.Metrics{
		ID:    "",
		MType: "",
		Delta: &i,
		Hash:  "",
	}
	c.Set(v)
	fmt.Print(c)

	// Output:
	// 58
}

func ExampleCounter_SetFromText() {

	metValue := "0.01"

	predVal, _ := strconv.ParseFloat(metValue, 64)
	g := repository.Gauge(predVal)

	fmt.Print(g)

	// Output:
	// 0.01
}
