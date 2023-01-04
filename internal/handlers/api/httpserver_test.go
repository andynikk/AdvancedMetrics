package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

var srv HTTPServer

func ExampleRepStore_HandlerGetAllMetrics() {
	ts := httptest.NewServer(srv.Router)
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

	ts := httptest.NewServer(srv.Router)
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

	ts := httptest.NewServer(srv.Router)
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

	storege := handlers.RepStore{}

	// NewRepStore инициализация хранилища, роутера, заполнение настроек.

	smm := new(repository.SyncMapMetrics)
	smm.MutexRepo = make(repository.MutexRepo)
	storege.SyncMapMetrics = smm

	sc := environment.ServerConfig{}
	sc.InitConfigServerENV()
	sc.InitConfigServerFile()
	sc.InitConfigServerDefault()

	storege.Config = &sc

	storege.PK, _ = encryption.InitPrivateKey(storege.Config.CryptoKey)

	storege.Config.TypeMetricsStorage, _ = repository.InitStoreDB(storege.Config.TypeMetricsStorage, storege.Config.DatabaseDsn)
	storege.Config.TypeMetricsStorage, _ = repository.InitStoreFile(storege.Config.TypeMetricsStorage, storege.Config.StoreFile)

	gRepStore := general.New[handlers.RepStore]()
	gRepStore.Set(constants.TypeSrvHTTP.String(), storege)

	srv.RepStore = gRepStore

	rp := srv.RepStore.Get(constants.TypeSrvHTTP.String())
	rp.MutexRepo = make(repository.MutexRepo)
	InitRoutersMux(&srv)

	valG := repository.Gauge(0)
	if ok := valG.SetFromText("0.001"); !ok {
		return
	}
	rp.MutexRepo["TestGauge"] = &valG
}