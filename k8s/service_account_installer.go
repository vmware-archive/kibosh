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
