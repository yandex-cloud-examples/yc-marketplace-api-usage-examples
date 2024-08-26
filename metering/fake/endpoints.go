package main

import (
	"context"
	"fmt"
	"slices"

	"metering/config"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/endpoint"
)

var endpointIDs = []string{"marketplace", "iam"}

type endpointsServer struct {
	endpoint.UnimplementedApiEndpointServiceServer
	c config.Config
}

func newEndpointsServer(c config.Config) *endpointsServer {
	return &endpointsServer{
		c: c,
	}
}

func (e endpointsServer) List(context.Context, *endpoint.ListApiEndpointsRequest) (*endpoint.ListApiEndpointsResponse, error) {

	es := []*endpoint.ApiEndpoint{}

	for _, id := range endpointIDs {
		es = append(es, &endpoint.ApiEndpoint{
			Id:      id,
			Address: e.c.Address(),
		})
	}

	return &endpoint.ListApiEndpointsResponse{
		Endpoints: es,
	}, nil
}

func (e endpointsServer) Get(_ context.Context, req *endpoint.GetApiEndpointRequest) (*endpoint.ApiEndpoint, error) {
	if slices.Contains(endpointIDs, req.ApiEndpointId) {
		return &endpoint.ApiEndpoint{
			Id:      req.ApiEndpointId,
			Address: e.c.Address(),
		}, nil
	}
	return nil, fmt.Errorf("endpoint not found")
}
