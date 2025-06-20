package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"demo/pkg/config"
	"demo/pkg/db"
	"demo/pkg/handlers"
	"demo/pkg/logging" // Import the new logging package
	"demo/pkg/metering"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
	"github.com/yandex-cloud/go-sdk/pkg/requestid"
	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	appCfg, err := config.LoadConfig(logger) // Use config.LoadConfig
	if err != nil {
		logger.Error("Failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	repo, err := db.NewRepo() // NewRepo already uses YDB_CONNECTION_STRING from env
	if err != nil {
		logger.Error("Could not connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// It's good practice to close the repo when main exits
	defer func() {
		if err := repo.Close(); err != nil {
			logger.Error("Failed to close database connection", slog.String("error", err.Error()))
		}
	}()

	var credentials ycsdk.Credentials
	if appCfg.ServiceAccountKeyFile != "" {
		fileData, err := os.ReadFile(appCfg.ServiceAccountKeyFile)
		if err != nil {
			logger.Error("Could not read service account key file", slog.String("error", err.Error()), slog.String("file", appCfg.ServiceAccountKeyFile))
			os.Exit(1)
		}
		var saKey iamkey.Key
		if err := saKey.UnmarshalJSON(fileData); err != nil {
			logger.Error("Could not unmarshal service account key", slog.String("error", err.Error()))
			os.Exit(1)
		}
		credentials, err = ycsdk.ServiceAccountKey(&saKey)
		if err != nil {
			logger.Error("Could not create service account credentials from key", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("Using service account key file credentials", slog.String("file", appCfg.ServiceAccountKeyFile))
	} else {
		credentials = ycsdk.InstanceServiceAccount()
		logger.Info("Using instance service account credentials (YC_SA_KEY_FILE not set)")
	}

	sdkConfig := ycsdk.Config{
		Credentials: credentials,
	}
	ctx := context.Background()
	sdk, err := ycsdk.Build(
		ctx,
		sdkConfig,
		grpc.WithUnaryInterceptor(requestid.Interceptor()), // Use ycsdk requestid interceptor
	)
	if err != nil {
		logger.Error("SDK init error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	meteringClient := metering.NewClient(sdk, logger, appCfg.DefaultSkuID)
	s := handlers.NewServer(repo, sdk, logger, meteringClient)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)              // This sets the request ID in the context for HTTP
	r.Use(logging.LoggingMiddleware(logger)) // Use LoggingMiddleware from the logging package
	r.Use(middleware.Recoverer)

	r.Get("/register", s.RegisterGetHandler)
	r.Post("/register", s.RegisterPostHandler)
	r.Get("/login", s.LoginGetHandler)
	r.Post("/login", s.LoginPostHandler)
	r.Get("/logout", s.LogoutGetHandler)
	r.Post("/bind", s.BindPostHandler)
	r.Post("/report", s.ReportPostHandler)
	r.Get("/", s.IndexGetHandler)

	logger.Info("Server starting", slog.String("port", appCfg.Port))
	if err := http.ListenAndServe(":"+appCfg.Port, r); err != nil {
		logger.Error("Could not start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
