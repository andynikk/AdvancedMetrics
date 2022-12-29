package general

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andynikk/advancedmetrics/internal/compression"
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/constants/errs"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	grpchandlers "github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type IRepStore interface {
	handlers.RepStore | grpchandlers.RepStore
}

type Header map[string]string

type RepStore[T IRepStore] struct {
	data map[string]T
}

func (rs *RepStore[T]) Set(key string, value T) {
	rs.data[key] = value
}

func (rs *RepStore[T]) Get(key string) (v T) {
	if v, ok := rs.data[key]; ok {
		return v
	}

	return
}

func New[T IRepStore]() RepStore[T] {

	c := RepStore[T]{}
	c.data = make(map[string]T)

	return c
}

func (rs *RepStore[T]) getPKRepStore() *encryption.KeyEncryption {
	keyG := "grpchandlers"
	if t, ok := rs.data[keyG]; ok {
		return any(t).(grpchandlers.RepStore).PK
	}

	keyH := "handlers"
	if t, ok := rs.data[keyH]; ok {
		return any(t).(handlers.RepStore).PK
	}

	return &encryption.KeyEncryption{}
}

func (rs *RepStore[T]) getConfigRepStore() *environment.ServerConfig {
	keyG := "grpchandlers"
	if t, ok := rs.data[keyG]; ok {
		return any(t).(grpchandlers.RepStore).Config
	}

	keyH := "handlers"
	if t, ok := rs.data[keyH]; ok {
		return any(t).(handlers.RepStore).Config
	}

	return &environment.ServerConfig{}
}

func (rs *RepStore[T]) getSyncMapMetricsRepStore() *repository.SyncMapMetrics {
	keyG := "grpchandlers"
	if t, ok := rs.data[keyG]; ok {
		return any(t).(grpchandlers.RepStore).SyncMapMetrics
	}

	keyH := "handlers"
	if t, ok := rs.data[keyH]; ok {
		return any(t).(handlers.RepStore).SyncMapMetrics
	}

	return &repository.SyncMapMetrics{}
}

func (rs *RepStore[T]) RestoreData() {

	var arrMetricsAll []encoding.Metrics

	config := rs.getConfigRepStore()
	typeMetricsStorage := config.TypeMetricsStorage

	for _, valMetric := range typeMetricsStorage {

		arrMetrics, err := valMetric.GetMetric()
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

func (rs *RepStore[T]) SetValueInMapJSON(a []encoding.Metrics) int {

	key := rs.getConfigRepStore().Key
	smm := rs.getSyncMapMetricsRepStore()

	smm.Lock()
	defer smm.Unlock()

	for _, v := range a {
		var heshVal string

		switch v.MType {
		case handlers.GaugeMetric.String():
			var valValue float64
			valValue = *v.Value

			msg := fmt.Sprintf("%s:gauge:%f", v.ID, valValue)
			heshVal = cryptohash.HeshSHA256(msg, key)
			if _, findKey := smm.MutexRepo[v.ID]; !findKey {
				valG := repository.Gauge(0)
				smm.MutexRepo[v.ID] = &valG
			}
		case handlers.CounterMetric.String():
			var valDelta int64
			valDelta = *v.Delta

			msg := fmt.Sprintf("%s:counter:%d", v.ID, valDelta)
			heshVal = cryptohash.HeshSHA256(msg, key)
			if _, findKey := smm.MutexRepo[v.ID]; !findKey {
				valC := repository.Counter(0)
				smm.MutexRepo[v.ID] = &valC
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
		smm.MutexRepo[v.ID].Set(v)
	}
	return http.StatusOK

}

func (rs *RepStore[T]) BackupData() {

	rsConfig := rs.getConfigRepStore()
	typeMetricsStorage := rsConfig.TypeMetricsStorage
	storeInterval := rsConfig.StoreInterval

	ctx, cancelFunc := context.WithCancel(context.Background())
	saveTicker := time.NewTicker(storeInterval)
	for {
		select {
		case <-saveTicker.C:

			for _, val := range typeMetricsStorage {
				val.WriteMetric(rs.PrepareDataBU())
			}

		case <-ctx.Done():
			cancelFunc()
			return
		}
	}
}

func (rs *RepStore[T]) PrepareDataBU() encoding.ArrMetrics {

	cKey := rs.getConfigRepStore().Key
	smm := rs.getSyncMapMetricsRepStore()

	var storedData encoding.ArrMetrics
	for key, val := range smm.MutexRepo {
		storedData = append(storedData, val.GetMetrics(val.Type(), key, cKey))
	}
	return storedData
}

// Shutdown working out the service stop.
// We save the current values of metrics in the database.
func (rs *RepStore[T]) Shutdown() {

	typeMetricsStorage := rs.getConfigRepStore().TypeMetricsStorage
	smm := rs.getSyncMapMetricsRepStore()

	smm.Lock()
	defer smm.Unlock()

	for _, val := range typeMetricsStorage {
		val.WriteMetric(rs.PrepareDataBU())
	}
	constants.Logger.InfoLog("server stopped")
}

// HandlerUpdatesMetricJSON Handler, который работает с POST запросом формата "/updates".
// В теле получает массив JSON-значений со значением метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip. Сохраняет значение в физическое и временное хранилище.
func (rs *RepStore[T]) HandlerUpdatesMetricJSON(h Header, b []byte) error {

	contentEncoding := h["Content-Encoding"]
	contentEncryption := h["Content-Encryption"]

	PK := rs.getPKRepStore()

	err := errors.New("")
	if strings.Contains(contentEncryption, constants.TypeEncryption) {
		if b, err = PK.RsaDecrypt(b); err != nil {
			constants.Logger.ErrorLog(err)
			return errs.ErrDecrypt
		}
	}

	if strings.Contains(contentEncoding, "gzip") {
		b, err = compression.Decompress(b)
		if err != nil {
			constants.Logger.ErrorLog(err)
			return errs.ErrDecompress
		}
	}
	if err = rs.Updates(b); err != nil {
		if err == errs.ErrStatusInternalServer {
			return errs.ErrDecompress
		}
	}

	return nil
}

func (rs *RepStore[T]) Updates(msg []byte) error {

	bodyJSON := bytes.NewReader(msg)
	respByte, err := io.ReadAll(bodyJSON)

	if err != nil {
		constants.Logger.ErrorLog(err)
		return errs.ErrStatusInternalServer
	}

	var storedData encoding.ArrMetrics
	if err := json.Unmarshal(respByte, &storedData); err != nil {
		constants.Logger.ErrorLog(err)
		return errs.ErrStatusInternalServer
	}

	rs.SetValueInMapJSON(storedData)

	typeMetricsStorage := rs.getConfigRepStore().TypeMetricsStorage
	for _, val := range typeMetricsStorage {
		val.WriteMetric(storedData)
	}

	return nil
}
