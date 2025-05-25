package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"github.com/yandex-cloud/go-sdk/pkg/requestid"
	"google.golang.org/grpc"

	"demo/pkg/db"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	repo, err := db.NewRepo()
	if err != nil {
		log.Fatalf("Could not connect to database: %s\n", err)
	}

	var credentials ycsdk.Credentials
	saKeyFile := os.Getenv("YC_SA_KEY_FILE")
	if saKeyFile != "" {
		fileData, err := os.ReadFile(saKeyFile)
		if err != nil {
			log.Fatalf("Could not read service account key file: %s\n", err)
			return
		}
		var saKey iamkey.Key
		if err := saKey.UnmarshalJSON(fileData); err != nil {
			log.Fatalf("Could not unmarshal service account key: %s\n", err)
			return
		}
		credentials, err = ycsdk.ServiceAccountKey(&saKey)
		if err != nil {
			log.Fatalf("Could not create service account credentials: %s\n", err)
			return
		}
	} else {
		credentials = ycsdk.InstanceServiceAccount()
		log.Println("Using instance service account credentials")
	}

	cfg := ycsdk.Config{
		Credentials: credentials,
	}
	ctx := context.Background()
	sdk, err := ycsdk.Build(
		ctx,
		cfg,
		grpc.WithUnaryInterceptor(requestid.Interceptor()),
	)
	if err != nil {
		log.Fatal("SDK init error: " + err.Error())
	}

	s := NewServer(repo, sdk)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/register", s.registerHandler)
	r.Post("/register", s.registerPostHandler)

	r.Get("/login", s.loginHandler)
	r.Post("/login", s.loginPostHandler)

	r.Get("/logout", s.logoutHandler)

	r.Post("/bind", s.bindPostHandler)

	r.Post("/report", s.reportPostHandler)

	r.Get("/", s.indexHandler)

	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
