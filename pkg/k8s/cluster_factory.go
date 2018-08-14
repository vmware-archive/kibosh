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

package k8s

import (
	"github.com/cf-platform-eng/kibosh/pkg/config"
)

//go:generate counterfeiter ./ ClusterFactory
type ClusterFactory interface {
	DefaultCluster() (Cluster, error)
	GetCluster(creds *config.ClusterCredentials) (Cluster, error)
}

func NewClusterFactory(clusterCredentials config.ClusterCredentials) ClusterFactory {
	return &clusterFactory{clusterCredentials}
}

type clusterFactory struct {
	clusterCredentials config.ClusterCredentials
}

func (cf clusterFactory) DefaultCluster() (Cluster, error) {
	return NewCluster(&cf.clusterCredentials)
}

func (cf clusterFactory) GetCluster(creds *config.ClusterCredentials) (Cluster, error) {
	return NewCluster(creds)
}
