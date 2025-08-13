package main

import (
	"github.com/tomiok/queuety/server"
	"log"
)

func main() {
	s, err := server.NewServer(server.Config{
		Protocol:   "tcp",
		Port:       ":9845",
		BadgerPath: "/tmp/data",
		Duration:   10,
		Auth:       nil,
	})

	if err != nil {
		panic(err)
	}

	log.Fatal(s.Start())
}
