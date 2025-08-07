package main

import (
	"github.com/tomiok/queuety/server"
	"log"
	"time"
)

func main() {
	s, err := server.NewServer("tcp4", ":9845", "", time.Second)
	if err != nil {
		panic(err)
	}
	log.Fatal(s.Start())
}
