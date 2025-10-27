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
	Topic Topic           `json:"Topic"`
	Body  json.RawMessage `json:"Body"`
}

type Message struct {
	ID         string
	NextID     string
	MType      MType
	User       string
	Password   string
	Topic      Topic
	Body       json.RawMessage
	BodyString string
	Timestamp  int64
	ACK        bool
	Attempts   int
}

func (m *Message) IncAttempts() {
	m.Attempts++
}

func (m *Message) updateACK() {
	m.ID = m.NextID
	m.ACK = true
}

func (m *Message) updateAuthSuccess() {
	m.MType = MessageAuthSuccess
}

func (m *Message) updateAuthFailed() {
	m.MType = MessageAuthFailed
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
			Timestamp: time.Now().Unix(),
			Attempts:  0,
			ACK:       false,
		},
	}
}

func (m *Message) Marshall() ([]byte, error) {
	mJSON := Message{
		ID:         m.ID,
		NextID:     m.NextID,
		MType:      m.MType,
		User:       m.User,
		Password:   m.Password,
		Topic:      m.Topic,
		Body:       m.Body,
		BodyString: m.BodyString,
		Timestamp:  m.Timestamp,
		ACK:        m.ACK,
		Attempts:   m.Attempts,
	}

	return json.Marshal(mJSON)
}

func (m *Message) Unmarshal(data []byte) error {
	var mJSON Message
	if err := json.Unmarshal(data, &mJSON); err != nil {
		return err
	}

	m.ID = mJSON.ID
	m.NextID = mJSON.NextID
	m.MType = mJSON.MType
	m.User = mJSON.User
	m.Password = mJSON.Password
	m.Topic = mJSON.Topic
	m.Body = mJSON.Body
	m.BodyString = mJSON.BodyString
	m.Timestamp = mJSON.Timestamp
	m.ACK = mJSON.ACK
	m.Attempts = mJSON.Attempts
	return nil
}

func DecodeMessage(b []byte) (Message, error) {
	r := bytes.NewReader(b)
	var mJSON Message
	if err := json.NewDecoder(r).Decode(&mJSON); err != nil {
		return Message{}, err
	}

	return Message{
		ID:         mJSON.ID,
		NextID:     mJSON.NextID,
		MType:      mJSON.MType,
		User:       mJSON.User,
		Password:   mJSON.Password,
		Topic:      mJSON.Topic,
		Body:       mJSON.Body,
		BodyString: mJSON.BodyString,
		Timestamp:  mJSON.Timestamp,
		ACK:        mJSON.ACK,
		Attempts:   mJSON.Attempts,
	}, nil
}

func (m *Message) String() string {
	return fmt.Sprintf("Message %s, %s, %s at %d", m.MType, m.Topic, m.Body, m.Timestamp)
}

func (mb *MessageBuilder) WithTopic(topic Topic) *MessageBuilder {
	mb.msg.Topic = topic
	return mb
}

func (mb *MessageBuilder) WithBody(body json.RawMessage) *MessageBuilder {
	mb.msg.Body = body
	mb.msg.BodyString = string(body)
	return mb
}

func (mb *MessageBuilder) WithID(ID string) *MessageBuilder {
	mb.msg.ID = ID
	return mb
}

func (mb *MessageBuilder) WithNextID(nextID string) *MessageBuilder {
	mb.msg.NextID = nextID
	return mb
}

func (mb *MessageBuilder) WithType(mtype MType) *MessageBuilder {
	mb.msg.MType = mtype
	return mb
}

func (mb *MessageBuilder) WithUser(user string) *MessageBuilder {
	mb.msg.User = user
	return mb
}

func (mb *MessageBuilder) WithPassword(password string) *MessageBuilder {
	mb.msg.Password = password
	return mb
}

func (mb *MessageBuilder) WithAck(ack bool) *MessageBuilder {
	mb.msg.ACK = ack
	return mb
}

func (mb *MessageBuilder) WithAttempts(attempts int) *MessageBuilder {
	mb.msg.Attempts = attempts
	return mb
}

func (mb *MessageBuilder) WithTimestamp(ts int64) *MessageBuilder {
	mb.msg.Timestamp = ts
	return mb
}

func (mb *MessageBuilder) Build() Message {
	return mb.msg
}

// MarshalBinary serializes Message to binary format
func (m *Message) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write ID length + ID
	idBytes := []byte(m.ID)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(idBytes))); err != nil {
		return nil, err
	}

	buf.Write(idBytes)

	// Write NextID length + NextID
	nextIDBytes := []byte(m.NextID)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(nextIDBytes))); err != nil {
		return nil, err
	}

	buf.Write(nextIDBytes)

	// Write Type length + Type
	typeBytes := []byte(m.MType)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(typeBytes))); err != nil {
		return nil, err
	}

	buf.Write(typeBytes)

	// Write User length + User
	userBytes := []byte(m.User)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(userBytes))); err != nil {
		return nil, err
	}

	buf.Write(userBytes)

	// Write Password length + Password
	passwordBytes := []byte(m.Password)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(passwordBytes))); err != nil {
		return nil, err
	}

	buf.Write(passwordBytes)

	// Write Topic name length + Topic name
	topicBytes := []byte(m.Topic.Name)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(topicBytes))); err != nil {
		return nil, err
	}

	buf.Write(topicBytes)

	// Write Body length + Body
	bodyBytes := []byte(m.Body)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(bodyBytes))); err != nil {
		return nil, err
	}

	buf.Write(bodyBytes)

	// Write BodyString length + BodyString
	bodyStringBytes := []byte(m.BodyString)
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(bodyStringBytes))); err != nil {
		return nil, err
	}

	buf.Write(bodyStringBytes)

	// Write Timestamp (8 bytes)
	if err := binary.Write(buf, binary.LittleEndian, m.Timestamp); err != nil {
		return nil, err
	}

	// Write ACK (1 byte)
	ackByte := byte(0)
	if m.ACK {
		ackByte = 1
	}

	if err := binary.Write(buf, binary.LittleEndian, ackByte); err != nil {
		return nil, err
	}

	// Write Attempts (4 bytes)
	if err := binary.Write(buf, binary.LittleEndian, int32(m.Attempts)); err != nil {
		return nil, err
	}

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
	m.ID = string(idBytes)

	// Read NextID
	var nextIDLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &nextIDLen); err != nil {
		return err
	}
	nextIDBytes := make([]byte, nextIDLen)
	if _, err := io.ReadFull(buf, nextIDBytes); err != nil {
		return err
	}
	m.NextID = string(nextIDBytes)

	// Read Format
	var typeLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &typeLen); err != nil {
		return err
	}
	typeBytes := make([]byte, typeLen)
	if _, err := io.ReadFull(buf, typeBytes); err != nil {
		return err
	}
	m.MType = MType(typeBytes)

	// Read User
	var userLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &userLen); err != nil {
		return err
	}
	userBytes := make([]byte, userLen)
	if _, err := io.ReadFull(buf, userBytes); err != nil {
		return err
	}
	m.User = string(userBytes)

	// Read Password
	var passwordLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &passwordLen); err != nil {
		return err
	}
	passwordBytes := make([]byte, passwordLen)
	if _, err := io.ReadFull(buf, passwordBytes); err != nil {
		return err
	}
	m.Password = string(passwordBytes)

	// Read Topic
	var topicLen uint16
	if err := binary.Read(buf, binary.LittleEndian, &topicLen); err != nil {
		return err
	}
	topicBytes := make([]byte, topicLen)
	if _, err := io.ReadFull(buf, topicBytes); err != nil {
		return err
	}
	m.Topic = Topic{Name: string(topicBytes)}

	// Read Body
	var bodyLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyLen); err != nil {
		return err
	}
	bodyBytes := make([]byte, bodyLen)
	if _, err := io.ReadFull(buf, bodyBytes); err != nil {
		return err
	}
	m.Body = json.RawMessage(bodyBytes)

	// Read BodyString
	var bodyStringLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &bodyStringLen); err != nil {
		return err
	}
	bodyStringBytes := make([]byte, bodyStringLen)
	if _, err := io.ReadFull(buf, bodyStringBytes); err != nil {
		return err
	}
	m.BodyString = string(bodyStringBytes)

	// Read Timestamp
	if err := binary.Read(buf, binary.LittleEndian, &m.Timestamp); err != nil {
		return err
	}

	// Read ACK
	var ackByte byte
	if err := binary.Read(buf, binary.LittleEndian, &ackByte); err != nil {
		return err
	}
	m.ACK = ackByte == 1

	// Read Attempts
	var attempts int32
	if err := binary.Read(buf, binary.LittleEndian, &attempts); err != nil {
		return err
	}
	m.Attempts = int(attempts)

	return nil
}
