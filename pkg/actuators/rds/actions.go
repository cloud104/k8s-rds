package rds

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Reconcile(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("reconcile", db.Name)
	log.Info("Starting reconcile")

	// Based in the field, it creates or restores
	if db.Spec.DBSnapshotIdentifier != "" {
		// log.Info("restoring")
		// status, err = a.handleRestoreDatabase(db)
	} else {
		log.Info("creating")
		status, err = a.handleCreateDatabase(db)
	}

	return status, err
}

func (a *Actuator) Delete(instance *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	log := a.log.WithValues("delete", instance.Name)

	//
	log.Info("deleting database")
	if !dryRunDelete {
		err := a.k8srds.DeleteDatabase(instance)
		if err != nil {
			return err
		}
	}

	//
	log.Info("deleting svc")
	err = a.kubeClient.DeleteService(instance.Namespace, instance.Name)
	if err != nil {
		log.Error(err, "could not delete service")
		return err
	}

	log.Info("Deletion of database done")
	return err
}
