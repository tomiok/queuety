package main

import (
	"github.com/tomiok/queuety/queuety/server"
	"log"
)

func main() {
	s, err := server.NewServer("tcp4", ":9845", "")
	if err != nil {
		panic(err)
	}
	log.Fatal(s.Start())
}
