package main

import (
	"context"

	"metering/config"

	current "github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/metering/v1"
	deprecated "github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/v1/metering"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type currentMarketplaceServer struct {
	current.UnimplementedImageProductUsageServiceServer
	ms *meteringService
}

func (m currentMarketplaceServer) Write(_ context.Context, request *current.WriteImageProductUsageRequest) (*current.WriteImageProductUsageResponse, error) {
	switch m.ms.mode {
	case config.ALWAYS:
		return m.makeResponse(request), nil
	case config.NEVER:
		return nil, status.Errorf(codes.PermissionDenied, "not authorized")
	case config.DRYRUN:
		if request.ValidateOnly {
			return m.makeResponse(request), nil
		} else {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
	}
	return nil, status.Error(codes.Internal, "unknown mode")
}

func (m currentMarketplaceServer) makeResponse(request *current.WriteImageProductUsageRequest) *current.WriteImageProductUsageResponse {
	var ur []*usageRecord
	for _, record := range request.UsageRecords {
		ur = append(ur, &usageRecord{
			Uuid:      record.Uuid,
			SkuId:     record.SkuId,
			Quantity:  record.Quantity,
			Timestamp: record.Timestamp.AsTime(),
		})
	}

	r := writeRequest{
		ValidateOnly: request.ValidateOnly,
		ProductId:    request.ProductId,
		UsageRecords: ur,
	}
	res := m.ms.makeResponse(&r)
	var accepted []*current.AcceptedUsageRecord
	var rejected []*current.RejectedUsageRecord

	for _, record := range res.Accepted {
		accepted = append(accepted, &current.AcceptedUsageRecord{
			Uuid: record.Uuid,
		})
	}
	for _, record := range res.Rejected {
		rejected = append(rejected, &current.RejectedUsageRecord{
			Uuid:   record.Uuid,
			Reason: m.mapReason(record.Reason),
		})
	}
	return &current.WriteImageProductUsageResponse{
		Accepted: accepted,
		Rejected: rejected,
	}
}

func (m currentMarketplaceServer) mapReason(reason Reason) current.RejectedUsageRecord_Reason {
	switch reason {
	case DUPLICATE:
		return current.RejectedUsageRecord_DUPLICATE
	case EXPIRED:
		return current.RejectedUsageRecord_EXPIRED
	case INVALID_TIMESTAMP:
		return current.RejectedUsageRecord_INVALID_TIMESTAMP
	case INVALID_SKU_ID:
		return current.RejectedUsageRecord_INVALID_SKU_ID
	case INVALID_PRODUCT_ID:
		return current.RejectedUsageRecord_INVALID_PRODUCT_ID
	case INVALID_QUANTITY:
		return current.RejectedUsageRecord_INVALID_QUANTITY
	case INVALID_ID:
		return current.RejectedUsageRecord_INVALID_ID
	default:
		return current.RejectedUsageRecord_REASON_UNSPECIFIED
	}
}

func newCurrentMarketplaceServer(ms *meteringService) current.ImageProductUsageServiceServer {
	return &currentMarketplaceServer{
		ms: ms,
	}
}

type deprecatedMarketplaceServer struct {
	current.UnimplementedImageProductUsageServiceServer
	ms *meteringService
}

func (m deprecatedMarketplaceServer) Write(_ context.Context, request *deprecated.WriteImageProductUsageRequest) (*deprecated.WriteImageProductUsageResponse, error) {
	switch m.ms.mode {
	case config.ALWAYS:
		return m.makeResponse(request), nil
	case config.NEVER:
		return nil, status.Errorf(codes.PermissionDenied, "not authorized")
	case config.DRYRUN:
		if request.ValidateOnly {
			return m.makeResponse(request), nil
		} else {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
	}
	return nil, status.Error(codes.Internal, "unknown mode")
}

func (m deprecatedMarketplaceServer) makeResponse(request *deprecated.WriteImageProductUsageRequest) *deprecated.WriteImageProductUsageResponse {
	var ur []*usageRecord
	for _, record := range request.UsageRecords {
		ur = append(ur, &usageRecord{
			Uuid:      record.Uuid,
			SkuId:     record.SkuId,
			Quantity:  record.Quantity,
			Timestamp: record.Timestamp.AsTime(),
		})
	}

	r := writeRequest{
		ValidateOnly: request.ValidateOnly,
		ProductId:    request.ProductId,
		UsageRecords: ur,
	}
	res := m.ms.makeResponse(&r)
	var accepted []*deprecated.AcceptedUsageRecord
	var rejected []*deprecated.RejectedUsageRecord

	for _, record := range res.Accepted {
		accepted = append(accepted, &deprecated.AcceptedUsageRecord{
			Uuid: record.Uuid,
		})
	}
	for _, record := range res.Rejected {
		rejected = append(rejected, &deprecated.RejectedUsageRecord{
			Uuid:   record.Uuid,
			Reason: m.mapReason(record.Reason),
		})
	}
	return &deprecated.WriteImageProductUsageResponse{
		Accepted: accepted,
		Rejected: rejected,
	}
}

func (m deprecatedMarketplaceServer) mapReason(reason Reason) deprecated.RejectedUsageRecord_Reason {
	switch reason {
	case DUPLICATE:
		return deprecated.RejectedUsageRecord_DUPLICATE
	case EXPIRED:
		return deprecated.RejectedUsageRecord_EXPIRED
	case INVALID_TIMESTAMP:
		return deprecated.RejectedUsageRecord_INVALID_TIMESTAMP
	case INVALID_SKU_ID:
		return deprecated.RejectedUsageRecord_INVALID_SKU_ID
	case INVALID_PRODUCT_ID:
		return deprecated.RejectedUsageRecord_INVALID_PRODUCT_ID
	case INVALID_QUANTITY:
		return deprecated.RejectedUsageRecord_INVALID_QUANTITY
	case INVALID_ID:
		return deprecated.RejectedUsageRecord_INVALID_ID
	default:
		return deprecated.RejectedUsageRecord_REASON_UNSPECIFIED
	}
}

func newDeprecatedMarketplaceServer(ms *meteringService) deprecated.ImageProductUsageServiceServer {
	return &deprecatedMarketplaceServer{
		ms: ms,
	}
}
