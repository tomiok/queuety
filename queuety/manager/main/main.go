package main

import (
	"fmt"
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

	go func() {
		for v := range conn.Consume(topic) {
			fmt.Println(v)
		}
	}()

	time.Sleep(1 * time.Second)
	var i int
	for {
		err = conn.PublishJSON(topic, fmt.Sprintf(`{"message": "hello %d"}`, i))
		if err != nil {
			panic(err)
		}

		i++
		time.Sleep(time.Second * 10)
	}
}
