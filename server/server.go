package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type MessageFormat byte

const (
	FormatJSON   MessageFormat = 0x01
	FormatBinary MessageFormat = 0x02
)

type Server struct {
	protocol string
	port     string

	clients map[Topic][]Client
	window  *time.Ticker

	User     string
	Password string

	DB BadgerDB

	listener net.Listener

	webServer    *http.Server
	sentMessages map[Topic]*atomic.Int32
}

type Config struct {
	Protocol     string
	Port         string
	BadgerPath   string
	Duration     time.Duration
	Auth         *Auth
	InMemoryData bool

	WebServerPort string
}

type Auth struct {
	User     string
	Password string // not encrypted.
}

type Client struct {
	conn   net.Conn
	Format MessageFormat
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

	if c.WebServerPort == "" || c.WebServerPort == c.Port {
		panic("invalid web server port")
	}

	return &Server{
		protocol: c.Protocol,
		port:     c.Port,
		clients:  make(map[Topic][]Client),
		window:   time.NewTicker(time.Second * 3600),
		DB:       BadgerDB{DB: db},
		User:     user,
		Password: pass,
		webServer: &http.Server{
			Addr: c.WebServerPort,
		},
		sentMessages: make(map[Topic]*atomic.Int32),
	}, nil
}

func (s *Server) Start() error {
	l, err := net.Listen(s.protocol, s.port)
	if err != nil {
		return err
	}

	s.listener = l

	go func() {
		err = s.StartWebServer()
		if err != nil {
			log.Printf("web server failed to start: %v \n", err)
		}
	}()

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

type StatsData struct {
	Items []StatsItem `json:"items"`
}

type StatsItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *Server) printStats() (*StatsData, error) {
	stats := &StatsData{
		Items: []StatsItem{},
	}

	err := s.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(v []byte) error {
				stats.Items = append(stats.Items, StatsItem{
					Key:   key,
					Value: string(v),
				})
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *Server) handleConnections(conn net.Conn) {
	for {
		// Read format flag (1 byte)
		formatBuff := make([]byte, 1)
		_, err := io.ReadFull(conn, formatBuff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnect(conn)
				break
			}
			log.Printf("cannot read format flag %v \n", err)
			continue
		}
		format := MessageFormat(formatBuff[0])

		// Read length (4 bytes)
		lengthBuff := make([]byte, 4)
		_, err = io.ReadFull(conn, lengthBuff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnect(conn)
				break
			}
			log.Printf("cannot read message length %v \n", err)
			continue
		}
		messageLength := binary.LittleEndian.Uint32(lengthBuff)

		// Read message payload
		messageBuff := make([]byte, messageLength)
		_, err = io.ReadFull(conn, messageBuff)
		if err != nil {
			log.Printf("cannot read message body %v \n", err)
			continue
		}

		// Handle message based on detected format
		s.handleMessage(conn, messageBuff, format)
	}
}

func (s *Server) handleMessage(conn net.Conn, buff []byte, format MessageFormat) {
	var msg Message
	var err error

	switch format {
	case FormatJSON:
		msg, err = DecodeMessage(buff)
		if err != nil {
			log.Printf("cannot parse JSON message %v \n", err)
			return
		}
		fmt.Printf("decoded JSON message %s\n", msg.body)

	case FormatBinary:
		err = msg.UnmarshalBinary(buff)
		if err != nil {
			log.Printf("cannot parse binary message %v \n", err)
			return
		}

	default:
		log.Printf("unknown message format: %d \n", format)
		return
	}

	// same message handling logic for both formats
	switch msg.Type() {
	case MessageTypeNewTopic:
		s.addNewTopic(msg.Topic().Name)
	case MessageTypeNew:
		s.sendNewMessage(msg)
	case MessageTypeNewSubscriber:
		s.addNewSubscriber(conn, msg.Topic(), format)
	case MessageTypeACK:
		s.ack(msg)
	case MessageTypeAuth:
		s.doLogin(conn, msg)
	}
}

func (s *Server) sendNewMessage(message Message) {
	clients := s.clients[message.Topic()]
	if len(clients) == 0 {
		log.Printf("topic not found, actual name: %s, values in memory: %v \n", message.Topic().Name, s.clients)
		return
	}

	// write for all the clients/connections
	var payload []byte
	var err error

	for _, client := range clients {
		if FormatJSON == client.Format {
			payload, err = message.Marshall()
		} else {
			payload, err = message.MarshalBinary()
		}

		if err != nil {
			log.Printf("cannot marshall message: %v\n", err)
			return
		}

		err = binary.Write(client.conn, binary.LittleEndian, client.Format)
		if err != nil {
			log.Printf("cannot write format flag: %v\n", err)
			return
		}

		length := uint32(len(payload))
		err = binary.Write(client.conn, binary.LittleEndian, length)
		if err != nil {
			log.Printf("cannot write length: %v\n", err)
			return
		}

		_, err = client.conn.Write(payload)
		if err != nil {
			log.Printf("cannot write payload: %v\n", err)
			saveUnsentMessage(message, client.Format, s.save)
			return
		}

		if message.attempts <= 1 {
			s.save(message, client.Format)
		}

		s.incSentMessages(message.Topic())
	}
}
func (s *Server) doLogin(conn net.Conn, message Message) {
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

func (s *Server) save(message Message, format MessageFormat) {
	if err := s.DB.saveMessage(message, format); err != nil {
		log.Printf("cannot save message with id %s, %v\n", message.ID(), err)
	}
}

func (s *Server) addNewSubscriber(conn net.Conn, topic Topic, format MessageFormat) {
	s.clients[topic] = append(s.clients[topic], Client{
		conn:   conn,
		Format: format,
	})
}

func (s *Server) addNewTopic(name string) {
	s.clients[NewTopic(name)] = []Client{}
}

func (s *Server) ack(message Message) {
	if err := s.DB.updateMessageACK(message); err != nil {
		log.Printf("cannot ACK message with id %s, %v", message.ID(), err)
	}
}

func (s *Server) disconnect(conn net.Conn) {
	for topic, clients := range s.clients {
		for i, client := range clients {
			if client.conn == conn {
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

func saveUnsentMessage(msg Message, format MessageFormat, saveFn func(Message, MessageFormat)) {
	msg.IncAttempts()
	saveFn(msg, format)
}
