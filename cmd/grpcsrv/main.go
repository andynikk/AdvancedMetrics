package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers/api"
	"google.golang.org/grpc"
)

type server struct {
	storege grpchandlers.RepStore
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {

	fmt.Println(fmt.Sprintf("Build version: %s", buildVersion))
	fmt.Println(fmt.Sprintf("Build date: %s", buildDate))
	fmt.Println(fmt.Sprintf("Build commit: %s", buildCommit))

	server := new(server)
	grpchandlers.NewRepStore(&server.storege)
	fmt.Println(server.storege.Config.Address)

	gRepStore := general.New[grpchandlers.RepStore]()
	gRepStore.Set("grpchandlers", server.storege)

	if server.storege.Config.Restore {
		go gRepStore.RestoreData()
	}

	go gRepStore.BackupData()

	s := grpc.NewServer()
	srv := &api.GRPCServer{
		gRepStore,
	}
	api.RegisterUpdatersServer(s, srv)
	l, err := net.Listen("tcp", constants.AddressServer)
	if err != nil {
		log.Fatal(err)
	}

	go func() {

		if err = s.Serve(l); err != nil {
			log.Fatal(err)
		}

	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	gRepStore.Shutdown()
}
