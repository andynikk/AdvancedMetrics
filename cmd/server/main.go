// Start of the service for getting metrics.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	api2 "github.com/andynikk/advancedmetrics/internal/grpchandlers/api"
	"github.com/andynikk/advancedmetrics/internal/handlers"
	"github.com/andynikk/advancedmetrics/internal/handlers/api"
	"github.com/andynikk/advancedmetrics/internal/middlware"
)

type serverHTTP struct {
	storage handlers.RepStore
	srv     api.HTTPServer
}

type serverGRPS struct {
	storage grpchandlers.RepStore
	srv     api2.GRPCServer
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

type Server interface {
	Start() error
	RestoreData()
	BackupData()
	Shutdown()
}

func (s *serverHTTP) Start() error {
	fmt.Println("++++++++++++++14 Address", s.storage.Config.Address)
	HTTPServer := &http.Server{
		Addr: s.storage.Config.Address,
		//Handler: &s.srv.Router,
		Handler: s.srv.RouterChi,
	}

	if err := HTTPServer.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func (s *serverGRPS) Start() error {

	server := grpc.NewServer(middlware.WithServerUnaryInterceptor())
	api2.RegisterMetricCollectorServer(server, &s.srv)
	l, err := net.Listen("tcp", constants.AddressServer)
	if err != nil {
		return err
	}

	if err = server.Serve(l); err != nil {
		return err
	}

	return nil
}

func (s *serverHTTP) RestoreData() {
	if s.storage.Config.Restore {
		s.srv.RepStore.RestoreData()
	}
}

func (s *serverGRPS) RestoreData() {
	if s.storage.Config.Restore {
		s.srv.RepStore.RestoreData()
	}
}

func (s *serverHTTP) BackupData() {
	s.srv.RepStore.BackupData()
}

func (s *serverGRPS) BackupData() {
	s.srv.RepStore.BackupData()
}

func (s *serverHTTP) Shutdown() {
	s.srv.RepStore.Shutdown()
}

func (s *serverGRPS) Shutdown() {
	s.srv.RepStore.Shutdown()
}

func newHTTPServer(configServer *environment.ServerConfig) *serverHTTP {

	server := new(serverHTTP)

	server.storage.Config = configServer
	server.storage.PK, _ = encryption.InitPrivateKey(configServer.CryptoKey)
	api.NewRepStore(&server.storage)
	fmt.Println(&server.storage.Config.Address)

	gRepStore := general.New[handlers.RepStore]()
	gRepStore.Set(constants.TypeSrvHTTP.String(), server.storage)
	srv := api.HTTPServer{
		RepStore: gRepStore,
	}
	//api.InitRoutersMux(&srv)
	api.InitRoutersChi(&srv)
	server.srv = srv

	return server
}

func newGRPCServer(configServer *environment.ServerConfig) *serverGRPS {
	server := new(serverGRPS)

	server.storage.Config = configServer
	server.storage.PK, _ = encryption.InitPrivateKey(configServer.CryptoKey)

	grpchandlers.NewRepStore(&server.storage)
	fmt.Println(server.storage.Config.Address)

	gRepStore := general.New[grpchandlers.RepStore]()
	gRepStore.Set(constants.TypeSrvGRPC.String(), server.storage)

	srv := &api2.GRPCServer{
		RepStore: gRepStore,
	}
	server.srv = *srv

	return server
}

// NewServer реализует фабричный метод.
func NewServer(configServer *environment.ServerConfig) Server {
	fmt.Println("+++++++++++++", configServer.TypeServer)
	if configServer.TypeServer == constants.TypeSrvGRPC.String() {
		return newGRPCServer(configServer)
	}

	return newHTTPServer(configServer)
}

func main() {

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	fmt.Println("================1")
	config := environment.InitConfigServer()
	fmt.Println("================2")
	server := NewServer(config)
	fmt.Println("================3")
	go server.RestoreData()
	fmt.Println("================4")
	go server.BackupData()
	fmt.Println("================5")

	go func() {
		fmt.Println("================6")
		err := server.Start()
		fmt.Println("================7")
		if err != nil {
			constants.Logger.ErrorLog(err)
			return
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	server.Shutdown()
}
