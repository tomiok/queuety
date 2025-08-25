package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/tomiok/queuety/server"
)

type QConn struct {
	c net.Conn
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
		conn,
	}

	if auth != nil {
		msg := server.NewMessageBuilder().
			WithID(generateNextID()).
			WithType(server.MessageTypeAuth).
			WithUser(auth.User).
			WithPassword(auth.Pass).
			WithTimestamp(time.Now().Unix()).
			Build()

		b, _err := msg.Marshall()
		if _err != nil {
			return nil, _err
		}

		_, _ = conn.Write(b)

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

// ConsumeJSON will be used for type-safety. Is a generic function.
// Both publish types has the ergonomics to send body as JSON and the string representation.
// In this case, is just easier to reuse or replicate the JSON structure.
func ConsumeJSON[T any](q *QConn, topic server.Topic) <-chan T {
	if err := q.subscribe(topic); err != nil {
		log.Printf("cannot sub %v\n", err)
		return nil
	}

	ch := make(chan T, 1000)
	go func() {
		defer close(ch)
		for {
			b := make([]byte, 1024)
			n, err := q.c.Read(b)
			if err != nil {
				log.Printf("cannot read messege %v \n", err)
				continue
			}

			if n > 0 {
				msg, err := server.DecodeMessage(b[:n])
				if err != nil {
					log.Printf("cannot get messege %v \n", err)
					continue
				}

				var t T
				if err = json.Unmarshal(msg.Body(), &t); err != nil {
					log.Printf("unable to unmarshal %v\n", err)
				}

				ch <- t
				q.updateMessage(msg)
			}
		}
	}()

	return ch
}

// Consume will be used for receive the channel with string type. just raw string.
// Both publish types has the ergonomics to send body as JSON and the string representation.
// Consumer must be aware of which type is the publisher sending but is split in diff methods for simplicity and
// will be compatible in the future if any change is included.
func Consume(q *QConn, topic server.Topic) <-chan string {
	if err := q.subscribe(topic); err != nil {
		log.Printf("cannot sub %v\n", err)
		return nil
	}

	ch := make(chan string, 1000)
	go func() {
		defer close(ch)
		for {
			b := make([]byte, 1024)
			n, err := q.c.Read(b)
			if err != nil {
				log.Printf("cannot read messege %v \n", err)
				continue
			}

			if n > 0 {
				msg, err := getMessage(b[:n])
				if err != nil {
					log.Printf("cannot get messege %v \n", err)
					continue
				}

				ch <- msg.BodyString()
				q.updateMessage(msg)
			}
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

	log.Printf("sending sub\n")
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
	b, err := m.Marshall()
	if err != nil {
		return err
	}

	_, err = q.c.Write(b)
	return err
}

func getMessage(b []byte) (server.Message, error) {
	msg := server.Message{}
	err := msg.Unmarshal(b)
	if err != nil {
		return server.Message{}, err
	}

	return msg, nil
}

func generateNextID() string {
	return uuid.NewString()
}

func generateID(prefix string, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}
