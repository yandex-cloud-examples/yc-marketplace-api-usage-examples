package metering

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/metering/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client provides methods to interact with the Yandex Cloud Metering API.
type Client struct {
	sdk          *ycsdk.SDK
	logger       *slog.Logger
	defaultSkuID string // Added to store the default SKU ID from config
}

// NewClient creates a new metering client.
// It requires a Yandex Cloud SDK instance, a structured logger, and a default SKU ID.
func NewClient(sdk *ycsdk.SDK, logger *slog.Logger, defaultSkuID string) *Client {
	return &Client{
		sdk:          sdk,
		logger:       logger,
		defaultSkuID: defaultSkuID, // Store the default SKU ID
	}
}

// ReportUsageParameters holds parameters for reporting usage.
type ReportUsageParameters struct {
	ProductInstanceID string // The ID of the product instance to report usage for.
	SkuID             string // The SKU ID for the usage. If empty, a default demo SKU will be used.
	Amount            int    // The quantity of usage to report.
	UserID            string // Identifier for the user, used for logging and context.
}

// ReportUsage sends a usage record to the Yandex Cloud Metering service.
// It returns the UUID of the accepted usage record or an error if the operation fails or the record is rejected.
func (c *Client) ReportUsage(ctx context.Context, params ReportUsageParameters) (string, error) {
	recordID, err := uuid.NewUUID()
	if err != nil {
		c.logger.ErrorContext(ctx, "Could not generate UUID for usage record", slog.String("error", err.Error()), slog.String("user_id", params.UserID))
		return "", fmt.Errorf("could not generate UUID for usage record: %w", err)
	}

	skuID := params.SkuID
	if skuID == "" {
		skuID = c.defaultSkuID // Use the configured default SKU ID
		c.logger.DebugContext(ctx, "Using configured default SKU ID for metering", slog.String("sku_id", skuID), slog.String("user_id", params.UserID), slog.String("product_instance_id", params.ProductInstanceID))
	}

	writeUsageReq := &metering.WriteUsageRequest{
		ProductInstanceId: params.ProductInstanceID,
		UsageRecords: []*metering.UsageRecord{
			{
				Uuid:     recordID.String(),
				SkuId:    skuID,
				Quantity: int64(params.Amount),
				Timestamp: &timestamppb.Timestamp{
					Seconds: time.Now().Unix(),
				},
			},
		},
	}

	c.logger.InfoContext(ctx, "Attempting to write usage record",
		slog.String("user_id", params.UserID),
		slog.String("product_instance_id", params.ProductInstanceID),
		slog.String("sku_id", skuID),
		slog.Int("amount", params.Amount),
		slog.String("usage_record_id", recordID.String()),
	)

	resp, err := c.sdk.Marketplace().Metering().ProductUsage().Write(ctx, writeUsageReq)
	if err != nil {
		grpcStatus, _ := status.FromError(err)
		c.logger.ErrorContext(ctx, "Failed to write usage record to metering API",
			slog.String("error", err.Error()),
			slog.String("grpc_status_code", grpcStatus.Code().String()),
			slog.String("user_id", params.UserID),
			slog.String("product_instance_id", params.ProductInstanceID),
		)
		return "", fmt.Errorf("metering API call failed: %w", err)
	}

	if len(resp.Rejected) > 0 {
		rejectedItem := resp.Rejected[0]
		rejectedReason := "unknown"
		rejectedItemUuid := "unknown"
		if rejectedItem != nil {
			rejectedReason = rejectedItem.Reason.String()
			rejectedItemUuid = rejectedItem.Uuid
		}
		c.logger.ErrorContext(ctx, "Usage record rejected by metering service",
			slog.String("user_id", params.UserID),
			slog.String("product_instance_id", params.ProductInstanceID),
			slog.String("sku_id", skuID),
			slog.Int("amount", params.Amount),
			slog.String("rejected_uuid", rejectedItemUuid),
			slog.String("rejected_reason", rejectedReason),
			slog.Any("all_rejected_records_details", resp.Rejected), // Log all for full context
		)
		// Return an error that includes the reason and UUID of the rejected record.
		return "", fmt.Errorf("usage record (UUID: %s) rejected: %s", rejectedItemUuid, rejectedReason)
	}

	if len(resp.Accepted) > 0 && resp.Accepted[0].Uuid == recordID.String() {
		acceptedRecordID := resp.Accepted[0].Uuid
		c.logger.InfoContext(ctx, "Metering API confirmed acceptance of usage record",
			slog.String("user_id", params.UserID),
			slog.String("product_instance_id", params.ProductInstanceID),
			slog.String("accepted_uuid", acceptedRecordID),
		)
		return acceptedRecordID, nil
	} else if len(resp.Accepted) > 0 {
		// This case (accepted UUID mismatch) should be rare if sending one record.
		acceptedRecordID := resp.Accepted[0].Uuid
		c.logger.WarnContext(ctx, "Metering API accepted a record, but UUID does not match sent UUID",
			slog.String("user_id", params.UserID),
			slog.String("sent_uuid", recordID.String()),
			slog.String("accepted_uuid", acceptedRecordID),
		)
		return acceptedRecordID, nil // Still return accepted UUID
	}

	// If no rejections and no explicit acceptance matching the UUID, this is an ambiguous state.
	c.logger.WarnContext(ctx, "Usage record was not rejected, but no explicit acceptance message matching the sent UUID was found.",
		slog.String("user_id", params.UserID),
		slog.String("sent_usage_record_id", recordID.String()),
		slog.Any("api_response_accepted", resp.Accepted),
	)
	// Depending on strictness, this could be an error or a tentative success.
	// For now, let's consider it a non-error if not rejected, but log a warning.
	return recordID.String(), nil // Return original record ID, but with a warning logged.
}
