package main

import (
	"context"
	"log/slog"
	"os"

	"demo/pkg/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	repo, err := db.NewRepo()
	if err != nil {
		logger.Error("Could not connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			logger.Error("Failed to close database connection", slog.String("error", err.Error()))
		}
	}()

	ctx := context.Background()
	if err := repo.Migrate(ctx); err != nil {
		logger.Error("Could not migrate database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Database migration completed successfully.")
}
