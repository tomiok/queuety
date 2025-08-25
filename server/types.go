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

	MessageFormatJSON = "JSON"
	MsgPrefixFalse    = "false"
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

type PublishMessage struct {
	Topic Topic           `json:"topic"`
	Body  json.RawMessage `json:"body"`
}

type Message struct {
	id         string
	nextID     string
	mType      MType
	user       string
	password   string
	topic      Topic
	body       json.RawMessage
	bodyString string
	timestamp  int64
	ack        bool
	attempts   int
}

type messageJSON struct {
	ID         string          `json:"id"`
	NextID     string          `json:"next_id"`
	Type       MType           `json:"type"`
	User       string          `json:"user"`
	Password   string          `json:"password"`
	Topic      Topic           `json:"topic"`
	Body       json.RawMessage `json:"body"`
	BodyString string          `json:"body_string"`
	Timestamp  int64           `json:"timestamp"`
	ACK        bool            `json:"ack"`
	Attempts   int             `json:"attempts"`
}

func (m *Message) ID() string {
	return m.id
}

func (m *Message) NextID() string {
	return m.nextID
}

func (m *Message) Type() MType {
	return m.mType
}

func (m *Message) User() string {
	return m.user
}

func (m *Message) Password() string {
	return m.password
}

func (m *Message) Topic() Topic {
	return m.topic
}

func (m *Message) Body() json.RawMessage {
	return m.body
}

func (m *Message) BodyString() string {
	return m.bodyString
}

func (m *Message) Timestamp() int64 {
	return m.timestamp
}

func (m *Message) ACK() bool {
	return m.ack
}

func (m *Message) Attempts() int {
	return m.attempts
}

func (m *Message) IncAttempts() {
	m.attempts += 1
}

func (m *Message) updateACK() {
	m.id = m.nextID
	m.ack = true
}

func (m *Message) updateAuthSuccess() {
	m.mType = MessageAuthSuccess
}

func (m *Message) updateAuthFailed() {
	m.mType = MessageAuthFailed
}

func NewMessage(pubMsg PublishMessage) Message {
	return NewMessageBuilder().
		WithTopic(pubMsg.Topic).
		WithBody(pubMsg.Body).
		Build()
}

// MessageBuilder builder pattern
type MessageBuilder struct {
	msg Message
}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		msg: Message{
			timestamp: time.Now().Unix(),
			attempts:  0,
			ack:       false,
		},
	}
}

func (m *Message) Marshall() ([]byte, error) {
	mJSON := messageJSON{
		ID:         m.id,
		NextID:     m.nextID,
		Type:       m.mType,
		User:       m.user,
		Password:   m.password,
		Topic:      m.topic,
		Body:       m.body,
		BodyString: m.bodyString,
		Timestamp:  m.timestamp,
		ACK:        m.ack,
		Attempts:   m.attempts,
	}

	return json.Marshal(mJSON)
}

func (m *Message) Unmarshal(data []byte) error {
	var mJSON messageJSON
	if err := json.Unmarshal(data, &mJSON); err != nil {
		return err
	}

	m.id = mJSON.ID
	m.nextID = mJSON.NextID
	m.mType = mJSON.Type
	m.user = mJSON.User
	m.password = mJSON.Password
	m.topic = mJSON.Topic
	m.body = mJSON.Body
	m.bodyString = mJSON.BodyString
	m.timestamp = mJSON.Timestamp
	m.ack = mJSON.ACK
	m.attempts = mJSON.Attempts
	return nil
}

func DecodeMessage(b []byte) (Message, error) {
	r := bytes.NewReader(b)
	var mJSON messageJSON
	if err := json.NewDecoder(r).Decode(&mJSON); err != nil {
		return Message{}, err
	}
	return Message{
		id:         mJSON.ID,
		nextID:     mJSON.NextID,
		mType:      mJSON.Type,
		user:       mJSON.User,
		password:   mJSON.Password,
		topic:      mJSON.Topic,
		body:       mJSON.Body,
		bodyString: mJSON.BodyString,
		timestamp:  mJSON.Timestamp,
		ack:        mJSON.ACK,
		attempts:   mJSON.Attempts,
	}, nil
}

func (m *Message) String() string {
	return fmt.Sprintf("Message %s, %s, %s at %d", m.mType, m.topic, m.body, m.timestamp)
}

func (mb *MessageBuilder) WithTopic(topic Topic) *MessageBuilder {
	mb.msg.topic = topic
	return mb
}

func (mb *MessageBuilder) WithBody(body json.RawMessage) *MessageBuilder {
	mb.msg.body = body
	mb.msg.bodyString = string(body)
	return mb
}

func (mb *MessageBuilder) WithID(ID string) *MessageBuilder {
	mb.msg.id = ID
	return mb
}

func (mb *MessageBuilder) WithNextID(nextID string) *MessageBuilder {
	mb.msg.nextID = nextID
	return mb
}

func (mb *MessageBuilder) WithType(mtype MType) *MessageBuilder {
	mb.msg.mType = mtype
	return mb
}

func (mb *MessageBuilder) WithUser(user string) *MessageBuilder {
	mb.msg.user = user
	return mb
}

func (mb *MessageBuilder) WithPassword(password string) *MessageBuilder {
	mb.msg.password = password
	return mb
}

func (mb *MessageBuilder) WithAck(ack bool) *MessageBuilder {
	mb.msg.ack = ack
	return mb
}

func (mb *MessageBuilder) WithAttempts(attempts int) *MessageBuilder {
	mb.msg.attempts = attempts
	return mb
}

func (mb *MessageBuilder) WithTimestamp(ts int64) *MessageBuilder {
	mb.msg.timestamp = ts
	return mb
}

func (mb *MessageBuilder) Build() Message {
	return mb.msg
}
