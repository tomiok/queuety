package server

import (
	"log"
)

func (s *Server) run(query func() ([]Message, error)) {
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

			for _, msg := range messages {
				s.sendNewMessage(msg)
			}
		}
	}
}
