package server

import (
	"context"
	badger "github.com/dgraph-io/badger/v4"
)

type BadgerDB struct {
	*badger.DB
}

func NewBadger() (*badger.DB, error) {
	return badger.Open(badger.DefaultOptions("/tmp/badger"))
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
