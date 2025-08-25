package main

import (
	"github.com/tomiok/queuety/server"
	"log"
	"time"
)

func main() {
	s, err := server.NewServer(server.Config{
		Protocol:      "tcp",
		Port:          ":9845",
		BadgerPath:    "/tmp/data",
		WebServerPort: ":9846",
		Duration:      3600 * time.Second,
		Auth:          nil,
	})

	if err != nil {
		panic(err)
	}

	log.Fatal(s.Start())
}
