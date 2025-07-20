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

	var i int
	for {
		i = i + 10
		err = conn.NewTopic(fmt.Sprintf("hello %d", i))
		if err != nil {
			panic(err)
		}

		i++
		time.Sleep(5 * time.Second)
	}
}
