package main

import (
	"github.com/tomiok/queuety/server"
	"log"
)

func main() {
	s, err := server.NewServer(server.Config{
		Protocol:   "tcp4",
		Port:       ":9845",
		BadgerPath: "",
		Duration:   10,
		Auth:       nil,
	})

	if err != nil {
		panic(err)
	}

	log.Fatal(s.Start())
}
