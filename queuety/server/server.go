package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"io"
	"log"
	"net"
)

type Server struct {
	protocol string
	port     string
	format   string

	clients map[Topic][]net.Conn

	Topics map[Topic]struct{}
	DB     BadgerDB
}

type Publisher struct {
}

type Consumer struct {
}

func NewServer(protocol, port string) (*Server, error) {
	badger, err := NewBadger()
	if err != nil {
		return nil, err
	}

	return &Server{
		protocol: protocol,
		port:     port,
		format:   string(MessageFormatJSON),
		Topics:   make(map[Topic]struct{}),
		clients:  make(map[Topic][]net.Conn),
		DB:       BadgerDB{DB: badger},
	}, nil
}

func (s *Server) Start() error {
	l, err := net.Listen(s.protocol, s.port)

	if err != nil {
		return err
	}

	for {
		conn, errAccept := l.Accept()
		if errAccept != nil {
			log.Printf("cannot accpet conn %v \n", errAccept)
			continue
		}

		go s.handleConnections(conn)
	}
}

func (s *Server) handleConnections(conn net.Conn) {
	for {
		buff := make([]byte, 2048)

		i, err := conn.Read(buff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}

			log.Printf("cannot read message %v \n", err)
			continue
		}

		switch s.format {
		case string(MessageFormatJSON):
			s.handleJSON(conn, buff[:i])
		}
	}

}

func (s *Server) handleJSON(conn net.Conn, buff []byte) {
	msg := Message{}
	err := json.NewDecoder(bytes.NewReader(buff)).Decode(&msg)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
	}
	switch msg.Type {
	case MessageTypeNewTopic:
		s.addNewTopic(msg.Topic.Name)
	case MessageTypeNew:
		s.newMessage(msg)
		s.save(msg)
	case MessageTypeNewSubscriber:
		s.addNewSubscriber(conn, msg.Topic)
	case MessageTypeACK:
		s.ack(msg)
	}
}

func (s *Server) save(message Message) {
	if err := s.DB.SaveMessage(context.Background(), message); err != nil {
		log.Printf("cannot save message with id %s, %v\n", message.ID, err)
	}

	err := s.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Println(err)
	}
}

func (s *Server) ack(message Message) {
	if err := s.DB.UpdateMessageACK(context.Background(), message); err != nil {
		log.Printf("cannot ACK message with id %s, %v", message.ID, err)
	}
}

func (s *Server) addNewSubscriber(conn net.Conn, topic Topic) {
	s.clients[topic] = append(s.clients[topic], conn)
}

func (s *Server) addNewTopic(name string) {
	s.Topics[NewTopic(name)] = struct{}{}
}

func (s *Server) newMessage(v Message) {
	clients := s.clients[v.Topic]
	if len(clients) == 0 {
		log.Printf("topic not found \n actual name: %s, values in memory: %v", v.Topic.Name, s.clients)
		return
	}

	// write for all the clients/connections
	for _, conn := range clients {
		b, err := v.Marshall()
		if err != nil {
			log.Printf("cannot marshall message: %v\n", err)
			return
		}

		_, err = conn.Write(b)
		if err != nil {
			log.Printf("cannot marshall message: %v\n", err)
			return
		}
	}
}
