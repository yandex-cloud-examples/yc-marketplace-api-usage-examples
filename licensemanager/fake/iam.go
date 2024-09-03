package main

import (
	"context"
	"time"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type iamServer struct {
	iam.UnimplementedIamTokenServiceServer
}

func newIamServer() *iamServer {
	return &iamServer{}
}

func (s *iamServer) Create(context.Context, *iam.CreateIamTokenRequest) (*iam.CreateIamTokenResponse, error) {

	return &iam.CreateIamTokenResponse{
		IamToken: "fake_token",
		ExpiresAt: &timestamppb.Timestamp{
			Seconds: time.Now().Add(time.Hour).Unix(),
		},
	}, nil
}
