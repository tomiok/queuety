package server

import (
	"log"
	"time"
)

type Scheduler struct {
	window *time.Ticker
}

// todo finish this
func (s Scheduler) run(query func() ([]Message, error)) {
	for {
		select {
		case <-s.window.C:
			messages, err := query()
			if err != nil {
				log.Printf("cannot fetch messages %v", err)
			}

			if len(messages) == 0 {
				continue
			}

			log.Printf("a dead message here %s", messages[0])
		}
	}
}
