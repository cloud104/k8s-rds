package rds

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	k8srds "github.com/cloud104/kube-db/pkg/actuators/rds/client"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Delete(instance *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error, status databasesv1.RdsStatus) {
	log := a.log.WithValues("delete", instance.Name)

	log.Info("deleting database", "name", instance.Name)
	r := k8srds.AWS{RDS: a.rdsclient}
	if !dryRunDelete {
		err := r.DeleteDatabase(instance)
		if err != nil {
			return err, status
		}
	}

	err = a.kubeClient.DeleteService(instance.Namespace, instance.Name)
	if err != nil {
		log.Error(err, "could not delete service")
		return err, status
	}

	log.Info("Deletion of database done", "name", instance.Name)

	return nil, status
}
