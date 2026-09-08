// Command api runs the deployment tracker HTTP service.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dovlahaha/deploytrack/internal/api"
	"github.com/dovlahaha/deploytrack/internal/store"
)

func main() {
	// Read configuration from the environment so the same image runs
	// unchanged against a local database, a CI sidecar, or production.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s, err := store.Open(dsn)
	if err != nil {
		log.Fatalf("could not open the database: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.New(s).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server stopped: %v", err)
	}
}
