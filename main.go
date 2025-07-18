package main

import (
	"github.com/tomiok/queuety/queuety"
	"log"
)

func main() {
	s := queuety.NewServer("tcp4", ":9999")
	log.Fatal(s.Start())
}
