package server

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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

// MarshalBinary serializes Message to binary format
func (m *Message) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write ID length + ID
	idBytes := []byte(m.id)
	binary.Write(buf, binary.LittleEndian, uint16(len(idBytes)))
	buf.Write(idBytes)

	// Write NextID length + NextID
	nextIDBytes := []byte(m.nextID)
	binary.Write(buf, binary.LittleEndian, uint16(len(nextIDBytes)))
	buf.Write(nextIDBytes)

	// Write Type length + Type
	typeBytes := []byte(m.mType)
	binary.Write(buf, binary.LittleEndian, uint16(len(typeBytes)))
	buf.Write(typeBytes)

	// Write User length + User
	userBytes := []byte(m.user)
	binary.Write(buf, binary.LittleEndian, uint16(len(userBytes)))
	buf.Write(userBytes)

	// Write Password length + Password
	passwordBytes := []byte(m.password)
	binary.Write(buf, binary.LittleEndian, uint16(len(passwordBytes)))
	buf.Write(passwordBytes)

	// Write Topic name length + Topic name
	topicBytes := []byte(m.topic.Name)
	binary.Write(buf, binary.LittleEndian, uint16(len(topicBytes)))
	buf.Write(topicBytes)

	// Write Body length + Body
	bodyBytes := []byte(m.body)
	binary.Write(buf, binary.LittleEndian, uint32(len(bodyBytes)))
	buf.Write(bodyBytes)

	// Write BodyString length + BodyString
	bodyStringBytes := []byte(m.bodyString)
	binary.Write(buf, binary.LittleEndian, uint32(len(bodyStringBytes)))
	buf.Write(bodyStringBytes)

	// Write Timestamp (8 bytes)
	binary.Write(buf, binary.LittleEndian, m.timestamp)

	// Write ACK (1 byte)
	ackByte := byte(0)
	if m.ack {
		ackByte = 1
	}
	binary.Write(buf, binary.LittleEndian, ackByte)

	// Write Attempts (4 bytes)
	binary.Write(buf, binary.LittleEndian, int32(m.attempts))

	return buf.Bytes(), nil
}

// UnmarshalBinary deserializes binary data into Message
func (m *Message) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	// Read ID
	var idLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &idLen); err != nil {
		return err
	}
	idBytes := make([]byte, idLen)
	if _, err := io.ReadFull(buf, idBytes); err != nil {
		return err
	}
	m.id = string(idBytes)

	// Read NextID
	var nextIDLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &nextIDLen); err != nil {
		return err
	}
	nextIDBytes := make([]byte, nextIDLen)
	if _, err := io.ReadFull(buf, nextIDBytes); err != nil {
		return err
	}
	m.nextID = string(nextIDBytes)

	// Read Type
	var typeLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &typeLen); err != nil {
		return err
	}
	typeBytes := make([]byte, typeLen)
	if _, err := io.ReadFull(buf, typeBytes); err != nil {
		return err
	}
	m.mType = MType(typeBytes)

	// Read User
	var userLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &userLen); err != nil {
		return err
	}
	userBytes := make([]byte, userLen)
	if _, err := io.ReadFull(buf, userBytes); err != nil {
		return err
	}
	m.user = string(userBytes)

	// Read Password
	var passwordLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &passwordLen); err != nil {
		return err
	}
	passwordBytes := make([]byte, passwordLen)
	if _, err := io.ReadFull(buf, passwordBytes); err != nil {
		return err
	}
	m.password = string(passwordBytes)

	// Read Topic
	var topicLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &topicLen); err != nil {
		return err
	}
	topicBytes := make([]byte, topicLen)
	if _, err := io.ReadFull(buf, topicBytes); err != nil {
		return err
	}
	m.topic = Topic{Name: string(topicBytes)}

	// Read Body
	var bodyLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyLen); err != nil {
		return err
	}
	bodyBytes := make([]byte, bodyLen)
	if _, err := io.ReadFull(buf, bodyBytes); err != nil {
		return err
	}
	m.body = json.RawMessage(bodyBytes)

	// Read BodyString
	var bodyStringLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyStringLen); err != nil {
		return err
	}
	bodyStringBytes := make([]byte, bodyStringLen)
	if _, err := io.ReadFull(buf, bodyStringBytes); err != nil {
		return err
	}
	m.bodyString = string(bodyStringBytes)

	// Read Timestamp
	if err := binary.Read(buf, binary.LittleEndian, &m.timestamp); err != nil {
		return err
	}

	// Read ACK
	var ackByte byte
	if err := binary.Read(buf, binary.LittleEndian, &ackByte); err != nil {
		return err
	}
	m.ack = ackByte == 1

	// Read Attempts
	var attempts int32
	if err := binary.Read(buf, binary.LittleEndian, &attempts); err != nil {
		return err
	}
	m.attempts = int(attempts)

	return nil
}
