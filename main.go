package main

import (
	"github.com/tomiok/queuety/queuety/server"
	"log"
)

func main() {
	s, _ := server.NewServer("tcp4", ":9999", "")
	log.Fatal(s.Start())
}
