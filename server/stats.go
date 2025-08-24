package server

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// Este archivo está reservado para futuras implementaciones de estadísticas // Agregar tipos de stats.go
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

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	// Usar context y time para eliminar advertencias
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := statistics{
		Connections: connections{},
		Topics:      make(map[string]topicDetail),
	}

	conns := make(map[net.Conn]bool)

	for topic, cc := range s.clients {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled while processing stats")
			return
		default:
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
		log.Println("adding existing value")
		val.Add(1)
		return
	}

	log.Println("adding new value")
	var newVal = &atomic.Int32{}
	newVal.Add(1)
	s.sentMessages[topic] = newVal
}
