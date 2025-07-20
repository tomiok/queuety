package manager

import (
	"encoding/json"
	"net"
)

const (
	messageTypeNewTopic = "NEW_TOPIC"
)

type QConn struct {
	c net.Conn
}

type Message struct {
	Type string `json:"type"`
	Name string `json:"name"`
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

func (q *QConn) NewTopic(name string) error {
	err := q.qWrite(messageTypeNewTopic, name, nil)
	if err != nil {
		return err
	}

	return nil
}

func (q *QConn) qWrite(s, topicName string, b []byte) error {
	switch s {
	case messageTypeNewTopic:
		return q.writeNewTopic(topicName)
	}

	return nil
}

func (q *QConn) writeNewTopic(name string) error {
	msg := Message{
		Type: messageTypeNewTopic,
		Name: name,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = q.c.Write(b)
	if err != nil {
		return err
	}

	return nil
}
