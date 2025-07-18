package queuety

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
)

type Server struct {
	protocol string
	port     string
	format   string
}

func NewServer(protocol, port string) Server {
	return Server{
		protocol: protocol,
		port:     port,
		format:   "JSON",
	}
}

func (s Server) Start() error {
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

func (s Server) handleConnections(conn net.Conn) {
	buff := make([]byte, 2048)

	i, err := conn.Read(buff)
	if err != nil {
		log.Printf("cannot read message %v \n", err)
	}

	switch s.format {
	case "JSON":
		handleJSON(buff[:i])
	}

}

func handleJSON(buff []byte) {
	var v any
	err := json.NewDecoder(bytes.NewReader(buff)).Decode(&v)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
	}

}
