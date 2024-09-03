package main

import (
	"context"
	"time"

	"metering/config"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/licensemanager/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type lmServer struct {
	licensemanager.UnimplementedLockServiceServer
	locks []*licensemanager.Lock
}

func newLmServer(c config.Config) *lmServer {
	var locks []*licensemanager.Lock

	start := time.Now()
	end := start.Add(time.Hour)

	for _, l := range c.Locks {
		locks = append(locks, &licensemanager.Lock{
			Id:         fakeId(),
			InstanceId: fakeId(),
			ResourceId: l.ResourceID,
			StartTime:  timestamppb.New(start),
			EndTime:    timestamppb.New(end),
			CreatedAt:  timestamppb.New(start),
			UpdatedAt:  nil,
			State:      licensemanager.Lock_LOCKED,
			TemplateId: l.TemplateID,
		})
	}

	return &lmServer{
		locks: locks,
	}
}

func (s *lmServer) List(context.Context, *licensemanager.ListLocksRequest) (*licensemanager.ListLocksResponse, error) {

	return &licensemanager.ListLocksResponse{
		Locks: s.locks,
	}, nil
}

func (s *lmServer) Ensure(_ context.Context, req *licensemanager.EnsureLockRequest) (*operation.Operation, error) {

	for _, l := range s.locks {
		if l.ResourceId == req.ResourceId && l.InstanceId == req.InstanceId {
			r, _ := anypb.New(l)
			m, _ := anypb.New(&licensemanager.EnsureLockMetadata{LockId: l.Id})
			return &operation.Operation{
				Id:          fakeId(),
				Description: "lock ensured",
				CreatedAt:   nil,
				CreatedBy:   "",
				ModifiedAt:  nil,
				Done:        true,
				Metadata:    m,
				Result:      &operation.Operation_Response{Response: r},
			}, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "lock not found")
}
