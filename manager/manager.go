package manager

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/tomiok/queuety/server"
)

type MessageFormat byte

const (
	FormatJSON   MessageFormat = 0x01
	FormatBinary MessageFormat = 0x02
)

type QConn struct {
	c             net.Conn
	defaultFormat MessageFormat
}

type Auth struct {
	User string
	Pass string
}

func Connect(protocol, addr string, auth *Auth) (*QConn, error) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		return nil, err
	}

	qConn := QConn{
		c:             conn,
		defaultFormat: FormatJSON, // Default to JSON for backward compatibility
	}

	if auth != nil {
		msg := newMessage(server.MessageTypeAuth, server.Topic{}, nil, &MessageOptions{
			User:     auth.User,
			Password: auth.Pass,
		})
		msg.ID = generateNextID()

		err = qConn.writeMessageWithFormat(msg, FormatJSON)
		if err != nil {
			return nil, err
		}

		// listen to the message back.
		buff := make([]byte, 1024)
		n, errRead := conn.Read(buff)
		if errRead != nil {
			return nil, errRead
		}

		msgResponse, err := server.DecodeMessage(buff[:n])
		if err != nil {
			return nil, err
		}

		if msgResponse.MType == server.MessageAuthFailed {
			return nil, errors.New("authentication failed")
		}

		return &qConn, nil
	}

	return &qConn, nil
}

func (q *QConn) SetDefaultFormat(format MessageFormat) {
	q.defaultFormat = format
}

func (q *QConn) NewTopic(name string) (server.Topic, error) {
	topic := server.NewTopic(name)
	msg := newSimpleMessage(server.MessageTypeNewTopic, topic)

	err := q.qWrite(msg)
	if err != nil {
		return server.Topic{}, err
	}

	return topic, nil
}

func (q *QConn) PublishMessage(pubMsg server.PublishMessage) error {
	return q.PublishMessageWithTTL(pubMsg, 0)
}

func (q *QConn) PublishMessageWithTTL(pubMsg server.PublishMessage, ttl int64) error {
	msg := newPublishMessage(pubMsg.Topic, pubMsg.Body, ttl)
	return q.qWrite(msg)
}

func (q *QConn) Publish(t server.Topic, body string) error {
	return q.PublishWithTTL(t, body, 0)
}

func (q *QConn) PublishWithTTL(t server.Topic, body string, ttl int64) error {
	msg := newPublishMessage(t, json.RawMessage(body), ttl)
	return q.qWrite(msg)
}

func (q *QConn) PublishJSON(t server.Topic, body []byte) error {
	return q.PublishJSONWithTTL(t, body, 0)
}

func (q *QConn) PublishJSONWithTTL(t server.Topic, body []byte, ttl int64) error {
	msg := newPublishMessage(t, body, ttl)
	return q.qWrite(msg)
}

func (q *QConn) PublishBinary(t server.Topic, body []byte) error {
	return q.PublishBinaryWithTTL(t, body, 0)
}

func (q *QConn) PublishBinaryWithTTL(t server.Topic, body []byte, ttl int64) error {
	msg := newPublishMessage(t, body, ttl)
	return q.writeMessageWithFormat(msg, FormatBinary)
}

// ConsumeJSON will be used for type-safety. Is a generic function.
// Both publish types has the ergonomics to send body as JSON and the string representation.
// In this case, is just easier to reuse or replicate the JSON structure.
func ConsumeJSON[T any](q *QConn, topic server.Topic) <-chan T {
	return consumeJSONWithFraming[T](q, topic)
}

func consumeJSONWithFraming[T any](q *QConn, topic server.Topic) <-chan T {
	if err := q.subscribe(topic); err != nil {
		log.Printf("cannot sub %v\n", err)
		return nil
	}

	ch := make(chan T, 1000)

	go func() {
		defer close(ch)

		for {
			// read format flag (1 byte) -> maybe we need this in the future.
			//formatBuff := make([]byte, 1)
			//_, err := io.ReadFull(q.c, formatBuff)
			//if err != nil {
			//	if err == io.EOF {
			//		log.Println("connection closed")
			//		return
			//	}
			//	log.Printf("cannot read format flag: %v\n", err)
			//	continue
			//}

			// read length (4 bytes little endian)
			lengthBuff := make([]byte, 5)
			_, err := io.ReadFull(q.c, lengthBuff)
			if err != nil {
				log.Printf("cannot read message length: %v\n", err)
				continue
			}

			// skip format flag (1 byte), read until 5th byte.
			messageLength := binary.LittleEndian.Uint32(lengthBuff[1:5])

			// safety check
			if messageLength > 10*1024*1024 { // 10MB max
				log.Printf("message too large: %d bytes, discarding\n", messageLength)
				_, _ = io.CopyN(io.Discard, q.c, int64(messageLength))
				continue
			}

			// read payload
			payload := make([]byte, messageLength)
			_, err = io.ReadFull(q.c, payload)
			if err != nil {
				log.Printf("cannot read payload: %v\n", err)
				continue
			}

			// 5. Decodificar según formato
			var msg server.Message

			msg, err = server.DecodeMessage(payload)
			if err != nil {
				log.Printf("cannot decode JSON message: %v\n", err)
				continue
			}

			// 6. Unmarshal body
			var t T
			if err = json.Unmarshal(msg.Body, &t); err != nil {
				log.Printf("unable to unmarshal body: %v\n", err)
				continue
			}

			ch <- t
			q.updateMessage(msg)
		}
	}()

	return ch
}

// Consume will be used for receive the channel with string type. just raw string.
// Both publish types has the ergonomics to send body as JSON and the string representation.
// Consumer must be aware of which type is the publisher sending but is split in diff methods for simplicity and
// will be compatible in the future if any change is included.
// Replace the existing Consume function in manager.go with this:
func Consume(q *QConn, topic server.Topic) <-chan string {
	if err := q.subscribe(topic); err != nil {
		log.Printf("cannot sub %v\n", err)
		return nil
	}

	ch := make(chan string, 1000)
	go func() {
		defer close(ch)
		for {
			// Read format flag (1 byte)
			formatBuff := make([]byte, 1)
			_, err := io.ReadFull(q.c, formatBuff)
			if err != nil {
				log.Printf("cannot read format flag %v \n", err)
				continue
			}
			format := MessageFormat(formatBuff[0])
			fmt.Printf("DEBUG: Format flag = %d\n", format)

			// Read length (4 bytes)
			lengthBuff := make([]byte, 4)
			_, err = io.ReadFull(q.c, lengthBuff)
			if err != nil {
				log.Printf("cannot read message length %v \n", err)
				continue
			}
			messageLength := binary.LittleEndian.Uint32(lengthBuff)

			// SAFETY CHECK - prevent huge allocations
			if messageLength > 10*1024*1024 { // 10MB max
				log.Printf("[ERROR] message length too large: %d bytes, skipping\n", messageLength)
				continue
			}

			// Always process as binary (no format check)

			// Read payload
			payload := make([]byte, messageLength)
			_, err = io.ReadFull(q.c, payload)
			if err != nil {
				log.Printf("cannot read message payload %v \n", err)
				continue
			}

			// Unmarshal binary message
			msg := server.Message{}
			err = server.UnmarshalBinary(payload, &msg)
			if err != nil {
				log.Printf("cannot unmarshal binary message %v \n", err)
				continue
			}

			ch <- msg.BodyString
			q.updateMessage(msg)
		}
	}()

	return ch
}

func (q *QConn) subscribe(t server.Topic) error {
	msg := newSimpleMessage(server.MessageTypeNewSubscriber, t)
	msg.NextID = msg.ID // For subscription, ID and NextID are the same

	return q.qWrite(msg)
}

func (q *QConn) unsubscribe() error {
	return nil
}

func (q *QConn) updateMessage(msg server.Message) {
	// Create ACK message by copying received message and updating type/ack
	ackMsg := msg
	ackMsg.MType = server.MessageTypeACK
	ackMsg.ACK = true

	if err := q.writeMessage(ackMsg); err != nil {
		log.Printf("cannot send ACK confirmation, message id %s \n", msg.ID)
	}
}

func (q *QConn) qWrite(m server.Message) error {
	return q.writeMessage(m)
}

func (q *QConn) writeMessage(m server.Message) error {
	return q.writeMessageWithFormat(m, q.defaultFormat)
}

func (q *QConn) writeMessageWithFormat(m server.Message, format MessageFormat) error {
	var payload []byte
	var err error

	switch format {
	case FormatJSON:
		payload, err = m.Marshall()
	case FormatBinary:
		payload, err = m.MarshalBinary()
	default:
		return fmt.Errorf("unsupported format: %d", format)
	}

	if err != nil {
		return err
	}

	// Write format flag (1 byte)
	if err = binary.Write(q.c, binary.LittleEndian, format); err != nil {
		return err
	}

	// Write length (4 bytes)
	length := uint32(len(payload))
	if err = binary.Write(q.c, binary.LittleEndian, length); err != nil {
		return err
	}

	// Write payload
	_, err = q.c.Write(payload)
	return err
}

// MessageOptions holds optional parameters for message creation
type MessageOptions struct {
	TTL      int64  // Time to live in seconds (0 = no expiration)
	User     string // For authentication messages
	Password string // For authentication messages
}

// newMessage creates a new message with common fields set
func newMessage(mType server.MType, topic server.Topic, body json.RawMessage, opts *MessageOptions) server.Message {
	msg := server.Message{
		MType:     mType,
		Topic:     topic,
		Body:      body,
		Timestamp: time.Now().Unix(),
		ACK:       false,
		Attempts:  0,
	}

	// Set body string if body is provided
	if body != nil {
		msg.BodyString = string(body)
	}

	// Apply options if provided
	if opts != nil {
		msg.TTL = opts.TTL
		msg.User = opts.User
		msg.Password = opts.Password
	}

	return msg
}

// newPublishMessage creates a message for publishing (with ID generation)
func newPublishMessage(topic server.Topic, body json.RawMessage, ttl int64) server.Message {
	nextID := generateNextID()
	msg := newMessage(server.MessageTypeNew, topic, body, &MessageOptions{TTL: ttl})
	msg.ID = generateID(server.MsgPrefixFalse, nextID)
	msg.NextID = nextID
	return msg
}

// newSimpleMessage creates a message with just type and topic (no body)
func newSimpleMessage(mType server.MType, topic server.Topic) server.Message {
	msg := newMessage(mType, topic, nil, nil)
	msg.ID = generateNextID()
	return msg
}

func generateNextID() string {
	return uuid.NewString()
}

func generateID(prefix string, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}
