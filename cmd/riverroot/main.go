// Package main is the entry point of the riverroot CI server: it opens the
// SQLite store, loads pipeline.yaml from the working directory, and serves
// the dashboard and REST API on :8080.
//
// Set DEMO=1 to pre-seed the database and disable the trigger endpoint,
// intended for public-facing deployments where we don't want anyone
// running real commands on the host.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/HeyItWorked/riverroot/internal/api"
	"github.com/HeyItWorked/riverroot/internal/pipeline"
	"github.com/HeyItWorked/riverroot/internal/store"
)

func main() {
	// open (or create) the database — same call as before, just not throwaway anymore
	s, err := store.NewSQLiteStore("data/riverroot.db")
	if err != nil {
		fmt.Println("error creating store:", err)
		os.Exit(1)
	}
	defer s.Close()

	// load pipeline config from working directory
	p, err := pipeline.Load("pipeline.yaml")
	if err != nil {
		fmt.Println("error loading pipeline:", err)
		os.Exit(1)
	}

	// if DEMO=1, seed some fake builds so the dashboard has stuff to show
	demo := os.Getenv("DEMO") == "1"
	if demo {
		if err := store.SeedDemo(s); err != nil {
			fmt.Println("error seeding demo data:", err)
			os.Exit(1)
		}
		fmt.Println("demo mode — seeded builds, trigger disabled")
	}

	// wire up the server — store for persistence, pipeline for triggering builds
	srv := api.NewServer(s, p, ".", demo)
	router := srv.SetupRouter()

	// explicit server instead of http.ListenAndServe so we can set timeouts —
	// otherwise a slow or idle client can hold a connection open forever.
	// WriteTimeout is deliberately huge: a build's logs are only written once
	// the whole run finishes, and pipeline.Run gives each step 5 minutes, so
	// anything tighter would truncate a long build's output mid-stream.
	httpSrv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,  // slowloris guard — headers are tiny
		ReadTimeout:       15 * time.Second, // requests have no body worth waiting on
		WriteTimeout:      30 * time.Minute, // generous — covers a multi-step build's logs
		IdleTimeout:       2 * time.Minute,  // how long an idle keep-alive sits before we reclaim it
	}

	// start serving — blocks until the process is killed
	fmt.Println("riverroot running on http://localhost:8080")
	err = httpSrv.ListenAndServe()
	fmt.Println("server failed:", err)
	os.Exit(1)
}
