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

package cf

import "github.com/cloudfoundry-community/go-cfclient"

//go:generate counterfeiter ./ Client
type Client interface {
	GetServiceBrokerByName(name string) (cfclient.ServiceBroker, error)
	CreateServiceBroker(csb cfclient.CreateServiceBrokerRequest) (cfclient.ServiceBroker, error)
	UpdateServiceBroker(guid string, csb cfclient.UpdateServiceBrokerRequest) (cfclient.ServiceBroker, error)
	DeleteServiceBroker(guid string) error
}
