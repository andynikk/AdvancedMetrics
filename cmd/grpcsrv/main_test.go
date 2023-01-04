package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/andynikk/advancedmetrics/internal/compression"
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/constants/errs"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers/api"
	"github.com/andynikk/advancedmetrics/internal/networks"
	"google.golang.org/grpc/metadata"
)

func TestFuncServer(t *testing.T) {
	server := new(server)

	t.Run("Checking init server", func(t *testing.T) {
		grpchandlers.NewRepStore(&server.storege)
		if server.storege.Config.Address == "" {
			t.Errorf("Error checking init server")
		}
	})

	gRepStore := general.New[grpchandlers.RepStore]()
	data := gRepStore.Get(constants.TypeSrvGRPC.String())

	t.Run("Checking init generics val", func(t *testing.T) {
		gRepStore.Set(constants.TypeSrvGRPC.String(), server.storege)

		data = gRepStore.Get(constants.TypeSrvGRPC.String())
		if data.Config.Address == "" {
			t.Errorf("Error checking init generics val")
		}
	})

	var IPAddress string
	t.Run("Checking get current IP", func(t *testing.T) {
		hn, _ := os.Hostname()
		IPs, _ := net.LookupIP(hn)
		IPAddress = networks.IPStr(IPs)
		if IPAddress == "" {
			t.Errorf("Error checking get current IP")
		}
	})

	mHeader := map[string]string{"Content-Type": "application/json",
		"Content-Encoding": "gzip",
		"X-Real-IP":        constants.TrustedSubnet}
	if data.PK != nil && data.PK.PrivateKey != nil && data.PK.PublicKey != nil {
		mHeader["Content-Encryption"] = data.PK.TypeEncryption
	}

	md := metadata.New(mHeader)
	ctx := context.TODO()

	ctx = metadata.NewOutgoingContext(ctx, md)

	GRPCServer := api.GRPCServer{gRepStore}

	t.Run("Checking handlers PING", func(t *testing.T) {

		req := api.EmtyRequest{}
		bl, err := GRPCServer.Ping(ctx, &req)
		if !bl.Result || err != nil {
			t.Errorf("Error checking handlers PING.")
		}
	})

	t.Run("Checking handlers Update", func(t *testing.T) {
		tests := []struct {
			name           string
			request        api.UpdateRequest
			wantStatusCode int
		}{
			{name: "Проверка на установку значения counter", request: api.UpdateRequest{MetType: []byte("counter"),
				MetName: []byte("testSetGet332"), MetValue: []byte("6")}, wantStatusCode: http.StatusOK},
			{name: "Проверка на не правильный тип метрики", request: api.UpdateRequest{MetType: []byte("notcounter"),
				MetName: []byte("testSetGet332"), MetValue: []byte("6")}, wantStatusCode: http.StatusNotImplemented},
			{name: "Проверка на не правильное значение метрики", request: api.UpdateRequest{MetType: []byte("counter"),
				MetName: []byte("testSetGet332"), MetValue: []byte("non")}, wantStatusCode: http.StatusBadRequest},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := GRPCServer.Update(ctx, &tt.request)
				if errs.StatusError(err) != int32(tt.wantStatusCode) {
					t.Errorf("Error checking handlers Update. %s", tt.name)
				}
			})
		}
	})

	t.Run("Checking handlers Update JSON", func(t *testing.T) {
		tests := []struct {
			name           string
			request        encoding.Metrics
			wantStatusCode int
		}{
			{name: "Проверка на установку значения gauge", request: testMericGouge(data.Config.Key),
				wantStatusCode: http.StatusOK},
			{name: "Проверка на установку значения counter", request: testMericCaunter(data.Config.Key),
				wantStatusCode: http.StatusOK},
			{name: "Проверка на не правильный тип метрики gauge", request: testMericNoGouge(data.Config.Key),
				wantStatusCode: http.StatusNotImplemented},
			{name: "Проверка на не правильный тип метрики counter", request: testMericNoCounter(data.Config.Key),
				wantStatusCode: http.StatusNotImplemented},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var gziparrMetrics []byte
				var storedData encoding.ArrMetrics
				storedData = append(storedData, tt.request)

				t.Run("Checking gzip", func(t *testing.T) {
					arrMetrics, err := json.MarshalIndent(storedData, "", " ")
					if err != nil {
						t.Errorf("Error checking gzip. %s", tt.name)
					}

					gziparrMetrics, err = compression.Compress(arrMetrics)
					if err != nil {
						t.Errorf("Error checking gzip. %s", tt.name)
					}

				})

				req := api.UpdateStrRequest{Body: gziparrMetrics}
				bl, err := GRPCServer.UpdateJSON(ctx, &req)
				if !bl.Result && errs.StatusError(err) != int32(tt.wantStatusCode) {
					t.Errorf("Error checking handlers Update JSON. %s", tt.name)
				}
			})
		}
	})

	t.Run("Checking handlers Updates JSON", func(t *testing.T) {
		var storedData encoding.ArrMetrics
		storedData = append(storedData, testMericGouge(data.Config.Key))
		storedData = append(storedData, testMericCaunter(data.Config.Key))

		arrMetrics, err := json.MarshalIndent(storedData, "", " ")
		if err != nil {
			t.Errorf("Error checking gzip. %s", "Updates JSON")
		}
		gziparrMetrics, err := compression.Compress(arrMetrics)
		if err != nil {
			t.Errorf("Error checking gzip. %s", "Updates JSON")
		}

		req := api.UpdatesRequest{Body: gziparrMetrics}
		bl, err := GRPCServer.UpdatesJSON(ctx, &req)
		if !bl.Result || err != nil {
			t.Errorf("Error checking handlers Update JSON.")
		}
	})

	t.Run("Checking handlers Value JSON", func(t *testing.T) {

		tests := []struct {
			name           string
			request        encoding.Metrics
			wantStatusCode int
		}{
			{name: "Проверка на установку значения gauge", request: testMericGouge(data.Config.Key),
				wantStatusCode: http.StatusOK},
			{name: "Проверка на установку значения counter", request: testMericCaunter(data.Config.Key),
				wantStatusCode: http.StatusOK},
			{name: "Проверка на не правильное значение метрики gauge", request: testMericWrongGouge(data.Config.Key),
				wantStatusCode: http.StatusNotFound},
			{name: "Проверка на не правильное значение метрики counter", request: testMericWrongCounter(data.Config.Key),
				wantStatusCode: http.StatusNotFound},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				arrMetrics, err := json.MarshalIndent(tt.request, "", " ")
				if err != nil {
					t.Errorf("Error checking gzip. %s", tt.name)
				}

				gziparrMetrics, err := compression.Compress(arrMetrics)
				if err != nil {
					t.Errorf("Error checking gzip. %s", tt.name)
				}

				req := api.UpdatesRequest{Body: gziparrMetrics}
				_, err = GRPCServer.ValueJSON(ctx, &req)

				if errs.StatusError(err) != int32(tt.wantStatusCode) {
					t.Errorf("Error checking handlers Value JSON. %s", tt.name)
				}
			})
		}
	})

	t.Run("Checking handlers Value", func(t *testing.T) {

		tests := []struct {
			name           string
			request        string
			wantStatusCode int
		}{
			{name: "Проверка на установку значения gauge", request: testMericGouge(data.Config.Key).ID,
				wantStatusCode: http.StatusOK},
			{name: "Проверка на установку значения counter", request: testMericCaunter(data.Config.Key).ID,
				wantStatusCode: http.StatusOK},
			{name: "Проверка на не правильное значение метрики gauge", request: testMericWrongGouge(data.Config.Key).ID,
				wantStatusCode: http.StatusInternalServerError},
			{name: "Проверка на не правильное значение метрики counter", request: testMericWrongCounter(data.Config.Key).ID,
				wantStatusCode: http.StatusInternalServerError},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {

				req := api.UpdatesRequest{Body: []byte(tt.request)}
				_, err := GRPCServer.Value(ctx, &req)

				if errs.StatusError(err) != int32(tt.wantStatusCode) {
					t.Errorf("Error checking handlers Value. %s", tt.name)
				}
			})
		}
	})

	t.Run("Checking handlers ListMetrics", func(t *testing.T) {

		req := api.EmtyRequest{}
		res, _ := GRPCServer.ListMetrics(ctx, &req)

		if !strings.Contains(string(res.Result), "TestGauge") ||
			!strings.Contains(string(res.Result), "TestCounter") {
			t.Errorf("Error checking handlers ListMetrics.")
		}
	})

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

func testMericWrongGouge(configKey string) encoding.Metrics {

	var fValue float64 = 0.001

	var mGauge encoding.Metrics
	mGauge.ID = "TestGauge322"
	mGauge.MType = "gauge"
	mGauge.Value = &fValue
	mGauge.Hash = cryptohash.HeshSHA256(mGauge.ID, configKey)

	return mGauge
}

func testMericNoGouge(configKey string) encoding.Metrics {

	var fValue float64 = 0.001

	var mGauge encoding.Metrics
	mGauge.ID = "TestGauge"
	mGauge.MType = "nogauge"
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

func testMericNoCounter(configKey string) encoding.Metrics {
	var iDelta int64 = 10

	var mCounter encoding.Metrics
	mCounter.ID = "TestCounter"
	mCounter.MType = "nocounter"
	mCounter.Delta = &iDelta
	mCounter.Hash = cryptohash.HeshSHA256(mCounter.ID, configKey)

	return mCounter
}

func testMericWrongCounter(configKey string) encoding.Metrics {
	var iDelta int64 = 10

	var mCounter encoding.Metrics
	mCounter.ID = "TestCounter322"
	mCounter.MType = "counter"
	mCounter.Delta = &iDelta
	mCounter.Hash = cryptohash.HeshSHA256(mCounter.ID, configKey)

	return mCounter
}
