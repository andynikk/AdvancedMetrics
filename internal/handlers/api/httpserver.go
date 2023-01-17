package api

import (
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/mux"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/constants/errs"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/middlware"
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

type HTTPServer struct {
	RepStore  general.RepStore[handlers.RepStore]
	Router    *mux.Router
	RouterChi chi.Router
}

func InitRoutersMux(s *HTTPServer) {

	r := mux.NewRouter()

	r.HandleFunc("/", s.HandlerGetAllMetrics).Methods("GET")
	r.HandleFunc("/value/{metType}/{metName}", s.HandlerGetValue).Methods("GET")
	r.HandleFunc("/value", s.HandlerValueMetricaJSON).Methods("POST")

	r.Handle("/ping", middlware.CheckIP(s.HandlerPingDB)).Methods("GET")
	r.Handle("/update/{metType}/{metName}/{metValue}", middlware.CheckIP(s.HandlerSetMetricaPOST)).Methods("POST")
	r.Handle("/update", middlware.CheckIP(s.HandlerUpdateMetricJSON)).Methods("POST")
	r.Handle("/updates", middlware.CheckIP(s.HandlerUpdatesMetricJSON)).Methods("POST")

	r.HandleFunc("/debug/pprof", pprof.Index)
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	r.Handle("/debug/block", pprof.Handler("block"))
	r.Handle("/debug/goroutine", pprof.Handler("goroutine"))
	r.Handle("/debug/heap", pprof.Handler("heap"))
	r.Handle("/debug/threadcreate", pprof.Handler("threadcreate"))
	r.Handle("/debug/allocs", pprof.Handler("allocs"))
	r.Handle("/debug/mutex", pprof.Handler("mutex"))
	r.Handle("/debug/mutex", pprof.Handler("mutex"))

	s.Router = r
}

func InitRoutersChi(s *HTTPServer) {

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	r.HandleFunc("/", s.HandleFunc)
	r.NotFound(s.HandlerNotFound)

	r.Get("/", s.HandlerGetAllMetrics)
	r.Get("/value/{metType}/{metName}", s.HandlerGetValue)
	r.Post("/update/{metType}/{metName}/{metValue}", s.HandlerSetMetricaPOST)
	r.Post("/update", s.HandlerUpdateMetricJSON)
	r.Post("/updates", s.HandlerUpdatesMetricJSON)
	r.Post("/value", s.HandlerValueMetricaJSON)
	r.Get("/ping", s.HandlerPingDB)

	s.RouterChi = r
}

func (s *HTTPServer) HandlerNotFound(rw http.ResponseWriter, r *http.Request) {

	http.Error(rw, "Метрика "+r.URL.Path+" не найдена", http.StatusNotFound)

}

func (s *HTTPServer) HandleFunc(rw http.ResponseWriter, rq *http.Request) {

	defer rq.Body.Close()
	rw.WriteHeader(http.StatusOK)
}

// NewRepStore инициализация хранилища, роутера, заполнение настроек.
func NewRepStore(rs *handlers.RepStore) {

	smm := new(repository.SyncMapMetrics)
	smm.MutexRepo = make(repository.MutexRepo)
	rs.SyncMapMetrics = smm

	//InitRoutersMux(rs)

	//rs.Config = environment.InitConfigServer()
	//rs.PK, _ = encryption.InitPrivateKey(rs.Config.CryptoKey)

	//rs.Config.StorageType, _ = repository.InitStoreDB(rs.Config.StorageType, rs.Config.DatabaseDsn)
	//rs.Config.StorageType, _ = repository.InitStoreFile(rs.Config.StorageType, rs.Config.StoreFile)
}

func FillHeader(h http.Header) general.Header {
	header := make(general.Header)
	for key, valH := range h {
		for _, val := range valH {
			header[strings.ToLower(key)] = val
		}
	}

	return header
}

// HandlerUpdatesMetricJSON Handler, который работает с POST запросом формата "/updates".
// В теле получает массив JSON-значений со значением метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip. Сохраняет значение в физическое и временное хранилище.
func (s *HTTPServer) HandlerUpdatesMetricJSON(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(w, "ошибка на сервере", http.StatusInternalServerError)

	}

	header := FillHeader(r.Header)
	if err = s.RepStore.HandlerUpdatesMetricJSON(header, body); err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HandlerUpdateMetricJSON Handler, который работает с POST запросом формата "/update".
// В теле получает JSON со значением метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip.
// Сохраняет значение в физическое и временное хранилище.
func (s *HTTPServer) HandlerUpdateMetricJSON(rw http.ResponseWriter, rq *http.Request) {

	bytBody, err := io.ReadAll(rq.Body)
	if err != nil {
		constants.Logger.InfoLog(fmt.Sprintf("$$ 1 %s", err.Error()))
		http.Error(rw, "Ошибка получения Content-Encoding", http.StatusInternalServerError)
		return
	}

	h := FillHeader(rq.Header)

	err = s.RepStore.HandlerUpdateMetricJSON(h, bytBody)
	if err != nil {
		constants.Logger.InfoLog(fmt.Sprintf("$$ 1 %s", err.Error()))
		http.Error(rw, "ошибка обновления метрик", http.StatusInternalServerError)
		return
	}
}

// HandlerGetAllMetrics Отрабатывает обращение к корневому узлу сервера (/).
// Выводит на страницу список наименований и значений метрик.
func (s *HTTPServer) HandlerGetAllMetrics(rw http.ResponseWriter, rq *http.Request) {

	h := FillHeader(rq.Header)

	header, body := s.RepStore.HandlerGetAllMetrics(h)
	for key, val := range header {
		rw.Header().Set(key, val)
	}

	if _, err := rw.Write(body); err != nil {
		constants.Logger.ErrorLog(err)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

// HandlerGetValue Handler, который работает с GET запросом формата "/value/{metType}/{metName}"
// Где metType наименование типа метрики, metName наименование метрики
func (s *HTTPServer) HandlerGetValue(rw http.ResponseWriter, rq *http.Request) {

	//metName := mux.Vars(rq)["metName"]
	metName := chi.URLParam(rq, "metName")

	val, err := s.RepStore.HandlerGetValue([]byte(metName))
	if err != nil {
		constants.Logger.ErrorLog(err)
		rw.WriteHeader(errs.StatusHTTP(err))
		return
	}

	_, err = io.WriteString(rw, val)
	if err != nil {
		rw.WriteHeader(errs.StatusHTTP(err))
		constants.Logger.ErrorLog(err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// HandlerPingDB Handler, который работает с GET запросом формата "/ping"
// Handler проверяет соединение с физическим хранилищем метрик.
// Физическое хранилище регулируется параметром среды "DATABASE_DSN" или флагом "d"
// Если заполнено "DATABASE_DSN" или "d", то это база данных. Иначе файл.
func (s *HTTPServer) HandlerPingDB(rw http.ResponseWriter, rq *http.Request) {
	h := FillHeader(rq.Header)
	err := s.RepStore.HandlerPingDB(h)
	if err != nil {
		constants.Logger.ErrorLog(err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// HandlerValueMetricaJSON Handler, который работает с POST запросом формата "/value".
// В теле получает JSON с имененм типа и именем метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip. Возвращает значение метрики по типу и наименованию.
func (s *HTTPServer) HandlerValueMetricaJSON(rw http.ResponseWriter, rq *http.Request) {

	h := FillHeader(rq.Header)

	bytBody, err := io.ReadAll(rq.Body)
	if err != nil {
		constants.Logger.ErrorLog(err)
		http.Error(rw, "Ошибка получения Content-Encoding", http.StatusInternalServerError)
		return
	}

	_, b, err := s.RepStore.HandlerValueMetricaJSON(h, &bytBody)
	if err != nil {
		fmt.Println("+++++++++1", err)
		http.Error(rw, err.Error(), errs.StatusHTTP(err))
		return
	}

	fmt.Println("+++++++++2", string(b))
	if _, err = rw.Write(b); err != nil {
		constants.Logger.ErrorLog(err)
		return
	}
}

// HandlerSetMetricaPOST Handler, который работает с POST запросом формата "/update/{metType}/{metName}/{metValue}".
// Где metType наименование типа метрики, metName наименование метрики, metValue значение метрики.
// Значение метрики записывается во временное хранилище метрик repository.MapMetrics
func (s *HTTPServer) HandlerSetMetricaPOST(w http.ResponseWriter, r *http.Request) {

	//metType := mux.Vars(r)["metType"]
	//metName := mux.Vars(r)["metName"]
	//metValue := mux.Vars(r)["metValue"]

	metType := chi.URLParam(r, "metType")
	metName := chi.URLParam(r, "metName")
	metValue := chi.URLParam(r, "metValue")

	err := s.RepStore.HandlerSetMetricaPOST(metType, metName, metValue)
	w.WriteHeader(errs.StatusHTTP(err))
	//w.WriteHeader(http.StatusBadRequest)
	//fmt.Println(errs.StatusHTTP(err))
}
