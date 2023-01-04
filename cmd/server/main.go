// Start of the service for getting metrics.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/handlers/api"
)

type server struct {
	storege handlers.RepStore
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {

	fmt.Println(fmt.Sprintf("Build version: %s", buildVersion))
	fmt.Println(fmt.Sprintf("Build date: %s", buildDate))
	fmt.Println(fmt.Sprintf("Build commit: %s", buildCommit))

	server := new(server)
	api.NewRepStore(&server.storege)
	fmt.Println(server.storege.Config.Address)

	gRepStore := general.New[handlers.RepStore]()
	gRepStore.Set(constants.TypeSrvHTTP.String(), server.storege)

	srv := api.HTTPServer{
		RepStore: gRepStore,
	}

	api.InitRoutersMux(&srv)

	if server.storege.Config.Restore {
		go srv.RepStore.RestoreData()
	}

	go srv.RepStore.BackupData()

	go func() {
		s := &http.Server{
			Addr:    server.storege.Config.Address,
			Handler: srv.Router,
		}

		if err := s.ListenAndServe(); err != nil {
			constants.Logger.ErrorLog(err)
			return
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	gRepStore.Shutdown()
}
