package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"metering/config"

	"github.com/yandex-cloud/go-genproto/yandex/cloud/endpoint"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/iam/v1"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/marketplace/licensemanager/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// basepath is the root directory of this package.
var basepath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	basepath = filepath.Dir(currentFile)
}

func main() {

	c, err := config.FromEnv()
	if err != nil {
		panic(err)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", c.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	certFile := filepath.Join(basepath, "x509/server.crt")
	keyFile := filepath.Join(basepath, "x509/server.pem")

	if os.Getenv("PLAINTEXT") != "" {
		opts = []grpc.ServerOption{}
	} else {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials: %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)

	endpoint.RegisterApiEndpointServiceServer(grpcServer, newEndpointsServer(*c))
	iam.RegisterIamTokenServiceServer(grpcServer, newIamServer())
	licensemanager.RegisterLockServiceServer(grpcServer, newLmServer(*c))
	reflection.Register(grpcServer)
	log.Printf("Starting server on port %d", c.Port)
	err = grpcServer.Serve(lis)
	if err != nil {
		panic(err)
	}
}
