package main

import (
	"context"
	"log"

	"demo/pkg/db"
)

func main() {
	repo, err := db.NewRepo()
	if err != nil {
		log.Fatalf("Could not connect to database: %s\n", err)
	}
	ctx := context.Background()
	repo.Migrate(ctx)
}
