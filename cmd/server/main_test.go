package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/andynikk/advancedmetrics/internal/compression"
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/postgresql"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

func TestFuncServer(t *testing.T) {
	var fValue float64 = 0.001
	var iDelta int64 = 10

	var postStr = "http://127.0.0.1:8080/update/gauge/Alloc/0.1\nhttp://127.0.0.1:8080/update/gauge/" +
		"BuckHashSys/0.002\nhttp://127.0.0.1:8080/update/counter/PollCount/5"

	t.Run("Checking the filling of metrics Gauge", func(t *testing.T) {

		messageRaz := strings.Split(postStr, "\n")
		if len(messageRaz) != 3 {
			t.Errorf("The string (%s) was incorrectly decomposed into an array", postStr)
		}
	})

	t.Run("Checking connect DB", func(t *testing.T) {
		ctx := context.Background()

		sc := environment.SetConfigServer()
		dbConn, err := postgresql.PoolDB(sc.DatabaseDsn)
		if err != nil {
			t.Errorf("Error create DB connection")
		}
		t.Run("Checking create DB table", func(t *testing.T) {
			mapTypeStore := sc.TypeMetricsStorage
			mapTypeStore[constants.MetricsStorageDB.String()] = &repository.TypeStoreDataDB{
				DBC: *dbConn, Ctx: ctx, DBDsn: sc.DatabaseDsn,
			}
			if ok := mapTypeStore[constants.MetricsStorageDB.String()].CreateTable(); !ok {
				t.Errorf("Error create DB table")
			}
		})
	})

	t.Run("Checking handlers", func(t *testing.T) {
		rp := new(handlers.RepStore)
		rp.MutexRepo = make(repository.MutexRepo)
		handlers.InitRoutersMux(rp)
		ts := httptest.NewServer(rp.Router)
		defer ts.Close()

		t.Run("Checking handler /update/{metType}/{metName}/{metValue}/", func(t *testing.T) {
			resp := testRequest(t, ts, http.MethodPost, "/update/gauge/TestGauge/0.01", nil)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Error handler //update/{metType}/{metName}/{metValue}/")
			}
			t.Run("Checking handler /value/", func(t *testing.T) {
				resp := testRequest(t, ts, http.MethodGet, "/value/gauge/TestGauge", nil)
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Error handler /value/")
				}
			})
		})
		t.Run("Checking handler /update POST/", func(t *testing.T) {
			testA := testArray("")
			arrMetrics, err := json.MarshalIndent(testA, "", " ")
			if err != nil {
				t.Errorf("Error handler /update POST/")
			}
			body := bytes.NewReader(arrMetrics)
			resp := testRequest(t, ts, http.MethodPost, "/update", body)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Error handler /update POST/")
			}
			t.Run("Checking handler /value POST/", func(t *testing.T) {
				metricJSON, err := json.MarshalIndent(testMericGouge(""), "", " ")
				if err != nil {
					t.Errorf("Error handler /value POST/")
				}
				body := bytes.NewReader(metricJSON)

				resp := testRequest(t, ts, http.MethodPost, "/value", body)
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Error handler /value POST/")
				}
			})
		})
	})

	t.Run("Checking the filling of metrics", func(t *testing.T) {
		t.Run("Checking the type of the first line", func(t *testing.T) {
			var typeGauge = "gauge"

			messageRaz := strings.Split(postStr, "\n")
			valElArr := messageRaz[0]

			if strings.Contains(valElArr, typeGauge) == false {
				t.Errorf("The Gauge type was incorrectly determined")
			}
		})

		tests := []struct {
			name           string
			request        string
			wantStatusCode int
		}{
			{name: "Проверка на установку значения counter", request: "/update/counter/testSetGet332/6",
				wantStatusCode: http.StatusOK},
			{name: "Проверка на не правильный тип метрики", request: "/update/notcounter/testSetGet332/6",
				wantStatusCode: http.StatusNotImplemented},
			{name: "Проверка на не правильное значение метрики", request: "/update/counter/testSetGet332/non",
				wantStatusCode: http.StatusBadRequest},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				r := mux.NewRouter()
				ts := httptest.NewServer(r)

				rp := new(handlers.RepStore)
				rp.MutexRepo = make(repository.MutexRepo)
				rp.Router = nil

				r.HandleFunc("/update/{metType}/{metName}/{metValue}", rp.HandlerSetMetricaPOST).Methods("POST")

				defer ts.Close()
				resp := testRequest(t, ts, http.MethodPost, tt.request, nil)
				defer resp.Body.Close()

				if resp.StatusCode != tt.wantStatusCode {
					t.Errorf("Ответ не верен")
				}
			})
		}
	})

	t.Run("Checking the filling of metrics Counter", func(t *testing.T) {
		t.Run("Checking the filling of metrics Counter", func(t *testing.T) {
			var typeCounter = "counter"

			messageRaz := strings.Split(postStr, "\n")
			valElArr := messageRaz[2]

			if strings.Contains(valElArr, typeCounter) == false {
				t.Errorf("The Counter type was incorrectly determined")
			}
		})

	})

	t.Run("Checking compresion - decompression", func(t *testing.T) {

		textGzip := "Testing massage"
		arrByte := []byte(textGzip)

		compresMsg, err := compression.Compress(arrByte)
		if err != nil {
			t.Errorf("Error compres")
		}

		decompresMsg, err := compression.Decompress(compresMsg)
		if err != nil {
			t.Errorf("Error decompres")
		}

		msgReader := bytes.NewReader(decompresMsg)
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(msgReader); err != nil {
			t.Errorf("Error read decompression msg")
		}
		decompresText := buf.String()

		if decompresText != textGzip {
			t.Errorf("Error checking compresion - decompression")
		}
	})

	t.Run("Checking Hash SHA 256", func(t *testing.T) {
		configKey := "testKey"
		txtData := "Test data"

		hashData := cryptohash.HeshSHA256(txtData, configKey)
		print(len(hashData))
		if hashData == "" || len(hashData) != 64 {
			t.Errorf("Error checking Hash SHA 256")
		}

		t.Run("Checking set val in map", func(t *testing.T) {
			rs := new(handlers.RepStore)
			rs.MutexRepo = make(map[string]repository.Metric)

			arrM := testArray(configKey)

			for idx, val := range arrM {
				if idx == 0 {
					valG := repository.Gauge(0)
					rs.MutexRepo[val.ID] = &valG
				} else {
					valC := repository.Counter(0)
					rs.MutexRepo[val.ID] = &valC
				}
				rs.MutexRepo[val.ID].Set(val)
			}

			erorr := false
			for idx, val := range rs.MutexRepo {
				gauge := repository.Gauge(fValue)
				counter := repository.Counter(iDelta)
				if idx == "TestGauge" && val.String() != gauge.String() {
					erorr = true
				} else if idx == "TestCounter" && val.String() != counter.String() {
					erorr = true
				}
			}

			if erorr {
				t.Errorf("Error checking Hash SHA 256")
			}
		})
	})
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil
	}

	defer resp.Body.Close()

	return resp
}

func testArray(configKey string) encoding.ArrMetrics {
	var arrM encoding.ArrMetrics

	arrM = append(arrM, testMericGouge(configKey))
	arrM = append(arrM, testMericCaunter(configKey))

	return arrM
}

func testMericGouge(configKey string) encoding.Metrics {

	var fValue float64 = 0.001

	var mGauge encoding.Metrics
	mGauge.ID = "TestGauge"
	mGauge.MType = "gauge"
	mGauge.Value = &fValue
	mGauge.Hash = cryptohash.HeshSHA256(mGauge.ID, configKey)

	return mGauge
}

func testMericCaunter(configKey string) encoding.Metrics {
	var iDelta int64 = 10

	var mCounter encoding.Metrics
	mCounter.ID = "TestCounter"
	mCounter.MType = "counter"
	mCounter.Delta = &iDelta
	mCounter.Hash = cryptohash.HeshSHA256(mCounter.ID, configKey)

	return mCounter
}
