package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tomiok/queuety/server/observability"
	"go.opentelemetry.io/otel/trace"
)

type Server struct {
	protocol string
	port     string
	format   string

	clients map[Topic][]net.Conn
	window  *time.Ticker

	User     string
	Password string

	DB BadgerDB

	listener net.Listener

	webServer    *http.Server
	sentMessages map[Topic]*atomic.Int32

	// Campos para rastrear conexiones activas
	mu                sync.Mutex
	activeConnections int
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

	return &Server{
		protocol: c.Protocol,
		port:     c.Port,
		format:   MessageFormatJSON,
		clients:  make(map[Topic][]net.Conn),
		window:   time.NewTicker(c.Duration),
		DB:       BadgerDB{DB: db},
		User:     user,
		Password: pass,

		webServer: &http.Server{
			Addr: net.JoinHostPort("", c.WebServerPort),
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
						fmt.Printf("value=%s\n", v)
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
	_, span := observability.StartSpan(
		context.Background(),
		"handle_connections",
		observability.WithSpanKind(trace.SpanKindServer),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("client.remote_addr", conn.RemoteAddr().String()),
		observability.StringAttribute("client.local_addr", conn.LocalAddr().String()),
	)

	s.mu.Lock()
	s.activeConnections++
	observability.UpdateActiveConnectionsCount(s.activeConnections)
	s.mu.Unlock()

	defer func() {
		// Decrementar contador de conexiones al finalizar
		s.mu.Lock()
		s.activeConnections--
		observability.UpdateActiveConnectionsCount(s.activeConnections)
		s.mu.Unlock()
	}()

	for {
		buff := make([]byte, 2048)

		i, err := conn.Read(buff)
		if err != nil {
			if errors.Is(err, io.EOF) {
				s.disconnect(conn)
				observability.EndSpan(span, err)
				break
			}

			log.Printf("cannot read message %v \n", err)
			observability.EndSpan(span, err)
			continue
		}

		switch s.format {
		case MessageFormatJSON:
			s.handleJSON(conn, buff[:i])
		}
	}
}

func (s *Server) handleJSON(conn net.Conn, buff []byte) {
	_, span := observability.StartSpan(
		context.Background(),
		"handle_json_message",
		observability.WithSpanKind(trace.SpanKindServer),
	)
	defer observability.EndSpan(span, nil)

	startTime := time.Now()
	msg, err := DecodeMessage(buff)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
		observability.EndSpan(span, err)
		return
	}

	observability.AddSpanAttributes(span,
		observability.StringAttribute("topic.name", msg.Topic().Name),
	)

	var operation string
	switch msg.Type() {
	case MessageTypeNewTopic:
		operation = "new_topic"
		s.addNewTopic(msg.Topic().Name)
	case MessageTypeNew:
		operation = "new_message"
		s.sendNewMessage(msg)
	case MessageTypeNewSubscriber:
		operation = "new_subscriber"
		s.addNewSubscriber(conn, msg.Topic())
	case MessageTypeACK:
		operation = "ack"
		s.ack(msg)
	case MessageTypeAuth:
		operation = "auth"
		s.doLogin(conn, msg)
	default:
		operation = "unknown"
	}

	observability.AddSpanAttributes(span,
		observability.StringAttribute("operation", operation),
	)

	// Observar tiempo de procesamiento
	processingTime := time.Since(startTime).Seconds()
	observability.ObserveMessageProcessingTime(msg.Topic().Name, operation, processingTime)
}

func (s *Server) sendNewMessage(message Message) {
	_, span := observability.StartSpan(
		context.Background(),
		"send_message",
		observability.WithSpanKind(trace.SpanKindProducer),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("topic.name", message.Topic().Name),
		observability.StringAttribute("message.id", message.ID()),
	)

	clients := s.clients[message.Topic()]
	if len(clients) == 0 {
		log.Printf("topic not found, actual name: %s, values in memory: %v", message.Topic().Name, s.clients)
		observability.EndSpan(span, fmt.Errorf("topic not found: %s", message.Topic().Name))
		return
	}

	observability.IncrementPublishedMessages(message.Topic().Name)

	// write for all the clients/connections
	for _, conn := range clients {
		b, err := message.Marshall()
		if err != nil {
			observability.EndSpan(span, fmt.Errorf("marshall error: %v", err))
			return
		}

		_, err = conn.Write(b)
		if err != nil {
			log.Printf("cannot write in connection: %v message\n", err)

			observability.IncrementFailedMessages(message.Topic().Name)

			observability.EndSpan(span, fmt.Errorf("write error: %v", err))
			saveUnsentMessage(message, s.save)
			return
		}

		observability.IncrementDeliveredMessages(message.Topic().Name)

		fmt.Println("saving message")
		s.save(message)

		// check if the message was saved
		s.incSentMessages(message.Topic())
	}
}

func (s *Server) doLogin(conn net.Conn, message Message) {
	_, span := observability.StartSpan(
		context.Background(),
		"do_login",
		observability.WithSpanKind(trace.SpanKindServer),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("user.attempt", message.User()),
		observability.StringAttribute("client.remote_addr", conn.RemoteAddr().String()),
	)

	fmt.Println(s)
	if !s.needAuth() {
		message.updateAuthSuccess() // no auth need means successful.
		b, err := message.Marshall()
		if err != nil {
			// just close the connection.
			_ = conn.Close()
			observability.EndSpan(span, err)
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
			observability.EndSpan(span, err)
		}
		_, _ = conn.Write(b)

		observability.IncrementAuthAttempt("failed")
		observability.EndSpan(span, fmt.Errorf("authentication failed"))
		return
	}

	message.updateAuthSuccess()
	b, err := message.Marshall()
	if err != nil {
		// just close the connection.
		_ = conn.Close()
		observability.EndSpan(span, err)
	}
	_, _ = conn.Write(b)

	observability.IncrementAuthAttempt("success")
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

func (s *Server) save(message Message) {
	if err := s.DB.saveMessage(message); err != nil {
		log.Printf("cannot save message with id %s, %v\n", message.ID(), err)
	}
}

func (s *Server) addNewSubscriber(conn net.Conn, topic Topic) {
	_, span := observability.StartSpan(
		context.Background(),
		"add_subscriber",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("topic.name", topic.Name),
	)

	s.clients[topic] = append(s.clients[topic], conn)

	observability.UpdateSubscribersCount(topic.Name, len(s.clients[topic]))
}

func (s *Server) addNewTopic(name string) {
	_, span := observability.StartSpan(
		context.Background(),
		"add_topic",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("topic.name", name),
	)

	s.clients[NewTopic(name)] = []net.Conn{}

	observability.UpdateActiveTopicsCount(len(s.clients))
}

func (s *Server) ack(message Message) {
	_, span := observability.StartSpan(
		context.Background(),
		"message_ack",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("message.id", message.ID()),
		observability.StringAttribute("topic.name", message.Topic().Name),
	)

	if err := s.DB.updateMessageACK(message); err != nil {
		log.Printf("cannot ACK message with id %s, %v", message.ID(), err)

		observability.EndSpan(span, err)
	}
}

func (s *Server) disconnect(conn net.Conn) {
	_, span := observability.StartSpan(
		context.Background(),
		"disconnect",
		observability.WithSpanKind(trace.SpanKindServer),
	)
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("client.remote_addr", conn.RemoteAddr().String()),
		observability.StringAttribute("client.local_addr", conn.LocalAddr().String()),
	)

	var disconnectedTopics []string
	for topic, clients := range s.clients {
		for i, client := range clients {
			if client == conn {
				s.clients[topic] = append(clients[:i], clients[i+1:]...)
				log.Printf("client removed in topic: %s", topic.Name)

				disconnectedTopics = append(disconnectedTopics, topic.Name)
				observability.AddSpanAttributes(span,
					observability.StringAttribute("topic.name", topic.Name),
				)

				observability.UpdateSubscribersCount(topic.Name, len(s.clients[topic]))
				break
			}
		}

		if len(s.clients[topic]) == 0 {
			delete(s.clients, topic)
			log.Printf("%s is empty, deleting", topic.Name)
		}
	}

	if len(disconnectedTopics) > 0 {
		observability.AddSpanAttributes(span,
			observability.StringAttribute("disconnected_topics", strings.Join(disconnectedTopics, ",")),
		)
	}

	err := conn.Close()
	if err != nil {
		log.Printf("cannot close deleted connection %v", err)
		observability.EndSpan(span, err)
	}

	observability.UpdateActiveTopicsCount(len(s.clients))
}

func saveUnsentMessage(msg Message, saveFn func(m Message)) {
	msg.IncAttempts()
	saveFn(msg)
}

func (s *Server) StartWebServer() error {
	mux := http.NewServeMux()

	metricsEnabled := os.Getenv("QUEUETY_PROM_METRICS_ENABLED")
	if metricsEnabled == "" || metricsEnabled == "true" {
		mux.Handle("/metrics", promhttp.Handler())
	}

	// Agregar endpoint de stats
	mux.HandleFunc("/stats", s.handleStats)

	s.webServer.Handler = mux

	// Iniciar el servidor web
	log.Printf("Starting web server on %s", s.webServer.Addr)
	return s.webServer.ListenAndServe()
}

func (s *Server) ShutdownWebServer() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.webServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}
