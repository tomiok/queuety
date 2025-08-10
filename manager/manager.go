package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/tomiok/queuety/server"
	"log"
	"net"
	"time"
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
		msg := server.Message{
			ID:        generateNextID(),
			Type:      server.MessageTypeAuth,
			User:      auth.User,
			Password:  auth.Pass,
			Timestamp: time.Now().Unix(),
		}

		b, _err := msg.Marshall()
		if _err != nil {
			return nil, _err
		}

		_, _ = conn.Write(b)

		//listen to the message back.
		var buff = make([]byte, 1024)
		n, errRead := conn.Read(buff)
		if errRead != nil {
			return nil, errRead
		}

		var msgResponse server.Message
		if err = json.Unmarshal(buff[:n], &msgResponse); err != nil {
			return nil, err
		}

		if msgResponse.Type == server.MessageAuthFailed {
			return nil, errors.New("authentication failed")
		}

		return &qConn, nil
	}

	return &qConn, nil
}

func (q *QConn) NewTopic(name string) (server.Topic, error) {
	m := server.Message{
		ID:        uuid.NewString(),
		Type:      server.MessageTypeNewTopic,
		Topic:     server.NewTopic(name),
		Timestamp: time.Now().UnixMilli(),
		ACK:       false,
	}
	err := q.qWrite(m)
	if err != nil {
		return server.Topic{}, err
	}

	return server.Topic{
		Name: name,
	}, nil
}

func (q *QConn) Publish(t server.Topic, msg string) error {
	nextID := generateNextID()
	m := server.Message{
		ID:         generateID(server.MsgPrefixFalse, nextID),
		NextID:     nextID,
		Type:       server.MessageTypeNew,
		Topic:      t,
		BodyString: msg,
		Timestamp:  time.Now().Unix(),
		ACK:        false,
	}

	return q.qWrite(m)
}

func (q *QConn) PublishJSON(t server.Topic, msg string) error {
	nextID := generateNextID()
	m := server.Message{
		ID:        generateID(server.MsgPrefixFalse, nextID),
		NextID:    nextID,
		Type:      server.MessageTypeNew,
		Topic:     t,
		Body:      json.RawMessage(msg),
		Timestamp: time.Now().Unix(),
		ACK:       false,
	}

	return q.qWrite(m)
}

func (q *QConn) Consume(t server.Topic) <-chan server.Message {
	if err := q.subscribe(t); err != nil {
		log.Printf("cannot sub %v\n", err)
		return nil
	}

	ch := make(chan server.Message, 1000)
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

				ch <- msg
				q.updateMessage(msg)
			}
		}
	}()

	return ch
}

func (q *QConn) subscribe(t server.Topic) error {
	id := generateNextID()
	m := server.Message{
		ID:        id,
		NextID:    id,
		Type:      server.MessageTypeNewSubscriber,
		Topic:     t,
		Timestamp: time.Now().UnixMilli(),
		ACK:       false,
	}
	log.Printf("sending sub\n")
	return q.qWrite(m)
}

func (q *QConn) unsubscribe() error {
	return nil
}

func (q *QConn) updateMessage(msg server.Message) {
	msg.Type = server.MessageTypeACK
	msg.ACK = true
	if err := q.writeMessage(msg); err != nil {
		log.Printf("cannot send ACK confirmation, message id %s \n", msg.ID)
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
	var msg = server.Message{}
	err := json.Unmarshal(b, &msg)
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
