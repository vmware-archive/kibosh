package test

//go:generate counterfeiter -o fake_k8s_interface.go --fake-name FakeK8sInterface k8s.io/client-go/kubernetes.Interface
