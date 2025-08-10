package main

import (
	"github.com/tomiok/queuety/server"
	"log"
	"os"
)

func main() {
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
	})

	if err != nil {
		panic(err)
	}

	log.Fatal(s.Start())
}
