// Start of the service for getting metrics.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/handlers"
)

type server struct {
	storege handlers.RepStore
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

// Shutdown working out the service stop.
// We save the current values of metrics in the database.
func Shutdown(rs *handlers.RepStore) {
	rs.Lock()
	defer rs.Unlock()

	for _, val := range rs.Config.TypeMetricsStorage {
		val.WriteMetric(rs.PrepareDataBU())
	}
	constants.Logger.InfoLog("server stopped")
}

func main() {

	fmt.Println(fmt.Sprintf("Build version: %s", buildVersion))
	fmt.Println(fmt.Sprintf("Build date: %s", buildDate))
	fmt.Println(fmt.Sprintf("Build commit: %s", buildCommit))

	server := new(server)
	handlers.NewRepStore(&server.storege)
	fmt.Println(server.storege.Config.Address)
	if server.storege.Config.Restore {
		go server.storege.RestoreData()
	}

	go server.storege.BackupData()

	go func() {
		s := &http.Server{
			Addr:    server.storege.Config.Address,
			Handler: server.storege.Router}

		if err := s.ListenAndServe(); err != nil {
			constants.Logger.ErrorLog(err)
			return
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	Shutdown(&server.storege)
}
