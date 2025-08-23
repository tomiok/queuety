package main

import (
	"log"
	"os"

	"github.com/tomiok/queuety/server"
	"github.com/tomiok/queuety/telemetry"
)

func main() {
	telemetryEnabled := os.Getenv("QUEUETY_TELEMETRY_ENABLED") == "true"
	telemetryBackend := os.Getenv("QUEUETY_TELEMETRY_BACKEND")
	if telemetryBackend == "" {
		telemetryBackend = "prometheus" // Default
	}

	// Initialize telemetry
	tel, err := telemetry.New(telemetry.Config{
		Enabled:  telemetryEnabled,
		Backend:  telemetryBackend,
		Endpoint: os.Getenv("QUEUETY_TELEMETRY_ENDPOINT"),
		APIKey:   os.Getenv("QUEUETY_TELEMETRY_API_KEY"),
	})

	if err != nil {
		log.Printf("Failed to initialize telemetry: %v", err)
		// Create disabled telemetry instead of failing
		tel = &telemetry.Telemetry{}
	}

	badgerPath := os.Getenv("BADGER_PATH")
	if badgerPath == "" {
		badgerPath = "/tmp/badger" //for local and NOT using docker, use tmp. Otherwise, go through Dockerfile env variable.
	}

	s, err := server.NewServer(server.Config{
		Protocol:   "tcp4",
		Port:       ":9845",
		BadgerPath: badgerPath,
		Duration:   10,
		Auth:       nil,
		Telemetry:  tel, // Add telemetry to server config
	})

	if err != nil {
		panic(err)
	}

	log.Printf("Starting Queuety server with telemetry enabled: %v, backend: %s", telemetryEnabled, telemetryBackend)
	log.Fatal(s.Start())
}
