package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/tomiok/queuety/server/observability"
	"go.opentelemetry.io/otel/trace"
)

type BadgerDB struct {
	*badger.DB
}

func NewBadger(path string, inMemory bool) (*badger.DB, error) {
	if inMemory {
		return badger.Open(badger.DefaultOptions("").WithInMemory(true))
	}

	if path == "" {
		path = "/data/badger"
	}
	return badger.Open(badger.DefaultOptions(path))
}

// saveMessage will store the message at the first time, the id should start with false since is the
// 1st time we are storing the message.
func (b BadgerDB) saveMessage(message Message) error {
	_, span := observability.StartSpan(
		context.Background(),
		"badger_save_message",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in saveMessage: %v\n", r)
			observability.EndSpan(span, fmt.Errorf("panic: %v", r))
		}
	}()
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("message.id", message.ID()),
		observability.StringAttribute("topic.name", message.Topic().Name),
	)

	if !strings.HasPrefix(message.ID(), MsgPrefixFalse) {
		observability.IncrementBadgerOperation("saveMessage", "error")
		return errors.New("invalid key, should start with 'false'")
	}

	err := b.DB.Update(func(txn *badger.Txn) error {
		message.IncAttempts() // store the messages with attempt 1.
		msgBytes, err := message.Marshall()
		if err != nil {
			observability.IncrementBadgerOperation("saveMessage", "error")
			return err
		}

		err = txn.Set([]byte(message.ID()), msgBytes)
		if err != nil {
			observability.IncrementBadgerOperation("saveMessage", "error")
			return err
		}

		return nil
	})

	if err != nil {
		observability.IncrementBadgerOperation("saveMessage", "error")
		return err
	}

	observability.IncrementBadgerOperation("saveMessage", "success")
	return nil

}

func (b BadgerDB) updateMessageACK(message Message) error {
	_, span := observability.StartSpan(
		context.Background(),
		"badger_update_message_ack",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in updateMessageACK: %v\n", r)
			observability.EndSpan(span, fmt.Errorf("panic: %v", r))
		}
	}()
	defer observability.EndSpan(span, nil)

	observability.AddSpanAttributes(span,
		observability.StringAttribute("message.id", message.ID()),
		observability.StringAttribute("topic.name", message.Topic().Name),
	)

	err := b.DB.Update(func(txn *badger.Txn) error {
		// delete entry with old key.
		if err := txn.Delete([]byte(message.ID())); err != nil {
			log.Printf("cannot delete message with ID %s\n", message.ID())
			observability.IncrementBadgerOperation("updateMessageACK", "error")
			return err
		}

		message.updateACK()
		msgBytes, err := message.Marshall()
		if err != nil {
			observability.IncrementBadgerOperation("updateMessageACK", "error")
			return err
		}

		err = txn.Set([]byte(message.ID()), msgBytes)
		if err != nil {
			observability.IncrementBadgerOperation("updateMessageACK", "error")
			return err
		}

		return nil
	})

	if err != nil {
		observability.IncrementBadgerOperation("updateMessageACK", "error")
		return err
	}
	observability.IncrementBadgerOperation("updateMessageACK", "success")
	return nil

}

func (b BadgerDB) checkNotDeliveredMessages() ([]Message, error) {
	_, span := observability.StartSpan(
		context.Background(),
		"badger_check_not_delivered_messages",
		observability.WithSpanKind(trace.SpanKindInternal),
	)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered from panic in checkNotDeliveredMessages: %v\n", r)
			observability.EndSpan(span, fmt.Errorf("panic: %v", r))
		}
	}()
	defer observability.EndSpan(span, nil)

	var messages []Message
	topicsChecked := make(map[string]bool)
	err := b.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(MsgPrefixFalse)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			msg := Message{}
			err := item.Value(func(v []byte) error {
				err := json.Unmarshal(v, &msg)
				if err != nil {
					observability.IncrementBadgerOperation("checkNotDeliveredMessages", "error")
					return err
				}

				if msg.Attempts() <= 3 {
					msg.IncAttempts()
					messages = append(messages, msg)

					// Register undelivered message topics
					topicsChecked[msg.Topic().Name] = true
				}

				return nil
			})
			if err != nil {
				log.Printf("cannot get message with id %s, %v \n", k, err)
				observability.IncrementBadgerOperation("checkNotDeliveredMessages", "error")
				continue
			}
		}

		return nil
	})

	var topicsList []string
	for topic := range topicsChecked {
		topicsList = append(topicsList, topic)
	}

	observability.AddSpanAttributes(span,
		observability.IntAttribute("messages.count", len(messages)),
		observability.StringAttribute("topics.checked", strings.Join(topicsList, ",")),
	)

	if err == nil {
		observability.IncrementBadgerOperation("checkNotDeliveredMessages", "success")
	} else {
		observability.IncrementBadgerOperation("checkNotDeliveredMessages", "error")
	}

	return messages, err
}
