package main

import (
	"github.com/tomiok/queuety/server"
	"log"
)

func main() {
	s, err := server.NewServer(server.Config{
		Protocol:   "tcp4",
		Port:       ":9845",
		Duration:   10,
		BadgerPath: "/tmp/data",
		Auth: &server.Auth{
			User:     "admin",
			Password: "admin",
		},
	})
	if err != nil {
		panic(err)
	}
	log.Fatal(s.Start())
}
