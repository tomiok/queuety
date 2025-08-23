package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tomiok/queuety/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"io"
	"log"
	"net"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Server struct {
	protocol string
	port     string
	format   string

	clients map[Topic][]net.Conn
	window  *time.Ticker

	User     string
	Password string

	DB        BadgerDB
	telemetry *telemetry.Telemetry

	listener net.Listener
}

type Config struct {
	Protocol     string
	Port         string
	BadgerPath   string
	Duration     time.Duration
	Auth         *Auth
	InMemoryData bool

	Telemetry *telemetry.Telemetry
}

type Auth struct {
	User     string
	Password string // not encrypted.
}

func NewServer(c Config) (*Server, error) {
	db, err := NewBadger(c.BadgerPath, c.InMemoryData)
	if err != nil {
		return nil, err
	}

	var (
		user string
		pass string
	)

	if c.Auth != nil {
		user = c.Auth.User
		pass = c.Auth.Password
	}

	if c.Telemetry == nil {
		c.Telemetry = &telemetry.Telemetry{}
	}

	return &Server{
		protocol:  c.Protocol,
		port:      c.Port,
		format:    MessageFormatJSON,
		clients:   make(map[Topic][]net.Conn),
		window:    time.NewTicker(c.Duration),
		DB:        BadgerDB{DB: db},
		User:      user,
		Password:  pass,
		telemetry: c.Telemetry,
	}, nil
}

func (s *Server) Start() error {
	l, err := net.Listen(s.protocol, s.port)
	if err != nil {
		return err
	}

	s.listener = l

	for {
		conn, errAccept := l.Accept()
		if errAccept != nil {
			log.Printf("cannot accept conn %v \n", errAccept)
			continue
		}

		go s.handleConnections(conn)
		go s.run(s.DB.checkNotDeliveredMessages)
	}
}

func (s *Server) Close() error {
	return s.listener.Close()
}

// unused by now.
func (s *Server) printStats() {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			err := s.DB.View(func(txn *badger.Txn) error {
				opts := badger.DefaultIteratorOptions
				opts.PrefetchSize = 10
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					item := it.Item()
					err := item.Value(func(v []byte) error {
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
	}
}

func (s *Server) handleConnections(conn net.Conn) {
	ctx := context.Background()
	ctx, span := s.telemetry.StartSpan(ctx, "connection.handle")
	defer span.End()

	s.telemetry.IncrementActiveConnections(ctx)
	defer s.telemetry.DecrementActiveConnections(ctx)

	for {
		buff := make([]byte, 2048)

		i, err := conn.Read(buff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnect(conn)
				break
			}
			log.Printf("cannot read message %v \n", err)
			continue
		}

		s.telemetry.IncrementBytesReceived(ctx, int64(i))

		switch s.format {
		case MessageFormatJSON:
			s.handleJSON(ctx, conn, buff[:i])
		}
	}
}
func (s *Server) handleJSON(ctx context.Context, conn net.Conn, buff []byte) {
	var msg Message
	err := json.NewDecoder(bytes.NewReader(buff)).Decode(&msg)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
	}

	switch msg.Type() {
	case MessageTypeNewTopic:
		s.addNewTopic(msg.Topic().Name)
	case MessageTypeNew:
		s.sendNewMessage(ctx, msg)
	case MessageTypeNewSubscriber:
		s.addNewSubscriber(conn, msg.Topic())
	case MessageTypeACK:
		s.ack(ctx, msg, s.telemetry)
	case MessageTypeAuth:
		s.doLogin(conn, msg)
	}
}

func (s *Server) sendNewMessage(ctx context.Context, message Message) {
	ctx, span := s.telemetry.StartSpan(ctx, "message.deliver")
	defer span.End()

	topicName := message.Topic().Name
	span.SetAttributes(
		attribute.String("topic", topicName),
		attribute.String("message_id", message.ID()),
	)

	clients := s.clients[message.Topic()]
	if len(clients) == 0 {
		s.telemetry.IncrementMessagesFailed(ctx, topicName, "no_subscribers")
		log.Printf("topic not found \n actual name: %s, values in memory: %v", topicName, s.clients)
		return
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.telemetry.RecordMessageDuration(ctx, duration, topicName, "deliver")
	}()

	// write for all the clients/connections
	for _, conn := range clients {
		b, err := message.Marshall()
		if err != nil {
			s.telemetry.IncrementMessagesFailed(ctx, topicName, "marshall_error")
			log.Printf("cannot marshall message: %v\n", err)
			return
		}

		_, err = conn.Write(b)
		if err != nil {
			s.telemetry.IncrementMessagesFailed(ctx, topicName, "write_error")
			log.Printf("cannot write in connection: %v message\n", err)
			saveUnsentMessage(ctx, s.telemetry, message, s.save)
			return
		}

		s.telemetry.IncrementBytesSent(ctx, int64(len(b)))
		s.save(ctx, message, s.telemetry)
	}

	s.telemetry.IncrementMessagesDelivered(ctx, topicName)
}

func (s *Server) doLogin(conn net.Conn, message Message) {
	fmt.Println(s)
	if !s.needAuth() {
		message.updateAuthSuccess() // no auth need means successful.
		b, err := message.Marshall()
		if err != nil {
			// just close the connection.
			_ = conn.Close()
		}
		_, _ = conn.Write(b)
		return
	}

	if !s.validateAuth(message) {
		message.updateAuthFailed()
		b, err := message.Marshall()
		if err != nil {
			// just close the connection.
			_ = conn.Close()
		}
		_, _ = conn.Write(b)
		return
	}

	message.updateAuthSuccess()
	b, err := message.Marshall()
	if err != nil {
		// just close the connection.
		_ = conn.Close()
	}
	_, _ = conn.Write(b)
	return
}

func (s *Server) validateAuth(msg Message) bool {
	return s.User == msg.User() && s.Password == msg.Password()
}

// you need to set up user and password in order to secure the server.
func (s *Server) needAuth() bool {
	if s.User != "" {
		return true
	}

	if s.Password != "" {
		return true
	}

	return false
}

func (s *Server) save(ctx context.Context, message Message, tel *telemetry.Telemetry) {
	ctx, span := tel.StartBadgerSpan(ctx, "save")
	defer span.End()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		tel.RecordBadgerDuration(ctx, duration, "save")
	}()

	if err := s.DB.saveMessage(message); err != nil {
		log.Printf("cannot save message with id %s, %v\n", message.ID(), err)
		tel.IncrementBadgerOps(ctx, "save", "error")
		return
	}

	tel.IncrementBadgerOps(ctx, "save", "success")
}

func (s *Server) addNewSubscriber(conn net.Conn, topic Topic) {
	s.clients[topic] = append(s.clients[topic], conn)
}

func (s *Server) addNewTopic(name string) {
	s.clients[NewTopic(name)] = []net.Conn{}
}

func (s *Server) ack(ctx context.Context, message Message, tel *telemetry.Telemetry) {
	ctx, span := tel.StartBadgerSpan(ctx, "ack")
	defer span.End()

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		tel.RecordBadgerDuration(ctx, duration, "ack")
	}()

	if err := s.DB.updateMessageACK(message); err != nil {
		log.Printf("cannot ACK message with id %s, %v", message.ID(), err)
		tel.IncrementBadgerOps(ctx, "ack", "error")
		return
	}
	tel.IncrementBadgerOps(ctx, "ack", "success")
}

func (s *Server) disconnect(conn net.Conn) {
	for topic, clients := range s.clients {
		for i, client := range clients {
			if client == conn {
				s.clients[topic] = append(clients[:i], clients[i+1:]...)
				log.Printf("client removed in topic: %s", topic.Name)
				break
			}
		}

		if len(s.clients[topic]) == 0 {
			delete(s.clients, topic)
			log.Printf("%s is empty, deleting", topic.Name)
		}
	}

	err := conn.Close()
	if err != nil {
		log.Printf("cannot close deleted connection %v", err)
	}
}

func saveUnsentMessage(ctx context.Context, tel *telemetry.Telemetry, msg Message, saveFn func(context.Context, Message, *telemetry.Telemetry)) {
	msg.IncAttempts()
	saveFn(ctx, msg, tel)
}
