package server

import (
	"bytes"
	"encoding/json"
	"errors"
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
}

type Publisher struct {
}

type Consumer struct {
}

func NewServer(protocol, port string) *Server {
	return &Server{
		protocol: protocol,
		port:     port,
		format:   string(MessageFormatJSON),
		Topics:   make(map[Topic]struct{}),
		clients:  make(map[Topic][]net.Conn),
	}
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
	var v = Message{}
	err := json.NewDecoder(bytes.NewReader(buff)).Decode(&v)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
	}
	log.Println(v)
	switch v.Type {
	case MessageTypeNewTopic:
		s.addNewTopic(v.Topic.Name)
	case MessageTypeNew:
		s.newMessage(v)
	case MessageTypeNewSubscriber:
		s.addNewSubscriber(conn, v.Topic)
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
