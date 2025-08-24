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
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tomiok/queuety/server/observability"
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
	// Incrementar contador de conexiones
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
				break
			}

			log.Printf("cannot read message %v \n", err)
			continue
		}

		switch s.format {
		case MessageFormatJSON:
			s.handleJSON(conn, buff[:i])
		}
	}
}

func (s *Server) handleJSON(conn net.Conn, buff []byte) {
	startTime := time.Now()
	msg, err := DecodeMessage(buff)
	if err != nil {
		log.Printf("cannot parse message %v \n", err)
		return
	}

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

	// Observar tiempo de procesamiento
	processingTime := time.Since(startTime).Seconds()
	observability.ObserveMessageProcessingTime(msg.Topic().Name, operation, processingTime)
}

func (s *Server) sendNewMessage(message Message) {
	clients := s.clients[message.Topic()]
	if len(clients) == 0 {
		log.Printf("topic not found, actual name: %s, values in memory: %v", message.Topic().Name, s.clients)
		return
	}

	// Incrementar métrica de mensajes publicados
	observability.IncrementPublishedMessages(message.Topic().Name)

	// write for all the clients/connections
	for _, conn := range clients {
		b, err := message.Marshall()
		if err != nil {
			log.Printf("cannot marshall message: %v\n", err) // unclear error, don't want to re-intent.
			return
		}

		_, err = conn.Write(b)
		if err != nil {
			log.Printf("cannot write in connection: %v message\n", err)

			// Incrementar métrica de mensajes fallidos
			observability.IncrementFailedMessages(message.Topic().Name)

			saveUnsentMessage(message, s.save)
			return
		}

		// Incrementar métrica de mensajes entregados
		observability.IncrementDeliveredMessages(message.Topic().Name)

		fmt.Println("saving message")
		s.save(message)

		// check if the message was saved
		s.incSentMessages(message.Topic())
	}
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

		// Incrementar métrica de intento de autenticación fallido
		observability.IncrementAuthAttempt("failed")
		return
	}

	message.updateAuthSuccess()
	b, err := message.Marshall()
	if err != nil {
		// just close the connection.
		_ = conn.Close()
	}
	_, _ = conn.Write(b)

	// Incrementar métrica de intento de autenticación exitoso
	observability.IncrementAuthAttempt("success")
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

func (s *Server) save(message Message) {
	if err := s.DB.saveMessage(message); err != nil {
		log.Printf("cannot save message with id %s, %v\n", message.ID(), err)
	}
}

func (s *Server) addNewSubscriber(conn net.Conn, topic Topic) {
	s.clients[topic] = append(s.clients[topic], conn)

	// Actualizar métrica de suscriptores por tema
	observability.UpdateSubscribersCount(topic.Name, len(s.clients[topic]))
}

func (s *Server) addNewTopic(name string) {
	s.clients[NewTopic(name)] = []net.Conn{}

	// Actualizar métrica de temas activos
	observability.UpdateActiveTopicsCount(len(s.clients))
}

func (s *Server) ack(message Message) {
	if err := s.DB.updateMessageACK(message); err != nil {
		log.Printf("cannot ACK message with id %s, %v", message.ID(), err)
	}
}

func (s *Server) disconnect(conn net.Conn) {
	for topic, clients := range s.clients {
		for i, client := range clients {
			if client == conn {
				s.clients[topic] = append(clients[:i], clients[i+1:]...)
				log.Printf("client removed in topic: %s", topic.Name)

				// Actualizar métrica de suscriptores por tema
				observability.UpdateSubscribersCount(topic.Name, len(s.clients[topic]))
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

	// Actualizar métrica de temas activos
	observability.UpdateActiveTopicsCount(len(s.clients))
}

func saveUnsentMessage(msg Message, saveFn func(m Message)) {
	msg.IncAttempts()
	saveFn(msg)
}

func (s *Server) StartWebServer() error {
	mux := http.NewServeMux()

	// Verificar variable de entorno para métricas de Prometheus
	metricsEnabled := os.Getenv("QUEUETY_METRICS_ENABLED")
	if metricsEnabled == "" || metricsEnabled == "true" {
		// Configurar el endpoint de métricas de Prometheus solo si está habilitado
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
