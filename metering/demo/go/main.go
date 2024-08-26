package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/v1/metering"
	"github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// basepath is the root directory of this package.
var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

// buildProductUsageWriteRequest builds a product usage write request.
func buildProductUsageWriteRequest(
	productID string,
	skuID string,
	quantity int,
	timestamp time.Time,
	uuid string,
) metering.WriteImageProductUsageRequest {
	return metering.WriteImageProductUsageRequest{
		// ProductId is the identifier of the product.
		ProductId: productID,
		// UsageRecords is a list of usage records.
		// Each record contains a unique identifier, SKU identifier, quantity, and timestamp.
		// You can provide multiple records in a single request. Records can contain different SKUs.
		UsageRecords: []*metering.UsageRecord{
			{
				// Uuid is a unique identifier of the usage record.
				Uuid: uuid,
				// SkuId is the identifier of the SKU.
				SkuId: skuID,
				// Quantity is the amount of the product used. The unit of measurement is defined by the SKU.
				// Must be greater or equal to 0.
				Quantity: int64(quantity),
				// Timestamp is the time when the product was used.
				Timestamp: timestamppb.New(timestamp),
			},
		},
	}
}

// businessLogic is a placeholder for the actual business logic of your application.
func businessLogic(productID, skuID string) int {
	if productID == "Secure Firewall" && skuID == "Ingress network traffic" {
		return 1 + 1
	}

	if productID == "Secure Firewall" && skuID == "Egress network traffic" {
		return 1 * 1
	}

	return 0
}

// run is the function that writes product usage to Yandex.Cloud API.
func run(productID string, skuID string, quantity int, timestamp string, uuidStr string, fake bool, serviceAccountKey string) string {
	if productID == "" || skuID == "" || quantity == 0 {
		log.Fatalf("product-id, sku-id, and quantity are required")
	}
	ts := time.Now()
	var err error
	if timestamp != "" {
		ts, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			log.Fatalf("Invalid timestamp. Please provide a valid RFC3339 timestamp.")
		}
	}

	if uuidStr == "" {
		uuidStr = uuid.New().String()
	}

	// If we provide empty apiEndpoint, SDK will use default endpoint
	var apiEndpoint string
	var tlsConfig *tls.Config = nil
	// Use fake endpoint if fake flag is set
	if fake {
		apiEndpoint = "api.yc.local:8080"

		// Get CA cert file name
		caFilename := path.Join(basepath, "../../fake/x509/ca.crt")

		// Load system cert pool
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}

		// Read in the CA cert file
		certs, err := os.ReadFile(caFilename)
		if err != nil {
			log.Fatalf("Failed to append %q to RootCAs: %v", caFilename, err)
		}

		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			log.Println("No certs appended, using system certs only")
		}

		// Use the system cert pool in our tls.Config
		tlsConfig = &tls.Config{
			RootCAs:            rootCAs,
			InsecureSkipVerify: true,
		}
	}

	var creds ycsdk.Credentials

	// Use service account key if provided.
	if serviceAccountKey != "" {
		// Read service account key
		saData, err := os.ReadFile(serviceAccountKey)
		if err != nil {
			log.Fatalf("Unable to read service account key: %v", err)
		}
		var saKey iamkey.Key

		err = json.Unmarshal(saData, &saKey)
		if err != nil {
			log.Fatalf("Unable to unmarshal service account key: %v", err)
		}
		// Create sdk credentials
		creds, err = ycsdk.ServiceAccountKey(&saKey)
		if err != nil {
			log.Fatalf("Unable to create sdk credentials: %v", err)
		}
	} else {
		// If no service account key provided, use instance service account will be used.
		// But it would work only on Yandex.Cloud VMs.
		creds = ycsdk.InstanceServiceAccount()
		if err != nil {
			log.Fatalf("Unable to create sdk credentials: %v", err)
		}
	}

	ctx := context.Background()
	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: creds,
		Endpoint:    apiEndpoint,
		TLSConfig:   tlsConfig,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Create a client
	client := sdk.Marketplace().Metering().ImageProductUsage()

	// Build the request
	request := buildProductUsageWriteRequest(productID, skuID, 1, ts, uuidStr)

	// Step 0. Ensure consumer has all permissions to use the product (validate_only=True)
	request.ValidateOnly = true
	response, err := client.Write(ctx, &request)
	if err != nil {
		log.Fatalf("could not write: %v", err)
	}
	if len(response.Accepted) == 0 {
		log.Fatalf("Unable to provide the service to customer. Got empty list of accepted metrics.")
	}

	// Step 1. Provide your service to the customer
	businessLogic(productID, skuID)

	// Step 2. Write the product usage to Yandex.Cloud API (validate_only=False)
	request.ValidateOnly = false
	response, err = client.Write(ctx, &request)
	if err != nil {
		if status.Code(err) == codes.InvalidArgument {
			log.Fatalf("Invalid argument: %v", err)
		}
	}
	if len(response.Accepted) == 0 {
		log.Fatalf("Unable to provide the service to customer. Got empty list of accepted metrics.")
	}

	var jsonResp []byte

	jsonResp, err = json.Marshal(response)
	if err != nil {
		log.Fatalf("Unable to marshal response: %v", err)
	}

	return string(jsonResp)
}

func main() {
	// Define flags
	productID := flag.String("product-id", "", "Marketplace image product ID")
	skuID := flag.String("sku-id", "", "Marketplace image product SKU")
	quantity := flag.Int("quantity", 0, "Usage quantity")
	timestamp := flag.String("timestamp", "", "Usage time")
	uuidStr := flag.String("uuid", "", "Usage request unique identifier")
	fake := flag.Bool("fake", false, "Use local fake endpoint")
	serviceAccountKey := flag.String("service-account-key", "", "Service account key")

	// Parse the flags
	flag.Parse()

	// Call the main function
	fmt.Println(
		run(
			*productID,
			*skuID,
			*quantity,
			*timestamp,
			*uuidStr,
			*fake,
			*serviceAccountKey,
		),
	)
}
