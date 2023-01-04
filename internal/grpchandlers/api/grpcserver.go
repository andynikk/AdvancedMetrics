package api

import (
	"context"
	"fmt"

	"github.com/andynikk/advancedmetrics/internal/constants/errs"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"google.golang.org/grpc/metadata"
)

type GRPCServer struct {
	RepStore general.RepStore[grpchandlers.RepStore]
}

func Header(ctx context.Context) general.Header {
	mHeader := make(general.Header)

	mdI, _ := metadata.FromIncomingContext(ctx)
	for key, valMD := range mdI {
		for _, val := range valMD {
			mHeader[key] = val
		}
	}

	mdO, _ := metadata.FromOutgoingContext(ctx)
	for key, valMD := range mdO {
		for _, val := range valMD {
			mHeader[key] = val
		}
	}

	return mHeader
}

func (s *GRPCServer) mustEmbedUnimplementedUpdatersServer() {
	//TODO implement me
	panic("implement me")
}

func (s *GRPCServer) UpdatesJSON(ctx context.Context, req *UpdatesRequest) (*BoolRespons, error) {
	header := Header(ctx)

	err := s.RepStore.HandlerUpdatesMetricJSON(header, req.Body)
	if err != nil {
		return &BoolRespons{Result: false}, nil
	}
	return &BoolRespons{Result: true}, nil

	//sRMQ := new(rabbitmq.SettingRMQ)
	//sRMQ.ConnRMQ()
	//sRMQ.ChannelRMQ()
	//sRMQ.QueueRMQ()
	//sRMQ.MessageRMQ(mHeader, req.Body)
}

func (s *GRPCServer) UpdateJSON(ctx context.Context, req *UpdateStrRequest) (*BoolRespons, error) {

	header := Header(ctx)

	err := s.RepStore.HandlerUpdateMetricJSON(header, req.Body)
	if err != nil {
		return &BoolRespons{Result: false}, err
	}
	return &BoolRespons{Result: true}, nil
}

func (s *GRPCServer) Update(ctx context.Context, req *UpdateRequest) (*BoolRespons, error) {

	//md, _ := metadata.FromIncomingContext(ctx)
	//header := Header(md)

	err := s.RepStore.HandlerSetMetricaPOST(string(req.MetType), string(req.MetName), string(req.MetValue))
	if err != nil {
		return &BoolRespons{Result: false}, err
	}
	return &BoolRespons{Result: true}, nil

}

func (s *GRPCServer) Ping(ctx context.Context, req *EmtyRequest) (*BoolRespons, error) {
	header := Header(ctx)

	err := s.RepStore.HandlerPingDB(header)
	if err != nil {
		return &BoolRespons{Result: false}, nil
	}
	return &BoolRespons{Result: true}, nil
}

func (s *GRPCServer) ValueJSON(ctx context.Context, req *UpdatesRequest) (*FullRespons, error) {

	header := Header(ctx)

	h, body, err := s.RepStore.HandlerValueMetricaJSON(header, req.Body)
	ok := true
	if err != nil {
		ok = false
	}

	var hdr string
	for k, v := range h {
		hdr += fmt.Sprintf("%s:%s\n", k, v)
	}

	return &FullRespons{Header: []byte(hdr), Body: body, Result: ok}, err
}

func (s *GRPCServer) Value(ctx context.Context, req *UpdatesRequest) (*StatusRespons, error) {

	//md, _ := metadata.FromIncomingContext(ctx)
	//header := Header(md)

	val, err := s.RepStore.HandlerGetValue(req.Body)
	return &StatusRespons{Result: []byte(val), Status: errs.StatusError(err)}, err

}

func (s *GRPCServer) ListMetrics(ctx context.Context, req *EmtyRequest) (*StatusRespons, error) {

	header := Header(ctx)

	_, val := s.RepStore.HandlerGetAllMetrics(header)
	return &StatusRespons{Result: val, Status: 200}, nil

}
