package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/google/uuid"
)

func Test_ServerStart(t *testing.T) {
	s, err := NewServer(Config{
		Protocol:   "tcp",
		Port:       ":60000",
		BadgerPath: "/tmp/badger_test",
		Duration:   10,
		Auth:       nil,
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
	time.Sleep(500 * time.Millisecond)

	err = s.Close()
	if err != nil {
		t.Fatalf("should not see an err here %v", err)
	}
}

func Test_Server(t *testing.T) {
	db, err := NewBadger("", true)
	if err != nil {
		t.Fatalf("%v", err)
	}

	srv := Server{
		protocol: "tcp",
		port:     ":60123",
		clients:  map[Topic][]net.Conn{},
		window:   time.NewTicker(time.Minute * 10), // IDK, hope is the same.
		format:   MessageFormatJSON,
		DB: BadgerDB{
			DB: db,
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
	topic := NewTopic("test-topic")
	srv.addNewTopic("test-topic")
	srv.addNewSubscriber(conn, topic)

	_msg := msg{Value: 1}
	bMsg, _ := json.Marshal(_msg)

	id := uuid.NewString()
	nextID := uuid.NewString()

	msg := NewMessageBuilder().
		WithID("false-" + id).
		WithNextID(nextID).
		WithType(MessageTypeNew).
		WithTopic(topic).
		WithBody(bMsg).
		WithTimestamp(time.Now().Unix()).
		WithAck(false).
		Build()

	srv.save(context.TODO(), msg)

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
