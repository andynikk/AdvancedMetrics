package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/andynikk/advancedmetrics/internal/compression"
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/postgresql"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type MetricType int
type MetricError int

const (
	GaugeMetric MetricType = iota
	CounterMetric
)

type HTMLParam struct {
	Title       string
	TextMetrics []string
}

type RepStore struct {
	Config environment.ServerConfig
	Router *mux.Router
	sync.Mutex
	repository.MapMetrics
}

func (mt MetricType) String() string {
	return [...]string{"gauge", "counter"}[mt]
}

func (et MetricError) String() string {
	return [...]string{"Not error", "Error convert", "Error get type"}[et]
}

func NewRepStore(rs *RepStore) {

	rs.MutexRepo = make(repository.MutexRepo)

	InitRoutersMux(rs)

	rs.Config = environment.SetConfigServer()

	mapTypeStore := rs.Config.TypeMetricsStorage
	if _, findKey := mapTypeStore[constants.MetricsStorageDB.String()]; findKey {
		ctx := context.Background()

		dbc, err := postgresql.PoolDB(rs.Config.DatabaseDsn)
		if err != nil {
			constants.Logger.ErrorLog(err)
		}

		mapTypeStore[constants.MetricsStorageDB.String()] = &repository.TypeStoreDataDB{
			DBC: *dbc, Ctx: ctx, DBDsn: rs.Config.DatabaseDsn,
		}
		if ok := mapTypeStore[constants.MetricsStorageDB.String()].CreateTable(); !ok {
			constants.Logger.ErrorLog(err)
		}
	}
	if _, findKey := mapTypeStore[constants.MetricsStorageFile.String()]; findKey {
		mapTypeStore[constants.MetricsStorageDB.String()] = &repository.TypeStoreDataFile{StoreFile: rs.Config.StoreFile}
	}
}

func InitRoutersMux(rs *RepStore) {

	r := mux.NewRouter()

	r.HandleFunc("/", rs.HandlerGetAllMetrics).Methods("GET")
	r.HandleFunc("/value/{metType}/{metName}", rs.HandlerGetValue).Methods("GET")
	r.HandleFunc("/ping", rs.HandlerPingDB).Methods("GET")

	r.HandleFunc("/update/{metType}/{metName}/{metValue}", rs.HandlerSetMetricaPOST).Methods("POST")
	r.HandleFunc("/update", rs.HandlerUpdateMetricJSON).Methods("POST")
	r.HandleFunc("/updates", rs.HandlerUpdatesMetricJSON).Methods("POST")
	r.HandleFunc("/value", rs.HandlerValueMetricaJSON).Methods("POST")

	r.HandleFunc("/debug/pprof", pprof.Index)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	//r.Handle("/debug/pprof/profile", pprof.Handler("profile"))
	r.Handle("/debug/block", pprof.Handler("block"))
	r.Handle("/debug/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/heap", pprof.Handler("heap"))
	r.Handle("/debug/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/allocs", pprof.Handler("allocs"))
	r.Handle("/debug/mutex", pprof.Handler("mutex"))

	rs.Router = r
}

func (rs *RepStore) setValueInMap(metType string, metName string, metValue string) int {

	switch metType {
	case GaugeMetric.String():
		if val, findKey := rs.MutexRepo[metName]; findKey {
			if ok := val.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}
		} else {

			valG := repository.Gauge(0)
			if ok := valG.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}

			rs.MutexRepo[metName] = &valG
		}

	case CounterMetric.String():
		if val, findKey := rs.MutexRepo[metName]; findKey {
			if ok := val.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}
		} else {

			valC := repository.Counter(0)
			if ok := valC.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}

			rs.MutexRepo[metName] = &valC
		}
	default:
		return http.StatusNotImplemented
	}

	return http.StatusOK
}

func (rs *RepStore) SetValueInMapJSON(a []encoding.Metrics) int {

	rs.Lock()
	defer rs.Unlock()

	for _, v := range a {
		var heshVal string

		switch v.MType {
		case GaugeMetric.String():
			var valValue float64
			valValue = *v.Value

			msg := fmt.Sprintf("%s:gauge:%f", v.ID, valValue)
			heshVal = cryptohash.HeshSHA256(msg, rs.Config.Key)
			if _, findKey := rs.MutexRepo[v.ID]; !findKey {
				valG := repository.Gauge(0)
				rs.MutexRepo[v.ID] = &valG
			}
		case CounterMetric.String():
			var valDelta int64
			valDelta = *v.Delta

			msg := fmt.Sprintf("%s:counter:%d", v.ID, valDelta)
			heshVal = cryptohash.HeshSHA256(msg, rs.Config.Key)
			if _, findKey := rs.MutexRepo[v.ID]; !findKey {
				valC := repository.Counter(0)
				rs.MutexRepo[v.ID] = &valC
			}
		default:
			return http.StatusNotImplemented
		}

		heshAgent := []byte(v.Hash)
		heshServer := []byte(heshVal)

		hmacEqual := hmac.Equal(heshServer, heshAgent)

		if v.Hash != "" && !hmacEqual {
			constants.Logger.InfoLog(fmt.Sprintf("++ %s - %s", v.Hash, heshVal))
			return http.StatusBadRequest
		}
		rs.MutexRepo[v.ID].Set(v)
	}
	return http.StatusOK

}

func (rs *RepStore) HandlerGetValue(rw http.ResponseWriter, rq *http.Request) {

	metType := mux.Vars(rq)["metType"]
	metName := mux.Vars(rq)["metName"]

	rs.Lock()
	defer rs.Unlock()

	if _, findKey := rs.MutexRepo[metName]; !findKey {
		constants.Logger.InfoLog(fmt.Sprintf("== %d", 3))
		rw.WriteHeader(http.StatusNotFound)
		http.Error(rw, "Метрика "+metName+" с типом "+metType+" не найдена", http.StatusNotFound)
		return
	}

	strMetric := rs.MutexRepo[metName].String()
	_, err := io.WriteString(rw, strMetric)
	if err != nil {
		constants.Logger.ErrorLog(err)
		return
	}

	rw.WriteHeader(http.StatusOK)

}

func (rs *RepStore) HandlerSetMetricaPOST(rw http.ResponseWriter, rq *http.Request) {

	rs.Lock()
	defer rs.Unlock()

	metType := mux.Vars(rq)["metType"]
	metName := mux.Vars(rq)["metName"]
	metValue := mux.Vars(rq)["metValue"]

	rw.WriteHeader(rs.setValueInMap(metType, metName, metValue))
}

func (rs *RepStore) HandlerUpdateMetricJSON(rw http.ResponseWriter, rq *http.Request) {

	var bodyJSON io.Reader

	contentEncoding := rq.Header.Get("Content-Encoding")
	bodyJSON = rq.Body
	if strings.Contains(contentEncoding, "gzip") {
		bytBody, err := ioutil.ReadAll(rq.Body)
		if err != nil {
			constants.Logger.InfoLog(fmt.Sprintf("$$ 1 %s", err.Error()))
			http.Error(rw, "Ошибка получения Content-Encoding", http.StatusInternalServerError)
			return
		}

		arrBody, err := compression.Decompress(bytBody)
		if err != nil {
			constants.Logger.InfoLog(fmt.Sprintf("$$ 2 %s", err.Error()))
			http.Error(rw, "Ошибка распаковки", http.StatusInternalServerError)
			return
		}

		bodyJSON = bytes.NewReader(arrBody)
	}

	var v []encoding.Metrics
	err := json.NewDecoder(bodyJSON).Decode(&v)
	if err != nil {
		constants.Logger.InfoLog(fmt.Sprintf("$$ 3 %s", err.Error()))
		http.Error(rw, "Ошибка получения JSON", http.StatusInternalServerError)
		return
	}

	rw.Header().Add("Content-Type", "application/json")
	res := rs.SetValueInMapJSON(v)
	rw.WriteHeader(res)

	var arrMetrics encoding.ArrMetrics
	for _, val := range v {
		mt := rs.MutexRepo[val.ID].GetMetrics(val.MType, val.ID, rs.Config.Key)
		metricsJSON, err := mt.MarshalMetrica()
		if err != nil {
			constants.Logger.ErrorLog(err)
			return
		}
		if _, err := rw.Write(metricsJSON); err != nil {
			constants.Logger.ErrorLog(err)
			return
		}
		arrMetrics = append(arrMetrics, mt)
	}

	if res == http.StatusOK {
		for _, val := range rs.Config.TypeMetricsStorage {
			val.WriteMetric(arrMetrics)
		}
	}
}

func (rs *RepStore) HandlerUpdatesMetricJSON(rw http.ResponseWriter, rq *http.Request) {

	var bodyJSON io.Reader
	var arrBody []byte

	contentEncoding := rq.Header.Get("Content-Encoding")

	bodyJSON = rq.Body
	if strings.Contains(contentEncoding, "gzip") {
		bytBody, err := ioutil.ReadAll(rq.Body)
		if err != nil {
			constants.Logger.ErrorLog(err)
			http.Error(rw, "Ошибка получения Content-Encoding", http.StatusInternalServerError)
			return
		}

		arrBody, err = compression.Decompress(bytBody)
		if err != nil {
			constants.Logger.ErrorLog(err)
			http.Error(rw, "Ошибка распаковки", http.StatusInternalServerError)
			return
		}

		bodyJSON = bytes.NewReader(arrBody)
	}

	respByte, err := ioutil.ReadAll(bodyJSON)
	if err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(rw, "Ошибка распаковки", http.StatusInternalServerError)
	}

	var storedData encoding.ArrMetrics
	if err := json.Unmarshal(respByte, &storedData); err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(rw, "Ошибка распаковки", http.StatusInternalServerError)
	}

	rs.SetValueInMapJSON(storedData)

	for _, val := range rs.Config.TypeMetricsStorage {
		val.WriteMetric(storedData)
	}
}

func (rs *RepStore) HandlerValueMetricaJSON(rw http.ResponseWriter, rq *http.Request) {

	var bodyJSON io.Reader
	bodyJSON = rq.Body

	acceptEncoding := rq.Header.Get("Accept-Encoding")
	contentEncoding := rq.Header.Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		constants.Logger.InfoLog("-- метрика с агента gzip (value)")
		bytBody, err := ioutil.ReadAll(rq.Body)
		if err != nil {
			constants.Logger.ErrorLog(err)
			http.Error(rw, "Ошибка получения Content-Encoding", http.StatusInternalServerError)
			return
		}

		arrBody, err := compression.Decompress(bytBody)
		if err != nil {
			constants.Logger.ErrorLog(err)
			http.Error(rw, "Ошибка распаковки", http.StatusInternalServerError)
			return
		}

		bodyJSON = bytes.NewReader(arrBody)
	}

	v := encoding.Metrics{}
	err := json.NewDecoder(bodyJSON).Decode(&v)
	if err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(rw, "Ошибка получения JSON", http.StatusInternalServerError)
		return
	}
	metType := v.MType
	metName := v.ID

	rs.Lock()
	defer rs.Unlock()

	if _, findKey := rs.MutexRepo[metName]; !findKey {

		constants.Logger.InfoLog(fmt.Sprintf("== %d %s %d %s", 1, metName, len(rs.MutexRepo), rs.Config.DatabaseDsn))

		rw.WriteHeader(http.StatusNotFound)
		http.Error(rw, "Метрика "+metName+" с типом "+metType+" не найдена", http.StatusNotFound)
		return
	}

	mt := rs.MutexRepo[metName].GetMetrics(metType, metName, rs.Config.Key)
	metricsJSON, err := mt.MarshalMetrica()
	if err != nil {
		constants.Logger.ErrorLog(err)
		return
	}

	var bytMterica []byte
	bt := bytes.NewBuffer(metricsJSON).Bytes()
	bytMterica = append(bytMterica, bt...)
	compData, err := compression.Compress(bytMterica)
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	var bodyBate []byte
	rw.Header().Add("Content-Type", "application/json")
	if strings.Contains(acceptEncoding, "gzip") {
		rw.Header().Add("Content-Encoding", "gzip")
		bodyBate = compData
	} else {
		bodyBate = metricsJSON
	}

	if _, err := rw.Write(bodyBate); err != nil {
		constants.Logger.ErrorLog(err)
		return
	}
}

func (rs *RepStore) HandlerPingDB(rw http.ResponseWriter, rq *http.Request) {
	defer rq.Body.Close()
	mapTypeStore := rs.Config.TypeMetricsStorage
	if _, findKey := mapTypeStore[constants.MetricsStorageDB.String()]; !findKey {
		constants.Logger.ErrorLog(errors.New("соединение с базой отсутствует"))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mapTypeStore[constants.MetricsStorageDB.String()].ConnDB() == nil {
		constants.Logger.ErrorLog(errors.New("соединение с базой отсутствует"))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (rs *RepStore) HandleFunc(rw http.ResponseWriter, rq *http.Request) {

	defer rq.Body.Close()
	rw.WriteHeader(http.StatusOK)
}

func (rs *RepStore) HandlerGetAllMetrics(rw http.ResponseWriter, rq *http.Request) {

	arrMetricsAndValue := rs.MapMetrics.TextMetricsAndValue()

	content := `<!DOCTYPE html>
				<html>
				<head>
  					<meta charset="UTF-8">
  					<title>МЕТРИКИ</title>
				</head>
				<body>
				<h1>МЕТРИКИ</h1>
				<ul>
				`
	for _, val := range arrMetricsAndValue {
		content = content + `<li><b>` + val + `</b></li>` + "\n"
	}
	content = content + `</ul>
						</body>
						</html>`

	acceptEncoding := rq.Header.Get("Accept-Encoding")

	metricsHTML := []byte(content)
	byteMterics := bytes.NewBuffer(metricsHTML).Bytes()
	compData, err := compression.Compress(byteMterics)
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	var bodyBate []byte
	if strings.Contains(acceptEncoding, "gzip") {
		rw.Header().Add("Content-Encoding", "gzip")
		bodyBate = compData
	} else {
		bodyBate = metricsHTML
	}

	rw.Header().Add("Content-Type", "text/html")
	if _, err := rw.Write(bodyBate); err != nil {
		constants.Logger.ErrorLog(err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (rs *RepStore) PrepareDataBU() encoding.ArrMetrics {

	var storedData encoding.ArrMetrics
	for key, val := range rs.MutexRepo {
		storedData = append(storedData, val.GetMetrics(val.Type(), key, rs.Config.Key))
	}
	return storedData
}

func (rs *RepStore) RestoreData() {

	var arrMetricsAll []encoding.Metrics

	for _, val := range rs.Config.TypeMetricsStorage {
		arrMetrics, err := val.GetMetric()
		if err != nil {
			constants.Logger.ErrorLog(err)
			continue
		}
		for _, valArr := range arrMetrics {
			arrMetricsAll = append(arrMetricsAll, valArr)
		}
	}

	rs.SetValueInMapJSON(arrMetricsAll)
}

func (rs *RepStore) BackupData() {

	ctx, cancelFunc := context.WithCancel(context.Background())
	saveTicker := time.NewTicker(rs.Config.StoreInterval)
	for {
		select {
		case <-saveTicker.C:

			for _, val := range rs.Config.TypeMetricsStorage {
				val.WriteMetric(rs.PrepareDataBU())
			}

		case <-ctx.Done():
			cancelFunc()
			return
		}
	}
}

func (rs *RepStore) HandlerNotFound(rw http.ResponseWriter, r *http.Request) {

	http.Error(rw, "Метрика "+r.URL.Path+" не найдена", http.StatusNotFound)

}
