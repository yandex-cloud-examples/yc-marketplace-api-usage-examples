package main

import (
	"log"
	"time"

	"metering/config"
)

const gracePeriod = time.Hour

type usageRecord struct {
	Uuid      string
	SkuId     string
	Quantity  int64
	Timestamp time.Time
}

type writeRequest struct {
	ValidateOnly bool
	// Marketplace Product's ID.
	ProductId string
	// List of product usage records (up to 25 per request).
	UsageRecords []*usageRecord
}

type AcceptedUsageRecord struct {
	// UUID of the accepted product usage record.
	Uuid string
}

type Reason int32

const (
	REASON_UNSPECIFIED Reason = 0
	DUPLICATE          Reason = 1
	EXPIRED            Reason = 2
	INVALID_TIMESTAMP  Reason = 3
	INVALID_SKU_ID     Reason = 4
	INVALID_PRODUCT_ID Reason = 5
	INVALID_QUANTITY   Reason = 6
	INVALID_ID         Reason = 7
)

type RejectedUsageRecord struct {
	// UUID of the rejected product usage record.
	Uuid string
	// Reason of rejection.
	Reason Reason
}

type writeResponse struct {
	// List of accepted product usage records.
	Accepted []*AcceptedUsageRecord
	// List of rejected product usage records (with reason).
	Rejected []*RejectedUsageRecord
}

type meteringService struct {
	mode      config.WorkMode
	skuMap    map[string]string
	metricIDs map[string]struct{}
}

func (ms meteringService) makeResponse(request *writeRequest) *writeResponse {
	var accepted []*AcceptedUsageRecord
	var rejected []*RejectedUsageRecord
	for _, record := range request.UsageRecords {

		valid, reason := ms.checkBinding(record.SkuId, request.ProductId)
		if !valid {
			rejected = append(rejected, &RejectedUsageRecord{
				Uuid:   record.Uuid,
				Reason: reason,
			})
			continue
		}
		valid = ms.validateId(record, request.ValidateOnly)
		if !valid {
			rejected = append(rejected, &RejectedUsageRecord{
				Uuid:   record.Uuid,
				Reason: DUPLICATE,
			})
			continue
		}
		valid, reason = ms.validateTimestamp(record)
		if !valid {
			rejected = append(rejected, &RejectedUsageRecord{
				Uuid:   record.Uuid,
				Reason: reason,
			})
			continue
		}
		valid = ms.validateQuantity(record)
		if !valid {
			rejected = append(rejected, &RejectedUsageRecord{
				Uuid:   record.Uuid,
				Reason: INVALID_QUANTITY,
			})
			continue
		}
		accepted = append(accepted, &AcceptedUsageRecord{
			Uuid: record.Uuid,
		})
	}

	log.Printf("Accepted: %v, Rejected: %v", accepted, rejected)

	return &writeResponse{
		Accepted: accepted,
		Rejected: rejected,
	}
}

func (ms meteringService) validateId(record *usageRecord, dryRun bool) bool {
	_, ok := ms.metricIDs[record.Uuid]
	if !ok {
		if !dryRun {
			ms.metricIDs[record.Uuid] = struct{}{}
		}
		return true
	}
	return false
}

func (ms meteringService) checkBinding(skuID string, productID string) (bool, Reason) {
	if pid, ok := ms.skuMap[skuID]; !ok {
		return false, INVALID_SKU_ID
	} else if pid != productID {
		return false, INVALID_PRODUCT_ID
	}
	return true, REASON_UNSPECIFIED
}

func (ms meteringService) validateTimestamp(record *usageRecord) (bool, Reason) {
	now := time.Now()
	startTime := now.Add(-gracePeriod)
	endTime := now.Add(time.Minute)
	ts := record.Timestamp
	if ts.Before(startTime) {
		return false, EXPIRED
	}
	if ts.After(endTime) {
		return false, INVALID_TIMESTAMP
	}
	return true, REASON_UNSPECIFIED
}

func (ms meteringService) validateQuantity(record *usageRecord) bool {
	return record.Quantity > 0
}

func newMeteringService(config config.Config) *meteringService {
	skuMap := make(map[string]string)
	for _, binding := range config.SkuBindings {
		skuMap[binding.SkuID] = binding.ProductID
	}
	return &meteringService{
		mode:      config.Mode,
		skuMap:    skuMap,
		metricIDs: map[string]struct{}{},
	}
}
