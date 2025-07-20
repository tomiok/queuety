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

	publishers []Publisher
	consumers  []Consumer

	Topics map[string]struct{}
}

type Publisher struct {
}

type Consumer struct {
}

func NewServer(protocol, port string) *Server {
	return &Server{
		protocol: protocol,
		port:     port,
		format:   "JSON",
		Topics:   make(map[string]struct{}),
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
		case "JSON":
			s.handleJSON(buff[:i])
		}
	}

}

func (s *Server) handleJSON(buff []byte) {
	var v = make(map[string]any)
	err := json.NewDecoder(bytes.NewReader(buff)).Decode(&v)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
	}
	log.Println(v)
	_type := v["type"]
	switch _type {
	case "NEW_TOPIC":
		s.addNewTopic(v["name"].(string))
	}

	log.Println(s.Topics)
}

func (s *Server) addNewTopic(name string) {
	s.Topics[name] = struct{}{}
}
