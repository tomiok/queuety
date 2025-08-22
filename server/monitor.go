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

type monitor struct {
	addr   string
	server http.Server
	mqSrv  *Server

	sentMessages map[string]*atomic.Int32
}

func newMonitor(addr string, queueSrv *Server) *monitor {
	return &monitor{
		addr:  addr,
		mqSrv: queueSrv,

		sentMessages: make(map[string]*atomic.Int32),
	}
}

func (m *monitor) start() error {
	handler := http.NewServeMux()
	handler.HandleFunc("GET /stats", m.handleStats)

	m.server = http.Server{
		Handler: handler,
		Addr:    m.addr,
	}

	err := m.server.ListenAndServe()
	if err != nil {
		return err
	}

	return nil
}

func (m *monitor) incSentMessages(topic Topic) {
	m.sentMessages[topic.Name].Add(1)
}

func (m *monitor) handleStats(w http.ResponseWriter, _ *http.Request) {
	stats := statistics{
		Connections: connections{},
		Topics:      make(map[string]topicDetail),
	}

	connections := make(map[net.Conn]bool)

	for topic, cc := range m.mqSrv.clients {
		for _, c := range cc {
			_, ok := connections[c]
			if !ok {
				connections[c] = true
				stats.Connections.TotalConnected++
			}
		}

		_, ok := stats.Topics[topic.Name]
		if !ok {
			sentMsgs := m.sentMessages[topic.Name]

			stats.Topics[topic.Name] = topicDetail{
				Subscribers:  len(cc),
				MessagesSent: sentMsgs.Load(),
			}
		}
	}

	err := json.NewEncoder(w).Encode(&stats)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (m *monitor) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
