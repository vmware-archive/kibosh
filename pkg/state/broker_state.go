// kibosh
//
// Copyright (c) 2017-Present Pivotal Software, Inc. All Rights Reserved.
//
// This program and the accompanying materials are made available under the terms of the under the Apache License,
// Version 2.0 (the "License”); you may not use this file except in compliance with the License. You may
// obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the
// License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing permissions and
// limitations under the License.

package state

import (
	"encoding/json"

	"github.com/dgraph-io/badger"
)

var KeyNotFoundError = badger.ErrKeyNotFound

//go:generate counterfeiter ./ KeyValueStore
type KeyValueStore interface {
	Open(dataDir string) error

	// put val marshaled with json.Marshal()
	PutJson(key string, val interface{}) error

	// get val unmarshaled with json.Unmarshal()
	GetJson(key string, val interface{}) error

	Delete(key string) error

	Close()
}

type keyValueStore struct {
	db *badger.DB
}

func NewKeyValueStore() KeyValueStore {
	return &keyValueStore{}
}

func (kvs *keyValueStore) Open(dataDir string) error {
	opts := badger.DefaultOptions
	opts.Dir = dataDir
	opts.ValueDir = dataDir
	var err error
	kvs.db, err = badger.Open(opts)
	return err
}

func (kvs *keyValueStore) Close() {
	kvs.db.Close()
}

func (kvs *keyValueStore) PutJson(key string, val interface{}) error {
	bytes, err := json.Marshal(val)

	if err == nil {
		err = kvs.db.Update(func(txn *badger.Txn) error {
			err := txn.Set([]byte(key), bytes)
			return err
		})
	}
	return err
}

func (kvs *keyValueStore) GetJson(key string, val interface{}) error {
	err := kvs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))

		if err == nil {
			bytes, e := item.Value()

			if e != nil {
				return e
			}

			err = json.Unmarshal(bytes, val)
		}

		return err
	})

	return err
}

func (kvs *keyValueStore) Delete(key string) error {
	return kvs.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
