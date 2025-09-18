package main

import (
	"log"
	"os"
	"time"

	"github.com/tomiok/queuety/server"
)

const (
	portBrokerDefault = ":9845"
	portWebDefault    = ":9846"
)

func main() {
	badgerPath := os.Getenv("BADGER_PATH")
	if badgerPath == "" {
		badgerPath = "/tmp/badger1" //for local and NOT using docker, use tmp. Otherwise, go through Dockerfile env variable.
	}

	s, err := server.NewServer(server.Config{
		Protocol:      "tcp4",
		Port:          portBrokerDefault,
		WebServerPort: portWebDefault,
		BadgerPath:    badgerPath,
		Duration:      3600 * time.Second,
		Auth:          nil,

		// Rate limiting configuration
		RateLimitEnabled:     true,
		MaxMessagesPerSecond: 10,
		RateLimitQueueSize:   1000,
	})

	if err != nil {
		panic(err)
	}

	log.Printf("broker running on port %s \n Web server running on port %s \n",
		portBrokerDefault,
		portWebDefault,
	)

	log.Fatal(s.Start())
}
