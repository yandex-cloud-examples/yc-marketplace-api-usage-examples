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

	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/licensemanager/v1"
	"github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// basepath is the root directory of this package.
var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

var knownTemplates = map[string]string{
	"template-1": "Basic Subscription",
	"template-2": "Advanced Subscription",
}

// run is the function that writes product usage to Yandex.Cloud API.
func run(
	resourceID string,
	folderID string,
	fake bool,
	serviceAccountKey string,
) string {
	if resourceID == "" || folderID == "" {
		log.Fatalf("Resource ID and Folder ID must be provided")
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
	client := sdk.Marketplace().LicenseManager().Lock()

	// Build the request
	request := licensemanager.ListLocksRequest{
		FolderId:   folderID,
		ResourceId: resourceID,
	}

	// Step 1. List locks
	response, err := client.List(ctx, &request)
	if err != nil {
		log.Fatalf("could not write: %v", err)
	}

	var lock *licensemanager.Lock
	// Step 2. Find the lock for known template
	for _, l := range response.Locks {
		// if l template is known
		if sku, ok := knownTemplates[l.TemplateId]; ok {
			log.Printf("Lock found: %v, it corresponds to SKU: %v", l.Id, sku)
			lock = l
		}
	}

	if lock == nil {
		log.Fatalf("could not find lock for known template")
	}

	// Step 3. Ensure l while program is running
	for i := 0; i < 5; i++ {
		// Build the request
		request := licensemanager.EnsureLockRequest{
			ResourceId: lock.ResourceId,
			InstanceId: lock.InstanceId,
		}

		// Ensure lock
		lock, err := client.Ensure(ctx, &request)
		if err != nil {
			log.Fatalf("could not ensure l: %v", err)
		}

		log.Printf("Lock ensured: %v", lock.Id)

		// Sleep for a while
		time.Sleep(1 * time.Second)
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
	resourceID := flag.String("resource-id", "", "Resource ID")
	folderID := flag.String("folder-id", "", "Folder ID")
	fake := flag.Bool("fake", false, "Use local fake endpoint")
	serviceAccountKey := flag.String("service-account-key", "", "Service account key")

	// Parse the flags
	flag.Parse()

	// Call the main function
	fmt.Println(
		run(
			*resourceID,
			*folderID,
			*fake,
			*serviceAccountKey,
		),
	)
}
