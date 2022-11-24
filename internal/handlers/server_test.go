package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/andynikk/advancedmetrics/internal/repository"
)

var rs RepStore

func ExampleRepStore_HandlerGetAllMetrics() {
	ts := httptest.NewServer(rs.Router)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("Metrics: %s. HTTP-Status: %d",
		resp.Header.Get("Metrics-Val"), resp.StatusCode)
	fmt.Println(msg)

	// Output:
	// Metrics: TestGauge = 0.001. HTTP-Status: 200
}

func ExampleRepStore_HandlerSetMetricaPOST() {

	ts := httptest.NewServer(rs.Router)
	defer ts.Close()

	req, err := http.NewRequest("POST", ts.URL+"/update/gauge/TestGauge/0.01", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	fmt.Print(resp.StatusCode)

	// Output:
	// 200
}

func ExampleRepStore_HandlerGetValue() {

	ts := httptest.NewServer(rs.Router)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/value/gauge/TestGauge", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	fmt.Print(resp.StatusCode)

	// Output:
	// 200
}

func init() {
	rs.MutexRepo = make(repository.MutexRepo)
	valG := repository.Gauge(0)
	if ok := valG.SetFromText("0.001"); !ok {
		return
	}
	rs.MutexRepo["TestGauge"] = &valG
	InitRoutersMux(&rs)
}
