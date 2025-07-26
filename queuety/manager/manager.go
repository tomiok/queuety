package manager

import (
	"encoding/json"
	"fmt"
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
	m := server.Message{
		Type:      server.MessageTypeNew,
		Topic:     t,
		Body:      json.RawMessage(msg),
		Timestamp: time.Now().Unix(),
		ACK:       false,
	}

	return q.qWrite(m)
}

func (q *QConn) PublishJSON(t server.Topic, msg string) error {
	m := server.Message{
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

	go func() {
		for {
			b := make([]byte, 1024)
			n, err := q.c.Read(b)
			if err != nil {
				panic(err)
			}

			if len(b) > 0 {
				fmt.Println("got a message")
				fmt.Println(GetMessage(b[:n]))
			}
		}
	}()

	return nil
}

func (q *QConn) Subscribe(t server.Topic) error {
	m := server.Message{
		Type:      server.MessageTypeNewSubscriber,
		Topic:     t,
		Timestamp: time.Now().UnixMilli(),
		ACK:       false,
	}
	log.Printf("sending sub\n")
	return q.qWrite(m)
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
