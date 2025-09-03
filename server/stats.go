package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
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

func (s *Server) StartWebServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stats", s.handleStats)

	s.webServer.Handler = mux
	if err := s.webServer.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := statistics{
		Connections: connections{},
		Topics:      make(map[string]topicDetail),
	}

	conns := make(map[net.Conn]bool)

	for topic, clients := range s.clients {
		for _, c := range clients {
			_, ok := conns[c.conn]
			if !ok {
				conns[c.conn] = true
				stats.Connections.TotalConnected++
			}
		}

		_, ok := stats.Topics[topic.Name]
		if !ok {
			sentMsgs := s.sentMessages[topic]

			stats.Topics[topic.Name] = topicDetail{
				Subscribers:  len(clients),
				MessagesSent: sentMsgs.Load(),
			}
		}
	}

	if err := json.NewEncoder(w).Encode(&stats); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) incSentMessages(topic Topic) {
	val, ok := s.sentMessages[topic]
	if ok {
		val.Add(1)
		return
	}

	var newVal = &atomic.Int32{}
	newVal.Add(1)
	s.sentMessages[topic] = newVal
}

func (s *Server) ShutdownWebServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.webServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
