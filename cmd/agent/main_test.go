package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

func TestmakeMsg(memStats MetricsGauge) string {

	const adresServer = "127.0.0.1:8080"
	const msgFormat = "http://%s/update/%s/%s/%v"

	var msg []string

	val := memStats["Alloc"]
	msg = append(msg, fmt.Sprintf(msgFormat, adresServer, val.Type(), "Alloc", 0.1))

	val = memStats["BuckHashSys"]
	msg = append(msg, fmt.Sprintf(msgFormat, adresServer, val.Type(), "BuckHashSys", 0.002))

	return strings.Join(msg, "\n")
}

func TestFuncAgen(t *testing.T) {
	a := agent{}
	a.data.metricsGauge = make(MetricsGauge)

	var argErr = "err"

	t.Run("Checking the structure creation", func(t *testing.T) {

		var realResult MetricsGauge

		if a.data.metricsGauge["Alloc"] != realResult["Alloc"] && a.data.metricsGauge["RandomValue"] != realResult["RandomValue"] {

			//t.Errorf("Structure creation error", resultMS, realResult)
			t.Errorf("Structure creation error (%s)", argErr)
		}
		t.Run("Creating a submission line", func(t *testing.T) {
			var resultStr = "http://127.0.0.1:8080/update/gauge/Alloc/0.1\nhttp://127.0.0.1:8080/update/gauge/BuckHashSys/0.002"

			resultMassage := TestmakeMsg(realResult)

			if resultStr != resultMassage {

				//t.Errorf("Error creating a submission line", string(resultMS), realResult)
				t.Errorf("Error creating a submission line (%s)", argErr)
			}
		})
	})

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	a.fillMetric(&mem)
	t.Run("Checking the filling of metrics Gauge", func(t *testing.T) {

		val := a.data.metricsGauge["Frees"]
		if val.Type() != "gauge" {
			t.Errorf("Metric %s is not a type %s", "Frees", "Gauge")
		}
	})

	t.Run("Checking the metrics value Gauge", func(t *testing.T) {
		if a.data.metricsGauge["Alloc"] == 0 {
			t.Errorf("The metric %s a value of %v", "Alloc", 0)
		}

	})

	t.Run("Checking fillings the metrics", func(t *testing.T) {
		a.fillMetric(&mem)
		allMetrics := make(emtyArrMetrics, 0)
		i := 0
		tempMetricsGauge := &a.data.metricsGauge
		for key, val := range *tempMetricsGauge {
			valFloat64 := float64(val)

			msg := fmt.Sprintf("%s:gauge:%f", key, valFloat64)
			heshVal := cryptohash.HeshSHA256(msg, a.cfg.Key)

			metrica := encoding.Metrics{ID: key, MType: val.Type(), Value: &valFloat64, Hash: heshVal}
			allMetrics = append(allMetrics, metrica)

			i++
			if i == constants.ButchSize {
				if err := a.goPost2Server(allMetrics); err != nil {
					t.Errorf("Error checking fillings the metrics")
				}
				allMetrics = make(emtyArrMetrics, 0)
				i = 0
			}
		}

		cPollCount := repository.Counter(a.data.pollCount)
		msg := fmt.Sprintf("%s:counter:%d", "PollCount", a.data.pollCount)
		heshVal := cryptohash.HeshSHA256(msg, a.cfg.Key)

		metrica := encoding.Metrics{ID: "PollCount", MType: cPollCount.Type(),
			Delta: &a.data.pollCount, Hash: heshVal}
		allMetrics = append(allMetrics, metrica)
		if err := a.goPost2Server(allMetrics); err != nil {
			t.Errorf("Error checking fillings the metrics")
		}
	})

	t.Run("Checking the filling of metrics PollCount", func(t *testing.T) {

		val := repository.Counter(a.data.pollCount)
		if val.Type() != "counter" {
			t.Errorf("Metric %s is not a type %s", "Frees", "Counter")
		}
	})

	t.Run("Checking the metrics value PollCount", func(t *testing.T) {
		if a.data.pollCount == 0 {
			t.Errorf("The metric %s a value of %v", "PollCount", 0)
		}

	})

	t.Run("Increasing the metric PollCount", func(t *testing.T) {
		var res = int64(2)
		if a.data.pollCount != res {
			t.Errorf("The metric %s has not increased by %v", "PollCount", res)
		}

	})

}

func BenchmarkSendMetrics(b *testing.B) {
	a := agent{}
	a.cfg.Address = "localhost:8080"

	wg := sync.WaitGroup{}
	for i := 0; i < 10000; i++ {
		var allMetrics emtyArrMetrics

		val := repository.Gauge(0)
		for j := 0; j < 10; j++ {
			val = val + 1
			id := fmt.Sprintf("Metric %d", j)
			floatJ := float64(j)
			metrica := encoding.Metrics{ID: id, MType: val.Type(), Value: &floatJ, Hash: ""}
			allMetrics = append(allMetrics, metrica)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.goPost2Server(allMetrics)
		}()
	}
	wg.Wait()
}
