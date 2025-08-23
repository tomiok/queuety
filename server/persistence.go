package server

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/dgraph-io/badger/v4"
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
	if !strings.HasPrefix(message.ID(), MsgPrefixFalse) {
		return errors.New("invalid key, should start with 'false'")
	}

	return b.DB.Update(func(txn *badger.Txn) error {
		message.IncAttempts() // store the messages with attempt 1.
		msgBytes, err := message.Marshall()
		if err != nil {
			return err
		}

		err = txn.Set([]byte(message.ID()), msgBytes)
		if err != nil {
			return err
		}

		return nil
	})
}

func (b BadgerDB) updateMessageACK(message Message) error {
	return b.DB.Update(func(txn *badger.Txn) error {
		// delete entry with old key.
		if err := txn.Delete([]byte(message.ID())); err != nil {
			log.Printf("cannot delete message with ID %s", message.ID())
		}

		message.updateACK()
		msgBytes, err := message.Marshall()
		if err != nil {
			return err
		}

		err = txn.Set([]byte(message.ID()), msgBytes)
		if err != nil {
			return err
		}

		return nil
	})
}

func (b BadgerDB) checkNotDeliveredMessages() ([]Message, error) {
	var messages []Message
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
					return err
				}

				if msg.Attempts() <= 3 {
					msg.IncAttempts()
					messages = append(messages, msg)
				}

				return nil
			})
			if err != nil {
				log.Printf("cannot get message with id %s, %v \n", k, err)
				continue
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return messages, nil
}
