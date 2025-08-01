package server

import (
	"context"
	"encoding/json"
	badger "github.com/dgraph-io/badger/v4"
	"log"
)

type BadgerDB struct {
	*badger.DB
}

func NewBadger(path string) (*badger.DB, error) {
	if path == "" {
		path = "/tmp/badger"
	}
	return badger.Open(badger.DefaultOptions(path))
}

func (b BadgerDB) SaveMessage(_ context.Context, message Message) error {
	return b.DB.Update(func(txn *badger.Txn) error {
		msgBytes, err := message.Marshall()
		if err != nil {
			return err
		}

		err = txn.Set([]byte(message.ID), msgBytes)
		if err != nil {
			return err
		}

		return nil
	})
}

func (b BadgerDB) UpdateMessageACK(_ context.Context, message Message) error {
	return b.DB.Update(func(txn *badger.Txn) error {
		// delete entry with old key.
		if err := txn.Delete([]byte(message.ID)); err != nil {
			log.Printf("cannot delete message with ID %s", message.ID)
		}

		message.ID = message.NextID
		message.ACK = true
		msgBytes, err := message.Marshall()
		if err != nil {
			return err
		}

		err = txn.Set([]byte(message.ID), msgBytes)
		if err != nil {
			return err
		}

		return nil
	})
}

func (b BadgerDB) CheckNotDeliveredMessages() ([]Message, error) {
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
				err := json.Unmarshal(v, &messages)
				if err != nil {
					return err
				}
				messages = append(messages, msg)
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
