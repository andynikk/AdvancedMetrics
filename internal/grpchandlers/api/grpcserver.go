package api

import (
	"bufio"
	"context"
	"strings"

	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
)

type GRPCServer struct {
	RepStore general.RepStore[grpchandlers.RepStore]
}

func (s *GRPCServer) mustEmbedUnimplementedUpdatersServer() {
	//TODO implement me
	panic("implement me")
}

func (s *GRPCServer) Update(ctx context.Context, req *UpdateRequest) (*UpdateRespons, error) {
	mHeader := make(general.Header)
	strHeader := string(req.Header)

	scanner := bufio.NewScanner(strings.NewReader(strHeader))
	for scanner.Scan() {
		strH := scanner.Text()
		arrH := strings.Split(strH, ":")
		if len(arrH) != 2 {
			continue
		}
		mHeader[arrH[0]] = arrH[1]
	}

	//sRMQ := new(rabbitmq.SettingRMQ)
	//sRMQ.ConnRMQ()
	//sRMQ.ChannelRMQ()
	//sRMQ.QueueRMQ()
	//sRMQ.MessageRMQ(mHeader, req.Body)

	body := req.Body
	err := s.RepStore.HandlerUpdatesMetricJSON(mHeader, body)
	if err != nil {
		return &UpdateRespons{Result: false}, nil
	}
	return &UpdateRespons{Result: true}, nil
}
