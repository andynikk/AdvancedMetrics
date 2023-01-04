package handlers

import (
	"net/http"

	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type MetricError int

// RepStore структура для настроек сервера, роутера и хранилище метрик.
// Хранилище метрик защищено sync.Mutex
type RepStore struct {
	Config *environment.ServerConfig
	PK     *encryption.KeyEncryption
	*repository.SyncMapMetrics
}

func (et MetricError) String() string {
	return [...]string{"Not error", "Error convert", "Error get type"}[et]
}

func (rs *RepStore) HandleFunc(rw http.ResponseWriter, rq *http.Request) {

	defer rq.Body.Close()
	rw.WriteHeader(http.StatusOK)
}

func (rs *RepStore) HandlerNotFound(rw http.ResponseWriter, r *http.Request) {

	http.Error(rw, "Метрика "+r.URL.Path+" не найдена", http.StatusNotFound)

}
