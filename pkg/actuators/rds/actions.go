package rds

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Reconcile(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("reconcile", db.Name)

	// Based in the field, it creates or restores
	if db.Spec.DBSnapshotIdentifier != "" {
		log.Info("restoring")
		status, err = a.handleRestoreDatabase(db)
	} else {
		log.Info("creating")
		status, err = a.handleCreateDatabase(db)
	}

	return status, err
}

func (a *Actuator) Delete(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("delete", db.Name)

	//
	log.Info("deleting database")
	if !dryRunDelete {
		err := a.k8srds.DeleteDatabase(db)
		if err != nil {
			return databasesv1.NewStatus("ERROR", "ERROR"), err
		}
	}

	// If state is not deleting and arraived here, start deleting
	if !db.Is("DELETING") {
		return databasesv1.NewStatus("Deleting", "DELETING"), err
	}

	//
	log.Info("deleting svc")
	err = a.kubeClient.DeleteService(db.Namespace, db.Name)
	if err != nil {
		log.Error(err, "could not delete service")
		return databasesv1.NewStatus("ERROR", "ERROR"), err
	}

	log.Info("Deletion of database done")
	return databasesv1.NewStatus("Deleted", "DELETED"), err
}
