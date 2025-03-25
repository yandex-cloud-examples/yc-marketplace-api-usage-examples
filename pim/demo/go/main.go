package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ycsdk "github.com/yandex-cloud/go-sdk"

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

	cfg := ycsdk.Config{
		Credentials: ycsdk.InstanceServiceAccount(),
	}
	ctx := context.Background()
	sdk, err := ycsdk.Build(ctx, cfg)
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

	r.Get("/", s.indexHandler)

	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
