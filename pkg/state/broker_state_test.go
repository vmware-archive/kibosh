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

package state_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cf-platform-eng/kibosh/pkg/state"
)

type testStruct struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

var _ = Describe("Broker State", func() {
	var kvs state.KeyValueStore
	var dataDir string

	BeforeEach(func() {
		kvs = state.NewKeyValueStore()
		dir, err := ioutil.TempDir("", "kvs_test")
		Expect(err).To(BeNil())
		dataDir = dir
		err = kvs.Open(dir)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		err := os.RemoveAll(dataDir)
		Expect(err).To(BeNil())
	})

	Context("basic functionality", func() {
		It("put and item and get it back", func() {
			ts1 := testStruct{"hi", 101}
			err := kvs.PutJson("key1", ts1)
			Expect(err).To(BeNil())
			var ts2 testStruct
			err = kvs.GetJson("key1", &ts2)
			Expect(err).To(BeNil())
			Expect(ts2).To(Equal(ts1))
		})

		It("returns key not found for non-existent key", func() {
			var ts testStruct
			err := kvs.GetJson("key3", &ts)
			Expect(err).To(Equal(state.KeyNotFoundError))
		})

		It("put an item, delete it, and make sure its gone", func() {
			ts1 := testStruct{"bye", 42}
			err := kvs.PutJson("keyA", ts1)
			Expect(err).To(BeNil())
			var ts2 testStruct
			err = kvs.GetJson("keyA", &ts2)
			Expect(err).To(BeNil())
			err = kvs.Delete("keyA")
			Expect(err).To(BeNil())
			err = kvs.GetJson("keyA", &ts2)
			Expect(err).To(Equal(state.KeyNotFoundError))
		})
	})
})
