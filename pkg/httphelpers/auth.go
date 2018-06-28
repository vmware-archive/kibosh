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

package httphelpers

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

type AuthFilter interface {
	Filter(handler http.Handler) http.Handler
	CheckAuth(req *http.Request) bool
}

type authFilter struct {
	adminPassword string
	adminUsername string
}

func NewAuthFilter(adminUsername string, adminPassword string) AuthFilter {
	return &authFilter{
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}
}

func (a *authFilter) Filter(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.CheckAuth(r) {
			handler.ServeHTTP(w, r)
		} else {
			w.Header().Add("WWW-Authenticate", "Basic realm=kibosh")
			w.WriteHeader(401)
		}
	})
}

func (a *authFilter) CheckAuth(req *http.Request) bool {
	validAuth := BasicAuthHeaderVal(a.adminUsername, a.adminPassword)
	return req.Header.Get("Authorization") == validAuth
}

func BasicAuthHeaderVal(user string, pass string) string {
	encodedAuth := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", user, pass)),
	)
	return fmt.Sprintf("Basic %s", encodedAuth)
}

func AddBasicAuthHeader(r *http.Request, user string, pass string) {
	r.Header.Set("Authorization", BasicAuthHeaderVal(user, pass))
}
