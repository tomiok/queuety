package manager

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/tomiok/queuety/queuety/server"
	"log"
	"net"
	"time"
)

type QConn struct {
	c net.Conn
}

func Connect(protocol, addr string) (*QConn, error) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		return nil, err
	}

	qConn := QConn{
		conn,
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
	if err := q.Subscribe(t); err != nil {
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
				msg, err := GetMessage(b[:n])
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

func (q *QConn) Subscribe(t server.Topic) error {
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

func GetMessage(b []byte) (server.Message, error) {
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
