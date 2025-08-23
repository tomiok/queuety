package main

import (
	"encoding/json"
	"fmt"
	"github.com/tomiok/queuety/manager"
	"time"
)

type msg struct {
	Value int `json:"value"`
}

func main() {
	conn, err := manager.Connect("tcp4", ":9845", nil, nil)
	if err != nil {
		panic(err)
	}

	topic, err := conn.NewTopic("hello-1")
	if err != nil {
		panic(err)
	}

	go func() {
		for v := range manager.ConsumeJSON[msg](conn, topic) {
			fmt.Println(v)
		}
	}()

	time.Sleep(1 * time.Second)
	var i int
	for {
		_msg := msg{Value: i}
		b, _ := json.Marshal(_msg)
		err = conn.PublishJSON(topic, b)
		if err != nil {
			panic(err)
		}

		i++
		time.Sleep(time.Second * 2)
	}
}
