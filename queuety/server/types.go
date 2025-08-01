package server

import (
	"encoding/json"
	"fmt"
)

type MType string

const (
	MessageTypeNewTopic      MType = "NEW_TOPIC"
	MessageTypeNew           MType = "NEW_MESSAGE"
	MessageTypeNewSubscriber MType = "NEW_SUB"
	MessageTypeACK           MType = "ACK"

	MessageFormatJSON MType = "JSON"

	MsgPrefixFalse = "false"
)

type Topic struct {
	Name string
}

func NewTopic(name string) Topic {
	return Topic{Name: name}
}

func (t Topic) IsEmpty() bool {
	return t.Name == ""
}

type Message struct {
	ID         string          `json:"id"`
	NextID     string          `json:"next_id"`
	Type       MType           `json:"type"`
	Topic      Topic           `json:"topic"`
	Body       json.RawMessage `json:"body"`
	BodyString string          `json:"body_string"`
	Timestamp  int64           `json:"timestamp"`
	ACK        bool            `json:"ack"`
}

func (m Message) Marshall() ([]byte, error) {
	return json.Marshal(m)
}

func (m Message) String() string {
	return fmt.Sprintf("Message %s, %s, %s at %d", m.Type, m.Topic, m.Body, m.Timestamp)
}
