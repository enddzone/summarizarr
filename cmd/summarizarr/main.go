package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"summarizarr/internal/ai"
	"summarizarr/internal/api"
	"summarizarr/internal/database"
	"time"
	signalclient "summarizarr/internal/signal"
)

func main() {
	log.Println("Summarizarr starting...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := database.NewDB("summarizarr.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	aiModel := os.Getenv("OPENAI_MODEL")
	if aiModel == "" {
		aiModel = "gpt-4o"
	}

	aiClient := ai.NewClient(os.Getenv("OPENAI_API_KEY"), aiModel)
	apiServer := api.NewServer(":8081", db.DB)

	go apiServer.Start()

	// Replace with your actual address and number
	client := signalclient.NewClient("signal-cli-rest-api:8080", "+18177392137", db)

	go func() {
		if err := client.Listen(ctx); err != nil {
			log.Fatalf("Signal listener error: %v", err)
		}
	}()

	summarizationIntervalStr := os.Getenv("SUMMARIZATION_INTERVAL")
	if summarizationIntervalStr == "" {
		summarizationIntervalStr = "12h"
	}

	summarizationInterval, err := time.ParseDuration(summarizationIntervalStr)
	if err != nil {
		log.Fatalf("Invalid summarization interval: %v", err)
	}

	scheduler := ai.NewScheduler(db, aiClient, summarizationInterval)
	go scheduler.Start(ctx)

	<-ctx.Done()
	log.Println("Shutting down Summarizarr...")
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}
}