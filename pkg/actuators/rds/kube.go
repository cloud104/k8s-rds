package rds

import (
	"fmt"

	"github.com/k0kubun/pp"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// create an External named service object for Kubernetes
func (k *Kube) createServiceObj(s *v1.Service, namespace string, hostname string, internalname string) *v1.Service {
	s.Spec.Type = "ExternalName"
	s.Spec.ExternalName = hostname

	s.Name = internalname
	s.Annotations = map[string]string{"origin": "rds"}
	s.Namespace = namespace
	return s
}

// CreateService Creates or updates a service in Kubernetes with the new information
func (k *Kube) ReconcileService(namespace string, hostname string, internalName string) (err error) {
	// create a service in kubernetes that points to the AWS RDS instance
	serviceInterface := k.Client.CoreV1().Services(namespace)

	s, err := serviceInterface.Get(internalName, metav1.GetOptions{})

	s = k.createServiceObj(s, namespace, hostname, internalName)

	if err == nil {
		_, err = serviceInterface.Update(s)
	}

	if err != nil && metav1.Status(err.(k8s_errors.APIStatus).Status()).Code == 404 {
		_, err = serviceInterface.Create(s)
	}

	return err
}

func (k *Kube) DeleteService(namespace string, dbname string) error {
	serviceInterface := k.Client.CoreV1().Services(namespace)
	err := serviceInterface.Delete(dbname, &metav1.DeleteOptions{})
	if err != nil {
		// @TODO: Refactor, cuz ugly as fuck
		code := metav1.Status(err.(k8s_errors.APIStatus).Status()).Code
		if code == 404 {
			return nil
		}
		return errors.Wrap(err, fmt.Sprintf("delete of service %v failed in namespace %v", dbname, namespace))
	}
	return nil
}

func (k *Kube) GetSecret(namespace string, name string, key string) (string, error) {
	secret, err := k.Client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("unable to fetch secret %v", name))
	}
	password := secret.Data[key]
	return string(password), nil
}

func (k *Kube) hasService(namespace string, hostname string, internalname string) bool {
	serviceInterface := k.Client.CoreV1().Services(namespace)
	pp.Println(serviceInterface)
	return true
}
