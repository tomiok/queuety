package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

type statistics struct {
	Connections connections `json:"connections"`
	Topics      topics      `json:"topics"`
}

type topics map[string]topicDetail

type topicDetail struct {
	Subscribers  int   `json:"subscribers"`
	MessagesSent int32 `json:"messages_sent"`
}

type connections struct {
	Active         int `json:"active"`
	TotalConnected int `json:"total_connected"`
}

func (s *Server) incSentMessages(topic Topic) {
	s.sentMessages[topic].Add(1)
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := statistics{
		Connections: connections{},
		Topics:      make(map[string]topicDetail),
	}

	conns := make(map[net.Conn]bool)

	for topic, cc := range s.clients {
		for _, c := range cc {
			_, ok := conns[c]
			if !ok {
				conns[c] = true
				stats.Connections.TotalConnected++
			}
		}

		_, ok := stats.Topics[topic.Name]
		if !ok {
			sentMsgs := s.sentMessages[topic]

			stats.Topics[topic.Name] = topicDetail{
				Subscribers:  len(cc),
				MessagesSent: sentMsgs.Load(),
			}
		}
	}

	if err := json.NewEncoder(w).Encode(&stats); if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.webServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
