package main

import (
	"github.com/tomiok/queuety/queuety/manager"
	"time"
)

func main() {
	conn, err := manager.Connect("tcp4", ":9845")
	if err != nil {
		panic(err)
	}

	topic, err := conn.NewTopic("hello-1")
	if err != nil {
		panic(err)
	}

	go conn.Consume(topic)

	for {
		err = conn.Publish(topic, `hola`)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second * 5)
	}
}
