package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

type MType string

const (
	MessageTypeNewTopic      MType = "NEW_TOPIC"
	MessageTypeNew           MType = "NEW_MESSAGE"
	MessageTypeNewSubscriber MType = "NEW_SUB"
	MessageTypeACK           MType = "ACK"
	MessageTypeAuth          MType = "AUTH"
	MessageAuthSuccess       MType = "AUTH_SUCCESS"
	MessageAuthFailed        MType = "AUTH_FAILED"

	MsgPrefixFalse = "false"
)

type PublishMessage struct {
	Topic Topic           `json:"Topic"`
	Body  json.RawMessage `json:"Body"`
}

type Message struct {
	ID         string          `json:"id"`
	NextID     string          `json:"next_id"`
	MType      MType           `json:"type"`
	User       string          `json:"user"`
	Password   string          `json:"password"`
	Topic      Topic           `json:"topic"`
	Body       json.RawMessage `json:"body"`
	BodyString string          `json:"body_string"`
	Timestamp  int64           `json:"timestamp"`
	ACK        bool            `json:"ack"`
	Attempts   int             `json:"attempts"`
	TTL        int64           `json:"ttl"` //in seconds
}

func (m Message) IncAttempts() Message {
	m.Attempts++
	return m
}

func (m Message) updateACK() Message {
	m.ID = m.NextID
	m.ACK = true

	return m
}

func (m Message) Marshall() ([]byte, error) {
	return json.Marshal(m)
}

func (m Message) String() string {
	return fmt.Sprintf("Message %s, %s, %s at %d", m.MType, m.Topic, m.Body, m.Timestamp)
}

func (m Message) IsExpired() bool {
	if m.TTL == 0 {
		return false
	}

	return time.Now().Unix() > m.Timestamp+m.TTL
}

func DecodeMessage(b []byte) (Message, error) {
	r := bytes.NewReader(b)
	var msg Message
	if err := json.NewDecoder(r).Decode(&msg); err != nil {
		return Message{}, err
	}

	return msg, nil
}
