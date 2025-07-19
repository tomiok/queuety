package main

import (
	"github.com/tomiok/queuety/queuety/server"
	"log"
)

func main() {
	s := server.NewServer("tcp4", ":9845")
	log.Fatal(s.Start())
}
