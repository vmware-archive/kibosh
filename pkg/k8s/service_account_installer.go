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
	"code.cloudfoundry.org/lager"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	serviceAccountName = "tiller"
)

//go:generate counterfeiter ./ ServiceAccountInstaller
type ServiceAccountInstaller interface {
	Install() error
}

type serviceAccountInstaller struct {
	cluster Cluster
	logger  lager.Logger
}

func NewServiceAccountInstaller(cluster Cluster, logger lager.Logger) ServiceAccountInstaller {

	return &serviceAccountInstaller{
		cluster: cluster,
		logger:  logger,
	}
}

func (serviceAccountInstaller *serviceAccountInstaller) Install() error {
	err := serviceAccountInstaller.ensureAccount()
	if err != nil {
		return err
	}
	return serviceAccountInstaller.ensureRole()
}

func (serviceAccountInstaller *serviceAccountInstaller) ensureAccount() error {
	result, err := serviceAccountInstaller.cluster.ListServiceAccounts("kube-system", meta_v1.ListOptions{
		LabelSelector: "kibosh=tiller-service-account",
	})
	if err != nil {
		return err
	}

	if len(result.Items) < 1 {
		_, err = serviceAccountInstaller.cluster.CreateServiceAccount("kube-system", &api_v1.ServiceAccount{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   serviceAccountName,
				Labels: map[string]string{"kibosh": "tiller-service-account"},
			},
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (serviceAccountInstaller *serviceAccountInstaller) ensureRole() error {

	result, err := serviceAccountInstaller.cluster.ListClusterRoleBindings(meta_v1.ListOptions{
		LabelSelector: "kibosh=tiller-service-admin-binding",
	})
	if err != nil {
		return err
	}

	if len(result.Items) < 1 {
		// we should create
		_, err := serviceAccountInstaller.cluster.CreateClusterRoleBinding(&v1beta1.ClusterRoleBinding{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:   "tiller-cluster-admin",
				Labels: map[string]string{"kibosh": "tiller-service-admin-binding"},
			},
			RoleRef: v1beta1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []v1beta1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccountName,
					Namespace: "kube-system",
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
