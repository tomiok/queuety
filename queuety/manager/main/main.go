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

	conn.Consume(topic)
	time.Sleep(1 * time.Second)
	for {
		err = conn.PublishJSON(topic, `{"message": "hello"}`)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second * 5)
	}
}
