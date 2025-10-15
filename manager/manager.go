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
		msg := server.NewMessageBuilder().
			WithID(generateNextID()).
			WithType(server.MessageTypeAuth).
			WithUser(auth.User).
			WithPassword(auth.Pass).
			WithTimestamp(time.Now().Unix()).
			Build()

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

		if msgResponse.Type() == server.MessageAuthFailed {
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
	m := server.NewMessageBuilder().
		WithID(uuid.NewString()).
		WithType(server.MessageTypeNewTopic).
		WithTopic(server.NewTopic(name)).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	err := q.qWrite(m)
	if err != nil {
		return server.Topic{}, err
	}

	return server.Topic{
		Name: name,
	}, nil
}

func (q *QConn) PublishMessage(pubMsg server.PublishMessage) error {
	nextID := generateNextID()

	m := server.NewMessageBuilder().
		WithID(generateID(server.MsgPrefixFalse, nextID)).
		WithNextID(nextID).
		WithType(server.MessageTypeNew).
		WithTopic(pubMsg.Topic).
		WithBody(pubMsg.Body).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	return q.qWrite(m)
}

func (q *QConn) Publish(t server.Topic, msg string) error {
	nextID := generateNextID()

	m := server.NewMessageBuilder().
		WithID(generateID(server.MsgPrefixFalse, nextID)).
		WithNextID(nextID).
		WithType(server.MessageTypeNew).
		WithTopic(t).
		WithBody(json.RawMessage(msg)).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	return q.qWrite(m)
}

func (q *QConn) PublishJSON(t server.Topic, msg []byte) error {
	nextID := generateNextID()

	m := server.NewMessageBuilder().
		WithID(generateID(server.MsgPrefixFalse, nextID)).
		WithNextID(nextID).
		WithType(server.MessageTypeNew).
		WithTopic(t).
		WithBody(msg).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	return q.qWrite(m)
}

func (q *QConn) PublishBinary(t server.Topic, msg []byte) error {
	nextID := generateNextID()

	m := server.NewMessageBuilder().
		WithID(generateID(server.MsgPrefixFalse, nextID)).
		WithNextID(nextID).
		WithType(server.MessageTypeNew).
		WithTopic(t).
		WithBody(msg).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	return q.writeMessageWithFormat(m, FormatBinary)
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
			if err = json.Unmarshal(msg.Body(), &t); err != nil {
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
			fmt.Printf("DEBUG: Message length = %d\n", messageLength)

			// SAFETY CHECK - prevent huge allocations
			if messageLength > 10*1024*1024 { // 10MB max
				log.Printf("Message length too large: %d bytes, skipping\n", messageLength)
				continue
			}

			// Always process as binary (no format check)

			// Read payload
			payload := make([]byte, messageLength)
			fmt.Printf("DEBUG: About to read payload of %d bytes\n", messageLength)
			_, err = io.ReadFull(q.c, payload)
			if err != nil {
				log.Printf("cannot read message payload %v \n", err)
				continue
			}
			fmt.Printf("DEBUG: Successfully read payload\n")

			// Unmarshal binary message
			msg := server.Message{}
			err = msg.UnmarshalBinary(payload)
			if err != nil {
				log.Printf("cannot unmarshal binary message %v \n", err)
				continue
			}

			fmt.Printf("DEBUG: Unmarshaled message: %+v\n", msg)
			ch <- msg.BodyString()
			q.updateMessage(msg)
		}
	}()

	return ch
}

func (q *QConn) subscribe(t server.Topic) error {
	id := generateNextID()
	m := server.NewMessageBuilder().
		WithID(id).
		WithNextID(id).
		WithType(server.MessageTypeNewSubscriber).
		WithTopic(t).
		WithTimestamp(time.Now().UnixMilli()).
		WithAck(false).
		Build()

	return q.qWrite(m)
}

func (q *QConn) unsubscribe() error {
	return nil
}

func (q *QConn) updateMessage(msg server.Message) {
	m := server.NewMessageBuilder().
		WithID(msg.ID()).
		WithNextID(msg.NextID()).
		WithUser(msg.User()).
		WithPassword(msg.Password()).
		WithTopic(msg.Topic()).
		WithBody(msg.Body()).
		WithTimestamp(msg.Timestamp()).
		WithAttempts(msg.Attempts()).
		WithType(server.MessageTypeACK).
		WithAck(true).
		Build()

	if err := q.writeMessage(m); err != nil {
		log.Printf("cannot send ACK confirmation, message id %s \n", msg.ID())
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

func generateNextID() string {
	return uuid.NewString()
}

func generateID(prefix string, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}
