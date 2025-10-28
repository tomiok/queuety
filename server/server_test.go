package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

func Test_ServerStart(t *testing.T) {
	s, err := NewServer(Config{
		Protocol:      "tcp",
		Port:          ":60000",
		BadgerPath:    "/tmp/badger_test",
		WebServerPort: ":60001",
		Duration:      10,
		Auth:          nil,
	})
	if err != nil {
		t.Fatalf("should not see an err here %v", err)
	}

	go func() {
		err = s.Start()
		if err != nil {
			t.Errorf("should not see an err here %v", err)
			return
		}
	}()
	time.Sleep(100 * time.Millisecond)
}

func Test_Server(t *testing.T) {
	db, err := NewBadger("", true)
	if err != nil {
		t.Fatalf("%v", err)
	}

	srv := Server{
		protocol: "tcp",
		port:     ":60123",
		clients:  map[Topic][]Client{},
		window:   time.NewTicker(time.Minute * 10), // IDK, hope is the same.
		DB: BadgerDB{
			DB: db,
		},
		webServer: &http.Server{
			Addr: ":60124",
		},
		listener: nil,
	}

	go func() {
		err = srv.Start()
		if err != nil {
			t.Errorf("%v", err)
			return
		}
	}()
	time.Sleep(500 * time.Millisecond)

	conn, err := net.Dial("tcp", ":60123")
	topic := NewTopic("test-Topic")
	srv.addNewTopic("test-Topic")
	srv.addNewSubscriber(conn, topic, FormatJSON)

	_msg := msg{Value: 1}
	bMsg, _ := json.Marshal(_msg)

	id := uuid.NewString()
	nextID := uuid.NewString()

	__msg := Message{}
	__msg.ID = "false-" + id
	__msg.NextID = nextID
	__msg.MType = MessageTypeNew
	__msg.Topic = topic
	__msg.Body = bMsg
	__msg.Timestamp = time.Now().Unix()

	srv.save(__msg, FormatJSON)

	time.Sleep(10 * time.Microsecond)
	err = db.View(func(txn *badger.Txn) error {
		item, rerr := txn.Get([]byte("false-" + id))
		if rerr != nil {
			return rerr
		}

		key := string(item.Key())
		if !strings.HasPrefix(key, "false") {
			return errors.New("invalid key saved")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
}

type msg struct {
	Value int `json:"value"`
}
